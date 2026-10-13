package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"motoguard-backend/internal/domain"
)

type Client struct {
	ID        string
	UserID    string
	Send      chan []byte
	closeOnce sync.Once
}

func NewClient(id string, userID string, bufferSize int) *Client {
	if bufferSize <= 0 {
		bufferSize = 128
	}
	return &Client{
		ID:     id,
		UserID: userID,
		Send:   make(chan []byte, bufferSize),
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.Send)
	})
}

// Broker is a high-throughput, low-latency in-process event fanout engine
// designed for immediate telemetry dispatch with zero external infrastructure dependencies.
type Broker struct {
	mu          sync.RWMutex
	clients     map[*Client]struct{}
	userClients map[string]map[*Client]struct{}
	closeOnce   sync.Once
}

func NewBroker() *Broker {
	return &Broker{
		clients:     make(map[*Client]struct{}),
		userClients: make(map[string]map[*Client]struct{}),
	}
}

func (b *Broker) Register(c *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.clients[c] = struct{}{}
	if b.userClients[c.UserID] == nil {
		b.userClients[c.UserID] = make(map[*Client]struct{})
	}
	b.userClients[c.UserID][c] = struct{}{}

	slog.Info("SSE client connected", "client_id", c.ID, "user_id", c.UserID, "total_subscribers", len(b.clients))
}

func (b *Broker) Unregister(c *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.clients[c]; !exists {
		return
	}

	delete(b.clients, c)
	if group, ok := b.userClients[c.UserID]; ok {
		delete(group, c)
		if len(group) == 0 {
			delete(b.userClients, c.UserID)
		}
	}
	c.Close()

	slog.Info("SSE client disconnected", "client_id", c.ID, "user_id", c.UserID, "total_subscribers", len(b.clients))
}

// BroadcastEvent dispatches events immediately upon arrival from sensors with zero intermediary hops.
func (b *Broker) BroadcastEvent(userID string, eventType string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("SSE serialization error", "event_type", eventType, "error", err)
		return
	}

	frame := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload))

	b.mu.RLock()
	var targets []*Client
	if userID != "" {
		if group, ok := b.userClients[userID]; ok {
			targets = make([]*Client, 0, len(group))
			for c := range group {
				targets = append(targets, c)
			}
		}
	} else {
		targets = make([]*Client, 0, len(b.clients))
		for c := range b.clients {
			targets = append(targets, c)
		}
	}
	b.mu.RUnlock()

	var slowClients []*Client
	for _, c := range targets {
		select {
		case c.Send <- frame:
		default:
			// Client's network buffer is full; mark for eviction to prevent resource leaks
			slowClients = append(slowClients, c)
		}
	}

	if len(slowClients) > 0 {
		go func(evicted []*Client) {
			for _, c := range evicted {
				slog.Warn("Evicting slow or dead SSE subscriber", "client_id", c.ID, "user_id", c.UserID)
				b.Unregister(c)
			}
		}(slowClients)
	}
}

func (b *Broker) BroadcastBikeUpdate(userID string, bike domain.BikeDTO) {
	b.BroadcastEvent(userID, "bike_update", bike)
}

func (b *Broker) BroadcastAlert(userID string, alert any) {
	b.BroadcastEvent(userID, "alert", alert)
}

func (b *Broker) Stop() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		for c := range b.clients {
			c.Close()
		}
		b.clients = make(map[*Client]struct{})
		b.userClients = make(map[string]map[*Client]struct{})
	})
}
