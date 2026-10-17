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

	"motoguard-backend/internal/africastalking"
	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/notification"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/sse"
)

type USSDService struct {
	repo         *repository.USSDRepository
	atClient     africastalking.Client
	cmdPub       CommandPublisher
	sseBroker    *sse.Broker
	notifService *notification.Service
}

func NewUSSDService(
	repo *repository.USSDRepository,
	atClient africastalking.Client,
	cmdPub CommandPublisher,
	sseBroker *sse.Broker,
	notifService *notification.Service,
) *USSDService {
	return &USSDService{
		repo:         repo,
		atClient:     atClient,
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

// HandleIncomingSMS processes inbound SMS commands from riders (2-Way SMS).
func (s *USSDService) HandleIncomingSMS(ctx context.Context, req domain.SMSIncomingRequest) string {
	// SMS processing deactivated for now
	return "SMS service is currently deactivated."
}

func (s *USSDService) handleSMSRegister(ctx context.Context, phone string, args []string) string {
	if len(args) < 3 {
		return "MGuard: To register motorcycle via SMS, send: REG <PLATE> <IMEI> <PIN>. Example: REG KMDF123X 862000000000001 1234"
	}

	plate := strings.ToUpper(args[0])
	imei := strings.TrimSpace(args[1])
	pin := strings.TrimSpace(args[2])

	if len(pin) != 4 || !isAllDigits(pin) {
		return "MGuard: ❌ PIN must be exactly 4 numeric digits."
	}

	bike, err := s.repo.RegisterBike(ctx, repository.RegisterBikeParams{
		Phone:              phone,
		RegistrationNumber: plate,
		IMEI:               imei,
		Make:               "Bajaj",
		Model:              "Boxer",
	})
	if err != nil {
		return fmt.Sprintf("MGuard: ❌ Registration failed: %s", err.Error())
	}

	_ = s.repo.SaveRiderPIN(ctx, phone, HashPIN(phone, pin))

	return fmt.Sprintf("MGuard: ✅ Motorcycle %s registered successfully with IMEI %s! PIN is set. Send STATUS, LOCK %s, or UNLOCK %s to manage your bike.", bike.RegistrationNumber, imei, pin, pin)
}

func (s *USSDService) handleSMSStatus(bikes []domain.RiderBikeInfo, args []string) string {
	var target = bikes[0]
	if len(args) > 0 {
		regArg := strings.ToUpper(strings.ReplaceAll(args[0], " ", ""))
		for _, b := range bikes {
			cleanReg := strings.ToUpper(strings.ReplaceAll(b.RegistrationNumber, " ", ""))
			if cleanReg == regArg {
				target = b
				break
			}
		}
	}

	ign := "OFF"
	if target.Ignition {
		ign = "ON"
	}
	engine := "UNLOCKED"
	if target.IsImmobilized {
		engine = "LOCKED"
	}

	loc := target.CurrentAddress
	if loc == "" || loc == "Location pending update" {
		loc = fmt.Sprintf("Lat: %.4f, Lng: %.4f", target.Latitude, target.Longitude)
	}

	return fmt.Sprintf("MGuard [%s]: Status: %s | Ign: %s | Speed: %.0fkm/h | Batt: %d%% | Engine: %s | Loc: %s",
		target.RegistrationNumber, target.Status, ign, target.SpeedKMH, target.BatteryPercent, engine, loc)
}

func (s *USSDService) handleSMSLock(ctx context.Context, phone string, bikes []domain.RiderBikeInfo, args []string, lock bool) string {
	actionName := "LOCK"
	if !lock {
		actionName = "UNLOCK"
	}

	if len(args) == 0 {
		return fmt.Sprintf("MGuard: Please include your 4-digit PIN. Example: %s 1234", actionName)
	}

	var pin string
	var target = bikes[0]

	if len(args) == 1 {
		pin = args[0]
	} else {
		// Example: LOCK KME123A 1234
		regArg := strings.ToUpper(strings.ReplaceAll(args[0], " ", ""))
		pin = args[1]
		for _, b := range bikes {
			cleanReg := strings.ToUpper(strings.ReplaceAll(b.RegistrationNumber, " ", ""))
			if cleanReg == regArg {
				target = b
				break
			}
		}
	}

	valid, err := s.VerifyPIN(ctx, phone, pin)
	if err != nil {
		return fmt.Sprintf("MGuard: %s", err.Error())
	}
	if !valid {
		return fmt.Sprintf("MGuard: ❌ Incorrect PIN. Engine %s aborted.", strings.ToLower(actionName))
	}

	imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, target.ID, lock, "SMS")
	if err != nil {
		return "MGuard: System busy. Unable to update engine state right now."
	}

	mqttAction := "IMMOBILIZE"
	if !lock {
		mqttAction = "REACTIVATE"
	}
	if s.cmdPub != nil && imei != "" {
		_ = s.cmdPub.PublishCommand(imei, mqttAction)
	}

	if s.sseBroker != nil {
		s.sseBroker.BroadcastAlert(userID, map[string]any{
			"event_type":  mqttAction,
			"bike_id":     target.ID,
			"title":       fmt.Sprintf("Engine %sed via SMS", strings.Title(strings.ToLower(actionName))),
			"activity":    fmt.Sprintf("Command received from %s", phone),
			"recorded_at": time.Now().UTC(),
		})
	}

	if lock {
		return fmt.Sprintf("🔒 MGuard: Engine for %s is now LOCKED. Starter disabled.", target.RegistrationNumber)
	}
	return fmt.Sprintf("🔓 MGuard: Engine for %s is now UNLOCKED. Ready to ride.", target.RegistrationNumber)
}

func (s *USSDService) handleSMSSOS(ctx context.Context, phone string, bikes []domain.RiderBikeInfo) string {
	target := bikes[0]
	details := fmt.Sprintf("Emergency SOS triggered by rider (%s) via SMS", phone)
	_ = s.repo.RecordEmergencySOS(ctx, target.ID, target.UserID, target.Latitude, target.Longitude, details)

	// Alert owner via SMS
	if target.OwnerPhone != "" {
		ownerMsg := fmt.Sprintf("🚨 MGUARD SOS ALERT! Rider %s (%s) sent SOS for %s. Location: %s. Maps: https://maps.google.com/?q=%.6f,%.6f",
			target.RiderName, phone, target.RegistrationNumber, target.CurrentAddress, target.Latitude, target.Longitude)
		normOwner, _ := repository.NormalizePhoneNumber(target.OwnerPhone)
		go func() {
			_, _ = s.atClient.SendSMS(context.Background(), []string{normOwner}, ownerMsg, "")
		}()
	}

	return fmt.Sprintf("🚨 MGuard: Emergency SOS received for %s! Fleet manager notified with your location (%s).",
		target.RegistrationNumber, target.CurrentAddress)
}

func (s *USSDService) handleSMSChangePIN(ctx context.Context, phone string, args []string) string {
	if len(args) < 2 {
		return "MGuard: To change PIN via SMS, send: PIN <CURRENT_PIN> <NEW_PIN>. Example: PIN 1234 5678"
	}

	currentPIN := args[0]
	newPIN := args[1]

	valid, err := s.VerifyPIN(ctx, phone, currentPIN)
	if err != nil {
		return fmt.Sprintf("MGuard: %s", err.Error())
	}
	if !valid {
		return "MGuard: ❌ Current PIN is incorrect. PIN change failed."
	}

	if len(newPIN) != 4 || !isAllDigits(newPIN) {
		return "MGuard: ❌ New PIN must be exactly 4 numeric digits."
	}

	hashed := HashPIN(phone, newPIN)
	if err := s.repo.SaveRiderPIN(ctx, phone, hashed); err != nil {
		return "MGuard: Error updating PIN. Please try again."
	}

	return "MGuard: ✅ Your security PIN has been updated successfully!"
}

func (s *USSDService) smsHelpMessage() string {
	return "MGuard SMS Commands:\n- STATUS: Check motorcycle\n- LOCK <PIN>: Lock engine\n- UNLOCK <PIN>: Unlock engine\n- SOS: Send emergency alert\n- REG <PLATE> <IMEI> <PIN>: Register motorcycle\n- PIN <OLD> <NEW>: Change PIN"
}

func isAllDigits(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
