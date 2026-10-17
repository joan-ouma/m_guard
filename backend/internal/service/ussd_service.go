package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

	// Dynamic trigger label based on caller's dialed serviceCode
	trigger := "USSD"
	if strings.TrimSpace(req.ServiceCode) != "" {
		trigger = fmt.Sprintf("USSD (%s)", strings.TrimSpace(req.ServiceCode))
	}

	// Single bike flow vs Multi-bike flow
	if len(bikes) == 1 {
		return s.handleSingleBikeMenu(ctx, phone, trigger, bikes[0], steps)
	}

	return s.handleMultiBikeMenu(ctx, phone, trigger, bikes, steps)
}

func (s *USSDService) handleSingleBikeMenu(ctx context.Context, phone, trigger string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return "CON Welcome to M-Guard\n1. Find bike\n2. Immobilize bike\n3. Reactivate bike\n4. Emergency phones"
	}

	choice := steps[0]
	switch choice {
	case "1":
		return s.handleFindBike(bike)
	case "2":
		return s.handleImmobilizeBike(ctx, phone, trigger, bike, steps[1:])
	case "3":
		return s.handleReactivateBike(ctx, phone, trigger, bike, steps[1:])
	case "4":
		return s.handleEmergencyPhones(bike, steps[1:])
	default:
		return "END Invalid option. Session ended."
	}
}

func (s *USSDService) handleMultiBikeMenu(ctx context.Context, phone, trigger string, bikes []domain.RiderBikeInfo, steps []string) string {
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
	return s.handleSingleBikeMenu(ctx, phone, trigger, selectedBike, steps[1:])
}

// 1. Find bike
func (s *USSDService) handleFindBike(bike domain.RiderBikeInfo) string {
	return s.renderBikeStatus(bike)
}

