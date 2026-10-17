package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/notification"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/sse"
)

type USSDService struct {
	repo         *repository.USSDRepository
	cmdPub       CommandPublisher
	sseBroker    *sse.Broker
	notifService *notification.Service
}

func NewUSSDService(
	repo *repository.USSDRepository,
	cmdPub CommandPublisher,
	sseBroker *sse.Broker,
	notifService *notification.Service,
) *USSDService {
	return &USSDService{
		repo:         repo,
		cmdPub:       cmdPub,
		sseBroker:    sseBroker,
		notifService: notifService,
	}
}

// HashPIN securely hashes a 4-digit PIN with SHA-256.
func HashPIN(phone, pin string) string {
	normalized, _ := repository.NormalizePhoneNumber(phone)
	salt := "mguard_ussd_salt_" + normalized
	h := sha256.Sum256([]byte(salt + ":" + strings.TrimSpace(pin)))
	return hex.EncodeToString(h[:])
}

// VerifyPIN checks whether the entered PIN matches the stored hash (or default 1234).
func (s *USSDService) VerifyPIN(ctx context.Context, phone, enteredPIN string) (bool, error) {
	enteredPIN = strings.TrimSpace(enteredPIN)
	if len(enteredPIN) != 4 {
		return false, nil
	}

	record, err := s.repo.GetRiderPIN(ctx, phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Default PIN is 1234 if rider hasn't set custom PIN yet
			return enteredPIN == "1234", nil
		}
		return false, err
	}

	if record.LockedUntil != nil && record.LockedUntil.After(time.Now().UTC()) {
		return false, fmt.Errorf("account locked until %s due to repeated incorrect PIN attempts", record.LockedUntil.Format("15:04"))
	}

	expectedHash := HashPIN(phone, enteredPIN)
	if record.PINHash == expectedHash {
		_ = s.repo.ResetFailedPINAttempts(ctx, phone)
		return true, nil
	}

	// Increment failed attempts
	failed := record.FailedAttempts + 1
	var lockedUntil *time.Time
	if failed >= 5 {
		lockTime := time.Now().UTC().Add(15 * time.Minute)
		lockedUntil = &lockTime
	}
	_ = s.repo.RecordFailedPINAttempt(ctx, phone, failed, lockedUntil)

	return false, nil
}

// parseUSSDStack resolves Africa's Talking asterisk-delimited inputs into a navigation stack,
// allowing '0' to reliably return to the previous menu at any level.
func parseUSSDStack(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, "*")
	var stack []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "0" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		} else {
			stack = append(stack, part)
		}
	}
	return stack
}

// HandleUSSD processes Africa's Talking USSD callback and returns the session response.
func (s *USSDService) HandleUSSD(ctx context.Context, req domain.USSDRequest) string {
	phone := req.PhoneNumber

	// Lookup bikes registered to this caller's phone number
	bikes, err := s.repo.GetBikesByPhone(ctx, phone)
	if err != nil {
		slog.Error("USSD lookup bikes error", "phone", phone, "error", err)
		return "END System temporarily unavailable. Please try again shortly."
	}

	// 1. If phone number is not registered on the platform, do NOT allow procession
	if len(bikes) == 0 {
		return "END Phone number not registered on M-Guard. Please contact your administrator."
	}

	// 2. Parse multi-level USSD inputs with stack back-navigation
	steps := parseUSSDStack(strings.TrimSpace(req.Text))

	// Single bike flow vs Multi-bike flow
	if len(bikes) == 1 {
		return s.handleSingleBikeMenu(ctx, phone, bikes[0], steps)
	}

	return s.handleMultiBikeMenu(ctx, phone, bikes, steps)
}

func (s *USSDService) handleSingleBikeMenu(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return "CON Welcome to M-Guard\n1. Find bike\n2. Immobilize bike\n3. Reactivate bike\n4. Emergency phones"
	}

	choice := steps[0]
	switch choice {
	case "1":
		return s.handleFindBike(bike)
	case "2":
		return s.handleImmobilizeBike(ctx, phone, bike, steps[1:])
	case "3":
		return s.handleReactivateBike(ctx, phone, bike, steps[1:])
	case "4":
		return s.handleEmergencyPhones()
	default:
		return "END Invalid option. Session ended."
	}
}

