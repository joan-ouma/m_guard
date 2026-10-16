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
	salt := "motoguard_ussd_salt_" + normalized
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

// HandleUSSD processes Africa's Talking USSD callback and returns the session response.
func (s *USSDService) HandleUSSD(ctx context.Context, req domain.USSDRequest) string {
	phone := req.PhoneNumber
	rawText := strings.TrimSpace(req.Text)

	bikes, err := s.repo.GetBikesByPhone(ctx, phone)
	if err != nil {
		slog.Error("USSD lookup bikes error", "phone", phone, "error", err)
		return "END System temporarily unavailable. Please try again shortly."
	}

	if len(bikes) == 0 {
		normalized, _ := repository.NormalizePhoneNumber(phone)
		return fmt.Sprintf("END Welcome to MotoGuard.\nPhone number (%s) is not linked to any active motorcycle.\nPlease contact your fleet manager to assign your phone.", normalized)
	}

	// Parse multi-level USSD input separated by "*"
	var steps []string
	if rawText != "" {
		steps = strings.Split(rawText, "*")
	}

	// Single bike flow vs Multi-bike flow
	if len(bikes) == 1 {
		return s.handleSingleBikeMenu(ctx, phone, bikes[0], steps)
	}

	return s.handleMultiBikeMenu(ctx, phone, bikes, steps)
}

func (s *USSDService) handleSingleBikeMenu(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return fmt.Sprintf("CON MotoGuard: %s (%s)\n1. Check Status & Location\n2. Lock Bike (Immobilize)\n3. Unlock Bike (Mobilize)\n4. Emergency SOS\n5. Change PIN\n0. Exit",
			bike.RegistrationNumber, bike.Make)
	}

	choice := strings.TrimSpace(steps[0])
	switch choice {
	case "1":
		return s.renderBikeStatus(bike)
	case "2":
		return s.handleLockFlow(ctx, phone, bike, steps[1:])
	case "3":
		return s.handleUnlockFlow(ctx, phone, bike, steps[1:])
	case "4":
		return s.handleSOSFlow(ctx, phone, bike, steps[1:])
	case "5":
		return s.handleChangePINFlow(ctx, phone, steps[1:])
	case "0":
		return "END Thank you for using MotoGuard Fleet Security."
	default:
		return "END Invalid option selected. Please dial again."
	}
}

func (s *USSDService) handleMultiBikeMenu(ctx context.Context, phone string, bikes []domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		var sb strings.Builder
		sb.WriteString("CON MotoGuard Fleet\nSelect Motorcycle:\n")
		for i, b := range bikes {
			sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, b.RegistrationNumber, b.Make))
		}
		sb.WriteString("0. Exit")
		return sb.String()
	}

	idxStr := strings.TrimSpace(steps[0])
	if idxStr == "0" {
		return "END Thank you for using MotoGuard."
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 1 || idx > len(bikes) {
		return "END Invalid motorcycle selection. Please dial again."
	}

	selectedBike := bikes[idx-1]
	return s.handleSingleBikeMenu(ctx, phone, selectedBike, steps[1:])
}

func (s *USSDService) renderBikeStatus(bike domain.RiderBikeInfo) string {
	ignitionStr := "OFF"
	if bike.Ignition {
		ignitionStr = "ON"
	}

	engineStr := "UNLOCKED"
	if bike.IsImmobilized {
		engineStr = "LOCKED (Immobilized)"
	}

	loc := bike.CurrentAddress
	if loc == "" || loc == "Location pending update" {
		if bike.SubCounty != "" || bike.County != "" {
			loc = fmt.Sprintf("%s, %s", bike.SubCounty, bike.County)
		} else {
			loc = "In Transit / GPS tracking active"
		}
	}

	return fmt.Sprintf("END MotoGuard %s\nStatus: %s\nIgnition: %s\nSpeed: %.0f km/h\nBattery: %d%%\nEngine: %s\nLocation: %s",
		bike.RegistrationNumber,
		bike.Status,
		ignitionStr,
		bike.SpeedKMH,
		bike.BatteryPercent,
		engineStr,
		loc,
	)
}