func (s *USSDService) renderBikeStatus(bike domain.RiderBikeInfo) string {
	loc := strings.TrimSpace(bike.CurrentAddress)
	if loc == "" || strings.EqualFold(loc, "Location pending update") {
		subCounty := strings.TrimSpace(bike.SubCounty)
		county := strings.TrimSpace(bike.County)
		if subCounty != "" || county != "" {
			if subCounty != "" && county != "" {
				loc = fmt.Sprintf("%s, %s", subCounty, county)
			} else if subCounty != "" {
				loc = subCounty
			} else {
				loc = county
			}
		} else if bike.Latitude != 0 && bike.Longitude != 0 {
			loc = fmt.Sprintf("%.4f, %.4f", bike.Latitude, bike.Longitude)
		} else {
			loc = "Location pending update"
		}
	}

	status := strings.TrimSpace(bike.Status)
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
func (s *USSDService) handleImmobilizeBike(ctx context.Context, phone, trigger string, bike domain.RiderBikeInfo, steps []string) string {
	if bike.IsImmobilized {
		return fmt.Sprintf("END Bike %s is already immobilized.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON Turn off engine for %s?\n1. Yes, shut down\n0. Back", bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, true, trigger)
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

		if s.notifService != nil {
			_ = s.notifService.DispatchSecurityAlert(ctx, userID, notification.SecurityAlert{
				BikeID:     bike.ID,
				Title:      "🔒 Engine Immobilized via USSD",
				Body:       fmt.Sprintf("Motorcycle %s engine immobilized remotely via USSD by %s", bike.RegistrationNumber, phone),
				EventType:  "IMMOBILIZE",
				RecordedAt: time.Now().UTC(),
			})
		}

		return "END Engine immobilized. Starter has been disabled safely."
	}

	return "END Invalid option. Session ended."
}

// 3. Reactivate bike
func (s *USSDService) handleReactivateBike(ctx context.Context, phone, trigger string, bike domain.RiderBikeInfo, steps []string) string {
	if !bike.IsImmobilized {
		return fmt.Sprintf("END Bike %s is already active.", bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf("CON Restore engine for %s?\n1. Yes, reactivate\n0. Back", bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		imei, userID, err := s.repo.SetBikeImmobilizedByRider(ctx, bike.ID, false, trigger)
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

		if s.notifService != nil {
			_ = s.notifService.DispatchSecurityAlert(ctx, userID, notification.SecurityAlert{
				BikeID:     bike.ID,
				Title:      "🔓 Engine Reactivated via USSD",
				Body:       fmt.Sprintf("Motorcycle %s engine restored remotely via USSD by %s", bike.RegistrationNumber, phone),
				EventType:  "REACTIVATED",
				RecordedAt: time.Now().UTC(),
			})
		}

		return "END Engine reactivated. Bike is ready to ride."
	}

	return "END Invalid option. Session ended."
}

// 4. Emergency phones - searchable police & fleet directory
func (s *USSDService) handleEmergencyPhones(bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return "CON Emergency Directory:\n1. Fleet Owner\n2. Search Police Station\n0. Back"
	}

	switch steps[0] {
	case "1":
		var sb strings.Builder
		sb.WriteString("CON Fleet Owner:\n")
		if strings.TrimSpace(bike.OwnerPhone) != "" {
			ownerLabel := strings.TrimSpace(bike.OwnerName)
			if ownerLabel == "" || strings.EqualFold(ownerLabel, "Fleet Owner") {
				ownerLabel = "Fleet Manager"
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", ownerLabel, strings.TrimSpace(bike.OwnerPhone)))
		} else {
			sb.WriteString("No fleet owner phone on record.\n")
		}

		supportPhone := strings.TrimSpace(os.Getenv("EMERGENCY_SUPPORT_PHONE"))
		if supportPhone != "" {
			sb.WriteString(fmt.Sprintf("Support: %s\n", supportPhone))
		}

		policeHotline := strings.TrimSpace(os.Getenv("POLICE_HOTLINE"))
		if policeHotline == "" {
			policeHotline = "999 / 112"
		}
		sb.WriteString(fmt.Sprintf("National Police: %s\n", policeHotline))
		sb.WriteString("0. Back")
		return sb.String()

	case "2":
		if len(steps) == 1 {
			return "CON Enter town or station name (e.g. Rongai, Karen, Central):\n0. Back"
		}
		searchTerm := strings.TrimSpace(steps[1])
		return searchPoliceStations(searchTerm)

	default:
		return "END Invalid option. Session ended."
	}
}

type policeStationEntry struct {
	Name     string
	Phone    string
	Keywords []string
}

var kenyaPoliceStations = []policeStationEntry{
	{Name: "Ongata Rongai Police", Phone: "0721 334 999", Keywords: []string{"rongai", "ongata", "rimpa", "kajiado north"}},
	{Name: "Karen Police Station", Phone: "0722 841 000", Keywords: []string{"karen", "hardy"}},
	{Name: "Central Police Station", Phone: "020 222 2222", Keywords: []string{"central", "cbd", "nairobi cbd", "city"}},
	{Name: "Kilimani Police Station", Phone: "020 272 2222", Keywords: []string{"kilimani", "kileleshwa", "hurlingham"}},
	{Name: "Langata Police Station", Phone: "0721 365 000", Keywords: []string{"langata", "south c", "madaraka"}},
	{Name: "Kasarani Police Station", Phone: "0721 364 888", Keywords: []string{"kasarani", "roysambu", "thika road", "clay city"}},
	{Name: "Gigiri Police Station", Phone: "0721 364 777", Keywords: []string{"gigiri", "runda", "rosslyn"}},
	{Name: "Parklands / Westlands", Phone: "0721 364 111", Keywords: []string{"westlands", "parklands", "spring valley"}},
	{Name: "Embakasi Police Station", Phone: "0721 364 444", Keywords: []string{"embakasi", "fedha", "tassia", "pipeline"}},
	{Name: "Industrial Area Police", Phone: "0721 364 555", Keywords: []string{"industrial area", "south b"}},
	{Name: "Buruburu Police Station", Phone: "0721 364 666", Keywords: []string{"buruburu", "umoja", "donholm"}},
	{Name: "Kayole Police Station", Phone: "0721 364 333", Keywords: []string{"kayole", "komarock", "miango"}},
	{Name: "Pangani Police Station", Phone: "0721 364 222", Keywords: []string{"pangani", "ngara", "mlango kubwa"}},
	{Name: "Ngong Police Station", Phone: "0722 841 000", Keywords: []string{"ngong", "kiserian", "matasia"}},
	{Name: "Kitengela Police Station", Phone: "0720 555 888", Keywords: []string{"kitengela", "athi river", "mlolongo", "syokimau"}},
	{Name: "Ruiru Police Station", Phone: "0722 841 111", Keywords: []string{"ruiru", "kahawa", "juja"}},
	{Name: "Thika Police Station", Phone: "0722 841 222", Keywords: []string{"thika", "makongeni"}},
	{Name: "Machakos Police Station", Phone: "0722 841 333", Keywords: []string{"machakos", "katumani"}},
	{Name: "Nakuru Central Police", Phone: "0722 841 555", Keywords: []string{"nakuru", "njoro"}},
	{Name: "Naivasha Police Station", Phone: "0722 841 556", Keywords: []string{"naivasha"}},
	{Name: "Eldoret Central Police", Phone: "0722 841 666", Keywords: []string{"eldoret", "uasin gishu"}},
	{Name: "Kisumu Central Police", Phone: "0722 841 777", Keywords: []string{"kisumu", "kondele"}},
	{Name: "Mombasa Central Police", Phone: "0722 841 888", Keywords: []string{"mombasa", "nyali", "changamwe", "likoni"}},
	{Name: "Malindi Police Station", Phone: "0722 841 999", Keywords: []string{"malindi", "kilifi"}},
}

func searchPoliceStations(term string) string {
	q := strings.ToLower(strings.TrimSpace(term))
	if q == "" {
		return "CON Enter a valid town or station name:\n0. Back"
	}

	var matched []policeStationEntry
	for _, st := range kenyaPoliceStations {
		if strings.Contains(strings.ToLower(st.Name), q) {
			matched = append(matched, st)
			continue
		}
		for _, kw := range st.Keywords {
			if strings.Contains(kw, q) || strings.Contains(q, kw) {
				matched = append(matched, st)
				break
			}
		}
		if len(matched) >= 3 {
			break
		}
	}

	if len(matched) == 0 {
		return fmt.Sprintf("CON No station found for %q.\nNational Hotline: 999 / 112\nDCI Hotline: 0800 722 203\n0. Back", term)
	}

	var sb strings.Builder
	sb.WriteString("CON Search Results:\n")
	for _, st := range matched {
		sb.WriteString(fmt.Sprintf("%s:\n%s\n", st.Name, st.Phone))
	}
	sb.WriteString("0. Back")
	return sb.String()
}



func isAllDigits(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
