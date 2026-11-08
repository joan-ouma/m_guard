package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/sms"
)

const (
	// smsSessionIdle is how long a rider's SMS menu stays open without a reply.
	smsSessionIdle = 5 * time.Minute
	// smsMaxInput caps one rider reply so a stored conversation stays small.
	smsMaxInput = 40
	// SMSProjectKeyword is M-Guard's keyword on the shared SMS gateway. Riders text "kamilimu ignition <command>".
	SMSProjectKeyword = "ignition"
)

// smsStartWords open the menu, or restart it when one is already open.
var smsStartWords = map[string]bool{
	"menu":  true,
	"hi":    true,
	"hello": true,
	"start": true,
}

// SetSMSSender enables SMS replies for the menu conversation. Without it, HandleSMS returns an error.
func (s *USSDService) SetSMSSender(sender sms.Sender) {
	s.smsSender = sender
}

// SetSMSRouting sets the shared gateway routing keyword (for example "kamilimu"). The project
// keyword is SMSProjectKeyword.
func (s *USSDService) SetSMSRouting(routing string) {
	s.smsRouting = strings.ToLower(strings.TrimSpace(routing))
}

// routeSMSText removes the gateway routing prefix from an inbound text. forUs is false when the
// text is routed to another project. A text that is only the prefix counts as "menu".
// The gateway may strip the prefix before it calls us, so bare text is accepted too.
func (s *USSDService) routeSMSText(text string) (rest string, forUs bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", true
	}
	first := strings.ToLower(fields[0])

	if s.smsRouting != "" && first == s.smsRouting {
		if len(fields) < 2 || strings.ToLower(fields[1]) != SMSProjectKeyword {
			return "", false
		}
		rest = strings.Join(fields[2:], " ")
		if rest == "" {
			rest = "menu"
		}
		return rest, true
	}

	if first == SMSProjectKeyword {
		rest = strings.Join(fields[1:], " ")
		if rest == "" {
			rest = "menu"
		}
		return rest, true
	}

	return text, true
}

// HandleSMS continues a rider's menu conversation over SMS and texts the reply back.
// It uses the same menu tree as USSD: each text is one menu reply, appended to the
// conversation's steps, and the reply starts with the same CON/END markers.
func (s *USSDService) HandleSMS(ctx context.Context, req domain.SMSIncomingRequest) error {
	to, ok := sms.NormalizePhone(req.From)
	if !ok {
		return fmt.Errorf("sms sender number is not valid: %q", req.From)
	}
	if s.smsSender == nil {
		return errors.New("sms sender is not configured")
	}

	reply, err := s.smsReply(ctx, req.From, req.Text)
	if err != nil {
		return err
	}
	if reply == "" {
		return nil
	}

	results, err := s.smsSender.Send(ctx, []string{to}, reply)
	if err != nil {
		return fmt.Errorf("send sms menu reply: %w", err)
	}
	if len(results) > 0 && !results[0].IsAccepted() {
		return fmt.Errorf("sms menu reply rejected: status %d %s", results[0].StatusCode, results[0].Status)
	}
	return nil
}

// smsReply works out the text to send back for one inbound rider text. It returns an empty
// string when nothing should be sent.
func (s *USSDService) smsReply(ctx context.Context, phone, text string) (string, error) {
	rest, forUs := s.routeSMSText(text)
	if !forUs {
		return "", nil
	}
	input := cleanSMSInput(rest)
	if input == "" {
		return "", nil
	}

	// Like dialling USSD: with no open menu, any text shows the numbered welcome menu.
	// Inside an open menu, the text is the rider's reply to it.
	steps := ""
	if !smsStartWords[strings.ToLower(input)] {
		openSteps, open, err := s.openSMSSteps(ctx, phone)
		if err != nil {
			return "", err
		}
		if open {
			steps = joinSMSSteps(openSteps, input)
		}
	}

	body := s.HandleUSSD(ctx, domain.USSDRequest{PhoneNumber: phone, Text: steps})
	reply, open := splitSMSReply(body)

	if open {
		if err := s.repo.SaveSMSSession(ctx, phone, steps); err != nil {
			slog.Error("SMS menu session save failed", "phone", phone, "error", err)
			return "", err
		}
	} else if err := s.repo.DeleteSMSSession(ctx, phone); err != nil {
		slog.Warn("SMS menu session close failed", "phone", phone, "error", err)
	}

	return reply, nil
}

// openSMSSteps returns the stored steps of a rider's open SMS menu. It reports false when there
// is no menu or the rider has been idle longer than smsSessionIdle.
func (s *USSDService) openSMSSteps(ctx context.Context, phone string) (string, bool, error) {
	steps, updatedAt, err := s.repo.GetSMSSession(ctx, phone)
	if errors.Is(err, repository.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if time.Since(updatedAt) > smsSessionIdle {
		return "", false, nil
	}
	return steps, true, nil
}

// cleanSMSInput trims a rider's text and removes the characters the USSD menu uses as
// separators, so a reply cannot push the menu to a different level.
func cleanSMSInput(text string) string {
	cleaned := strings.NewReplacer("*", " ", "#", " ").Replace(text)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if r := []rune(cleaned); len(r) > smsMaxInput {
		cleaned = string(r[:smsMaxInput])
	}
	return cleaned
}

// joinSMSSteps adds one rider reply to the steps of an open menu, in USSD form.
func joinSMSSteps(steps, input string) string {
	if steps == "" {
		return input
	}
	return steps + "*" + input
}

// splitSMSReply strips the CON/END marker from a USSD menu response. open is true when the
// menu expects another reply (CON).
func splitSMSReply(resp string) (body string, open bool) {
	switch {
	case strings.HasPrefix(resp, "CON "):
		return strings.TrimPrefix(resp, "CON "), true
	case strings.HasPrefix(resp, "END "):
		return strings.TrimPrefix(resp, "END "), false
	default:
		return resp, false
	}
}