func (s *USSDService) handleLockFlow(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if bike.IsImmobilized {
		return fmt.Sprintf("END Motorcycle %s engine is ALREADY LOCKED (Immobilized). Starter is disabled.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON LOCK ENGINE: %s\nEnter your 4-digit PIN to disable starter:", bike.RegistrationNumber)
	}

	pin := strings.TrimSpace(steps[0])
	valid, err := s.VerifyPIN(ctx, phone, pin)
	if err != nil {
		return fmt.Sprintf("END %s", err.Error())
	}
	if !valid {
		return "END ❌ Incorrect PIN. Engine lock cancelled for your security."
	}

	// Perform Immobilization
	imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, true, "USSD (*384#)")
	if err != nil {
		slog.Error("Failed to lock bike via USSD", "bike_id", bike.ID, "error", err)
		return "END Unable to lock engine at this time. Please retry shortly."
	}

	// Publish MQTT command to ESP32 hardware
	if s.cmdPub != nil && imei != "" {
		_ = s.cmdPub.PublishCommand(imei, "IMMOBILIZE")
	}

	// Broadcast SSE update to frontend portal
	if s.sseBroker != nil {
		s.sseBroker.BroadcastAlert(userID, map[string]any{
			"event_type":  "IMMOBILIZED",
			"bike_id":     bike.ID,
			"title":       "🔒 Engine Immobilized via USSD",
			"activity":    fmt.Sprintf("Rider on %s engaged engine immobilization remotely.", phone),
			"recorded_at": time.Now().UTC(),
		})
	}

	// Send SMS confirmation
	smsMsg := fmt.Sprintf("🔒 MotoGuard: Engine for %s has been LOCKED successfully via USSD. Starter disabled.", bike.RegistrationNumber)
	go func() {
		norm, _ := repository.NormalizePhoneNumber(phone)
		_, _ = s.atClient.SendSMS(context.Background(), []string{norm}, smsMsg, "")
	}()

	return fmt.Sprintf("END 🔒 Engine LOCKED successfully!\nMotorcycle %s starter is now disabled.", bike.RegistrationNumber)
}

func (s *USSDService) handleUnlockFlow(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if !bike.IsImmobilized {
		return fmt.Sprintf("END Motorcycle %s engine is ALREADY UNLOCKED. Ready to ride.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON UNLOCK ENGINE: %s\nEnter your 4-digit PIN to restore starter:", bike.RegistrationNumber)
	}

	pin := strings.TrimSpace(steps[0])
	valid, err := s.VerifyPIN(ctx, phone, pin)
	if err != nil {
		return fmt.Sprintf("END %s", err.Error())
	}
	if !valid {
		return "END ❌ Incorrect PIN. Engine unlock cancelled."
	}

	// Perform Mobilization
	imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, false, "USSD (*384#)")
	if err != nil {
		slog.Error("Failed to unlock bike via USSD", "bike_id", bike.ID, "error", err)
		return "END Unable to unlock engine at this time. Please retry."
	}

	// Publish MQTT command to ESP32 hardware
	if s.cmdPub != nil && imei != "" {
		_ = s.cmdPub.PublishCommand(imei, "REACTIVATE")
	}

	// Broadcast SSE update
	if s.sseBroker != nil {
		s.sseBroker.BroadcastAlert(userID, map[string]any{
			"event_type":  "REACTIVATED",
			"bike_id":     bike.ID,
			"title":       "🔓 Engine Reactivated via USSD",
			"activity":    fmt.Sprintf("Rider on %s unlocked engine remotely.", phone),
			"recorded_at": time.Now().UTC(),
		})
	}

	// Send SMS confirmation
	smsMsg := fmt.Sprintf("🔓 MotoGuard: Engine for %s has been UNLOCKED successfully via USSD. Ready to ride.", bike.RegistrationNumber)
	go func() {
		norm, _ := repository.NormalizePhoneNumber(phone)
		_, _ = s.atClient.SendSMS(context.Background(), []string{norm}, smsMsg, "")
	}()

	return fmt.Sprintf("END 🔓 Engine UNLOCKED successfully!\nMotorcycle %s is ready to start.", bike.RegistrationNumber)
}

