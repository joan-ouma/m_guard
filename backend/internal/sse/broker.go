package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"motoguard-backend/internal/domain"
)

type Client struct {
	ID     string
	UserID string
	Send   chan []byte
}

type Broker struct {
	mu          sync.RWMutex
	clients     map[*Client]bool
	userClients map[string]map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	quit        chan struct{}
}

func NewBroker() *Broker {
	b := &Broker{
		clients:     make(map[*Client]bool),
		userClients: make(map[string]map[*Client]bool),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		quit:        make(chan struct{}),
	}
	go b.run()
	return b
}

func (b *Broker) run() {
	// Send lightweight comment keepalive ping every 15 seconds to prevent mobile drops and proxy timeouts
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.quit:
			b.mu.Lock()
			for c := range b.clients {
				close(c.Send)
			}
			b.clients = make(map[*Client]bool)
			b.userClients = make(map[string]map[*Client]bool)
			b.mu.Unlock()
			return

		case c := <-b.register:
			b.mu.Lock()
			b.clients[c] = true
			if b.userClients[c.UserID] == nil {
				b.userClients[c.UserID] = make(map[*Client]bool)
			}
			b.userClients[c.UserID][c] = true
			slog.Info("SSE client registered", "client_id", c.ID, "user_id", c.UserID, "total_clients", len(b.clients))
			b.mu.Unlock()

		case c := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[c]; ok {
				delete(b.clients, c)
				if b.userClients[c.UserID] != nil {
					delete(b.userClients[c.UserID], c)
					if len(b.userClients[c.UserID]) == 0 {
						delete(b.userClients, c.UserID)
					}
				}
				close(c.Send)
				slog.Info("SSE client unregistered", "client_id", c.ID, "user_id", c.UserID, "total_clients", len(b.clients))
			}
			b.mu.Unlock()

		case <-ticker.C:
			pingMsg := []byte(": keepalive\n\n")
			b.mu.RLock()
			for c := range b.clients {
				select {
				case c.Send <- pingMsg:
				default:
				}
			}
			b.mu.RUnlock()
		}
	}
}

func (b *Broker) Register(c *Client) {
	b.register <- c
}

func (b *Broker) Unregister(c *Client) {
	b.unregister <- c
}

func (b *Broker) Stop() {
	close(b.quit)
}

func (b *Broker) BroadcastBikeUpdate(userID string, bike domain.BikeDTO) {
	b.BroadcastEvent(userID, "bike_update", bike)
}

func (b *Broker) BroadcastAlert(userID string, alert any) {
	b.BroadcastEvent(userID, "alert", alert)
}

func (b *Broker) BroadcastEvent(userID string, eventType string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("SSE marshal error", "error", err)
		return
	}

	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, dataBytes))

	b.mu.RLock()
	defer b.mu.RUnlock()

	if userID != "" {
		if clients, ok := b.userClients[userID]; ok {
			for c := range clients {
				select {
				case c.Send <- msg:
				default:
					slog.Warn("SSE client buffer full, skipping frame", "client_id", c.ID, "user_id", userID)
				}
			}
		}
		return
	}

	for c := range b.clients {
		select {
		case c.Send <- msg:
		default:
			slog.Warn("SSE client buffer full, skipping frame", "client_id", c.ID)
		}
	}
}
