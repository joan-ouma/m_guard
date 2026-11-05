package service

import (
	"context"
	"fmt"
	"strings"

	"motoguard-backend/internal/domain"
)

type SMSReplySender interface {
	SendSMS(ctx context.Context, to, message string) error
}

type SMSService struct {
	sender         SMSReplySender
	routingKeyword string
	projectKeyword string
}

func NewSMSService(sender SMSReplySender, routingKeyword, projectKeyword string) *SMSService {
	return &SMSService{
		sender:         sender,
		routingKeyword: strings.ToLower(strings.TrimSpace(routingKeyword)),
		projectKeyword: strings.ToLower(strings.TrimSpace(projectKeyword)),
	}
}

func (s *SMSService) HandleIncoming(ctx context.Context, req domain.SMSIncomingRequest) error {
	to := strings.TrimSpace(req.From)
	if to == "" {
		return fmt.Errorf("inbound SMS sender number is required")
	}
	if s.sender == nil {
		return fmt.Errorf("SMS reply provider is not configured")
	}

	message, shouldReply := s.replyFor(req.Text)
	if !shouldReply {
		return nil
	}
	if err := s.sender.SendSMS(ctx, to, message); err != nil {
		return fmt.Errorf("send SMS reply: %w", err)
	}
	return nil
}

func (s *SMSService) replyFor(text string) (string, bool) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(text)))

	if len(words) > 0 && words[0] == s.routingKeyword && s.routingKeyword != "" {
		words = words[1:]
		if len(words) == 0 || words[0] != s.projectKeyword {
			return "", false
		}
		words = words[1:]
	} else if len(words) > 0 && words[0] == s.projectKeyword && s.projectKeyword != "" {
		words = words[1:]
	}

	if len(words) == 0 || words[0] == "help" {
		return "M-Guard: Text POLICE <town/area> (e.g. POLICE Rongai) for a station directory match. Emergency: 999/112.", true
	}

	if words[0] != "police" {
		return "M-Guard: Unknown command. Text POLICE <town/area> or HELP. Emergency: 999/112.", true
	}
	if len(words) == 1 {
		return "M-Guard: Include your town/area, e.g. POLICE Rongai. Emergency: 999/112.", true
	}

	area := strings.Join(words[1:], " ")
	matches := findPoliceStations(area)
	if len(matches) == 0 {
		return fmt.Sprintf("M-Guard: No directory match for %s. Emergency: 999/112; DCI: 0800 722 203.", area), true
	}

	station := matches[0]
	return fmt.Sprintf("M-Guard police directory match for %s: %s, %s. Emergency: 999/112.", area, station.Name, station.Phone), true
}