func (s *USSDService) handleSOSFlow(ctx context.Context, phone string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return fmt.Sprintf("CON 🚨 EMERGENCY SOS: %s\nAlert fleet manager and send your current location?\n1. YES, Send SOS Now\n2. Cancel", bike.RegistrationNumber)
	}

	choice := strings.TrimSpace(steps[0])
	if choice != "1" {
		return "END Emergency SOS cancelled."
	}

	// Record Emergency Event
	details := fmt.Sprintf("Emergency SOS triggered by rider (%s) via USSD", phone)
	_ = s.repo.RecordEmergencySOS(ctx, bike.ID, bike.UserID, bike.Latitude, bike.Longitude, details)

	// Send Alert to Fleet Owner and Rider
	ownerMsg := fmt.Sprintf("🚨 MOTORGUARD SOS ALERT! Rider %s (%s) triggered Emergency SOS on %s. Location: %s. Maps: https://maps.google.com/?q=%.6f,%.6f",
		bike.RiderName, phone, bike.RegistrationNumber, bike.CurrentAddress, bike.Latitude, bike.Longitude)

	riderMsg := fmt.Sprintf("🚨 MotoGuard: Emergency SOS dispatched for %s! Fleet manager has been notified with your current location.", bike.RegistrationNumber)

	go func() {
		bgCtx := context.Background()
		recipients := make([]string, 0, 2)
		normRider, _ := repository.NormalizePhoneNumber(phone)
		recipients = append(recipients, normRider)

		if bike.OwnerPhone != "" {
			normOwner, _ := repository.NormalizePhoneNumber(bike.OwnerPhone)
			if normOwner != normRider {
				_, _ = s.atClient.SendSMS(bgCtx, []string{normOwner}, ownerMsg, "")
			}
		}

		_, _ = s.atClient.SendSMS(bgCtx, []string{normRider}, riderMsg, "")
	}()

	// Push Notification & SSE Broadcast
	if s.sseBroker != nil {
		s.sseBroker.BroadcastAlert(bike.UserID, map[string]any{
			"event_type":  "SOS_DURESS",
			"bike_id":     bike.ID,
			"title":       "🚨 EMERGENCY SOS TRIGGERED",
			"activity":    fmt.Sprintf("Rider %s (%s) triggered an SOS alert via USSD.", bike.RiderName, phone),
			"latitude":    bike.Latitude,
			"longitude":   bike.Longitude,
			"recorded_at": time.Now().UTC(),
		})
	}

	if s.notifService != nil {
		_ = s.notifService.DispatchSecurityAlert(ctx, bike.UserID, notification.SecurityAlert{
			BikeID:     bike.ID,
			Title:      "🚨 EMERGENCY SOS TRIGGERED",
			Body:       fmt.Sprintf("Rider %s triggered SOS on %s at %s", bike.RiderName, bike.RegistrationNumber, bike.CurrentAddress),
			EventType:  "SOS_DURESS",
			Latitude:   bike.Latitude,
			Longitude:  bike.Longitude,
			RecordedAt: time.Now().UTC(),
		})
	}

	return fmt.Sprintf("END 🚨 SOS ALERT DISPATCHED!\nFleet manager and emergency contacts notified with your live coordinates.\nHelp is on the way.")
}

func (s *USSDService) handleChangePINFlow(ctx context.Context, phone string, steps []string) string {
	if len(steps) == 0 {
		return "CON CHANGE PIN\nEnter CURRENT 4-digit PIN (default 1234):"
	}

	currentPIN := strings.TrimSpace(steps[0])
	valid, err := s.VerifyPIN(ctx, phone, currentPIN)
	if err != nil {
		return fmt.Sprintf("END %s", err.Error())
	}
	if !valid {
		return "END ❌ Current PIN is incorrect. PIN change cancelled."
	}

	if len(steps) == 1 {
		return "CON Enter NEW 4-digit PIN:"
	}

	newPIN := strings.TrimSpace(steps[1])
	if len(newPIN) != 4 || !isAllDigits(newPIN) {
		return "END ❌ New PIN must be exactly 4 numeric digits. Please try again."
	}

	if len(steps) == 2 {
		return "CON Confirm NEW 4-digit PIN:"
	}

	confirmPIN := strings.TrimSpace(steps[2])
	if newPIN != confirmPIN {
		return "END ❌ PIN confirmation does not match. Please try again."
	}

	hashed := HashPIN(phone, newPIN)
	if err := s.repo.SaveRiderPIN(ctx, phone, hashed); err != nil {
		slog.Error("Failed to save new rider PIN", "phone", phone, "error", err)
		return "END Unable to update PIN at this time. Please retry."
	}

	go func() {
		norm, _ := repository.NormalizePhoneNumber(phone)
		msg := "MotoGuard: Your security PIN has been updated successfully. Keep this PIN safe."
		_, _ = s.atClient.SendSMS(context.Background(), []string{norm}, msg, "")
	}()

	return "END ✅ PIN updated successfully!\nPlease keep your new PIN safe."
}