func (s *USSDService) handleMultiBikeMenu(ctx context.Context, phone string, bikes []domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		var sb strings.Builder
		sb.WriteString("CON Welcome to M-Guard\nSelect bike:\n")
		for i, b := range bikes {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, b.RegistrationNumber))
		}
		sb.WriteString("0. Exit")
		return sb.String()
	}

	idxStr := steps[0]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 1 || idx > len(bikes) {
		return "END Invalid selection. Session ended."
	}

	selectedBike := bikes[idx-1]
	return s.handleSingleBikeMenu(ctx, phone, selectedBike, steps[1:])
}

// 1. Find bike
func (s *USSDService) handleFindBike(bike domain.RiderBikeInfo) string {
	return s.renderBikeStatus(bike)
}

func (s *USSDService) renderBikeStatus(bike domain.RiderBikeInfo) string {
	loc := bike.CurrentAddress
	if loc == "" || loc == "Location pending update" {
		if bike.SubCounty != "" || bike.County != "" {
			loc = fmt.Sprintf("%s, %s", bike.SubCounty, bike.County)
		} else {
			loc = "GPS active"
		}
	}

	status := bike.Status
	if status == "" {
		if bike.Ignition {
			status = "RUNNING"
		} else {
			status = "STOPPED"
		}
	}

	return fmt.Sprintf("CON Bike %s\nStatus: %s\nBattery: %d%%\nLocation: %s\n0. Back",
		bike.RegistrationNumber, status, bike.BatteryPercent, loc)
}

// 2. Immobilize bike
func (s *USSDService) handleImmobilizeBike(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if bike.IsImmobilized {
		return fmt.Sprintf("END Bike %s is already immobilized.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON Turn off engine for %s?\n1. Yes, shut down\n0. Back", bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, true, "USSD (*384#)")
		if err != nil {
			slog.Error("Failed to immobilize bike via USSD", "bike_id", bike.ID, "error", err)
			return "END Unable to immobilize bike at this time. Please try again."
		}

		if s.cmdPub != nil && imei != "" {
			_ = s.cmdPub.PublishCommand(imei, "IMMOBILIZE")
		}

		if s.sseBroker != nil {
			s.sseBroker.BroadcastAlert(userID, map[string]any{
				"event_type":  "IMMOBILIZED",
				"bike_id":     bike.ID,
				"title":       "🔒 Engine Immobilized via USSD",
				"activity":    fmt.Sprintf("Rider on %s engaged engine immobilization remotely.", phone),
				"recorded_at": time.Now().UTC(),
			})
		}

		// SMS notification deactivated for now

		return "END Engine immobilized. Starter has been disabled safely."
	}

	return "END Invalid option. Session ended."
}

// 3. Reactivate bike
func (s *USSDService) handleReactivateBike(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if !bike.IsImmobilized {
		return fmt.Sprintf("END Bike %s is already active.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON Restore engine for %s?\n1. Yes, reactivate\n0. Back", bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, false, "USSD (*384#)")
		if err != nil {
			slog.Error("Failed to reactivate bike via USSD", "bike_id", bike.ID, "error", err)
			return "END Unable to reactivate bike at this time. Please try again."
		}

		if s.cmdPub != nil && imei != "" {
			_ = s.cmdPub.PublishCommand(imei, "REACTIVATE")
		}

		if s.sseBroker != nil {
			s.sseBroker.BroadcastAlert(userID, map[string]any{
				"event_type":  "REACTIVATED",
				"bike_id":     bike.ID,
				"title":       "🔓 Engine Reactivated via USSD",
				"activity":    fmt.Sprintf("Rider on %s restored engine remotely.", phone),
				"recorded_at": time.Now().UTC(),
			})
		}

		// SMS notification deactivated for now

		return "END Engine reactivated. Bike is ready to ride."
	}

	return "END Invalid option. Session ended."
}

// 4. Emergency phones
func (s *USSDService) handleEmergencyPhones() string {
	return "CON Emergency Helplines:\nControl Room: 0700 123 456\nPolice: 999 / 112\n0. Back"
}



func isAllDigits(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
