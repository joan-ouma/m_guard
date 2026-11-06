package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"motoguard-backend/internal/sms"
)

const (
	// lostBikeSMSMaxLength keeps the text to one GSM 03.38 segment.
	lostBikeSMSMaxLength = 160
	lostBikeSMSTimeout   = 20 * time.Second
	unknownLocation      = "unknown"
)

// notifyContactsBikeLost texts every saved contact that the rider's bike has been reported lost.
// It runs in the background after the report is saved, so a failure here never affects the report.
func (s *RecoveryService) notifyContactsBikeLost(userID, bikeID, reportedLocation string) {
	ctx, cancel := context.WithTimeout(context.Background(), lostBikeSMSTimeout)
	defer cancel()

	details, err := s.repo.LostBikeDetails(ctx, userID, bikeID)
	if err != nil {
		slog.Error("lost bike SMS: could not load bike details", "user_id", userID, "bike_id", bikeID, "error", err)
		return
	}
	if strings.TrimSpace(details.Plate) == "" {
		slog.Warn("lost bike SMS skipped: bike has no registration number", "user_id", userID, "bike_id", bikeID)
		return
	}

	contacts, err := s.repo.ListContacts(ctx, userID)
	if err != nil {
		slog.Error("lost bike SMS: could not load contacts", "user_id", userID, "error", err)
		return
	}

	recipients := make([]string, 0, len(contacts))
	seen := map[string]bool{}
	for _, c := range contacts {
		number, ok := sms.NormalizePhone(c.Phone)
		if !ok {
			slog.Warn("lost bike SMS skipped contact: phone number is not valid", "user_id", userID, "contact_id", c.ID)
			continue
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		recipients = append(recipients, number)
	}
	if len(recipients) == 0 {
		slog.Info("lost bike SMS skipped: rider has no contacts with a valid phone number", "user_id", userID, "bike_id", bikeID)
		return
	}

	address := strings.TrimSpace(details.LastAddress)
	if address == "" {
		address = strings.TrimSpace(reportedLocation)
	}

	message := lostBikeMessage(details.OwnerName, details.Plate, address)
	results, err := s.sms.Send(ctx, recipients, message)
	if err != nil {
		slog.Error("lost bike SMS failed", "user_id", userID, "bike_id", bikeID, "recipients", len(recipients), "error", err)
		return
	}

	accepted := 0
	for _, r := range results {
		if r.IsAccepted() {
			accepted++
			continue
		}
		slog.Warn("lost bike SMS rejected for recipient", "user_id", userID, "bike_id", bikeID, "status", r.Status, "status_code", r.StatusCode)
	}
	slog.Info("lost bike SMS dispatched", "user_id", userID, "bike_id", bikeID, "recipients", len(recipients), "accepted", accepted)
}

// lostBikeMessage builds: "<NAME> has lost their bike <PLATE>. Last location is <ADDRESS>."
// The address is shortened so the whole message fits in one SMS segment.
func lostBikeMessage(ownerName, plate, address string) string {
	name := strings.TrimSpace(ownerName)
	if name == "" {
		name = "A MotorGuard rider"
	}
	if address == "" {
		address = unknownLocation
	}

	prefix := fmt.Sprintf("%s has lost their bike %s. Last location is ", name, strings.TrimSpace(plate))
	suffix := "."
	room := lostBikeSMSMaxLength - len(prefix) - len(suffix)

	runes := []rune(address)
	if len(runes) > room {
		if room > 3 {
			address = string(runes[:room-3]) + "..."
		} else if room > 0 {
			address = string(runes[:room])
		}
	}
	return prefix + address + suffix
}