// HandleIncomingSMS processes inbound SMS commands from riders (2-Way SMS).
func (s *USSDService) HandleIncomingSMS(ctx context.Context, req domain.SMSIncomingRequest) string {
	phone := req.From
	text := strings.TrimSpace(req.Text)

	if text == "" {
		return s.smsHelpMessage()
	}

	tokens := strings.Fields(text)
	cmd := strings.ToUpper(tokens[0])

	bikes, err := s.repo.GetBikesByPhone(ctx, phone)
	if err != nil || len(bikes) == 0 {
		return fmt.Sprintf("MotoGuard: Phone %s is not linked to any active motorcycle. Contact your fleet manager.", phone)
	}

	switch cmd {
	case "STATUS", "INFO", "WHERE":
		return s.handleSMSStatus(bikes, tokens[1:])
	case "LOCK", "STOP":
		return s.handleSMSLock(ctx, phone, bikes, tokens[1:], true)
	case "UNLOCK", "START":
		return s.handleSMSLock(ctx, phone, bikes, tokens[1:], false)
	case "SOS", "HELP", "EMERGENCY":
		return s.handleSMSSOS(ctx, phone, bikes)
	case "PIN":
		return s.handleSMSChangePIN(ctx, phone, tokens[1:])
	default:
		return s.smsHelpMessage()
	}
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

	return fmt.Sprintf("MotoGuard [%s]: Status: %s | Ign: %s | Speed: %.0fkm/h | Batt: %d%% | Engine: %s | Loc: %s",
		target.RegistrationNumber, target.Status, ign, target.SpeedKMH, target.BatteryPercent, engine, loc)
}

func (s *USSDService) handleSMSLock(ctx context.Context, phone string, bikes []domain.RiderBikeInfo, args []string, lock bool) string {
	actionName := "LOCK"
	if !lock {
		actionName = "UNLOCK"
	}

	if len(args) == 0 {
		return fmt.Sprintf("MotoGuard: Please include your 4-digit PIN. Example: %s 1234", actionName)
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
		return fmt.Sprintf("MotoGuard: %s", err.Error())
	}
	if !valid {
		return fmt.Sprintf("MotoGuard: ❌ Incorrect PIN. Engine %s aborted.", strings.ToLower(actionName))
	}

	imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, target.ID, lock, "SMS")
	if err != nil {
		return "MotoGuard: System busy. Unable to update engine state right now."
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
		return fmt.Sprintf("🔒 MotoGuard: Engine for %s is now LOCKED. Starter disabled.", target.RegistrationNumber)
	}
	return fmt.Sprintf("🔓 MotoGuard: Engine for %s is now UNLOCKED. Ready to ride.", target.RegistrationNumber)
}

func (s *USSDService) handleSMSSOS(ctx context.Context, phone string, bikes []domain.RiderBikeInfo) string {
	target := bikes[0]
	details := fmt.Sprintf("Emergency SOS triggered by rider (%s) via SMS", phone)
	_ = s.repo.RecordEmergencySOS(ctx, target.ID, target.UserID, target.Latitude, target.Longitude, details)

	// Alert owner via SMS
	if target.OwnerPhone != "" {
		ownerMsg := fmt.Sprintf("🚨 MOTORGUARD SOS ALERT! Rider %s (%s) sent SOS for %s. Location: %s. Maps: https://maps.google.com/?q=%.6f,%.6f",
			target.RiderName, phone, target.RegistrationNumber, target.CurrentAddress, target.Latitude, target.Longitude)
		normOwner, _ := repository.NormalizePhoneNumber(target.OwnerPhone)
		go func() {
			_, _ = s.atClient.SendSMS(context.Background(), []string{normOwner}, ownerMsg, "")
		}()
	}

	return fmt.Sprintf("🚨 MotoGuard: Emergency SOS received for %s! Fleet manager notified with your location (%s).",
		target.RegistrationNumber, target.CurrentAddress)
}

func (s *USSDService) handleSMSChangePIN(ctx context.Context, phone string, args []string) string {
	if len(args) < 2 {
		return "MotoGuard: To change PIN via SMS, send: PIN <CURRENT_PIN> <NEW_PIN>. Example: PIN 1234 5678"
	}

	currentPIN := args[0]
	newPIN := args[1]

	valid, err := s.VerifyPIN(ctx, phone, currentPIN)
	if err != nil {
		return fmt.Sprintf("MotoGuard: %s", err.Error())
	}
	if !valid {
		return "MotoGuard: ❌ Current PIN is incorrect. PIN change failed."
	}

	if len(newPIN) != 4 || !isAllDigits(newPIN) {
		return "MotoGuard: ❌ New PIN must be exactly 4 numeric digits."
	}

	hashed := HashPIN(phone, newPIN)
	if err := s.repo.SaveRiderPIN(ctx, phone, hashed); err != nil {
		return "MotoGuard: Error updating PIN. Please try again."
	}

	return "MotoGuard: ✅ Your security PIN has been updated successfully!"
}

func (s *USSDService) smsHelpMessage() string {
	return "MotoGuard SMS Commands:\n- STATUS: Check motorcycle\n- LOCK <PIN>: Lock engine\n- UNLOCK <PIN>: Unlock engine\n- SOS: Send emergency alert\n- PIN <OLD> <NEW>: Change PIN"
}

func isAllDigits(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
