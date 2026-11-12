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
	"motoguard-backend/internal/sms"
	"motoguard-backend/internal/sse"
)

type USSDService struct {
	repo         *repository.USSDRepository
	cmdPub       CommandPublisher
	sseBroker    *sse.Broker
	notifService *notification.Service
	smsSender    sms.Sender
	// smsRouting is the shared gateway's routing keyword (for example "kamilimu").
	smsRouting string
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
	reply := s.ussdMenu(ctx, req, langEN)
	slog.Info("USSD menu", "phone", req.PhoneNumber, "session_id", req.SessionID,
		"service_code", req.ServiceCode, "input", req.Text, "reply", reply)
	return reply
}

// ussdMenu renders the menu reply for a USSD request. It is shared by USSD and the SMS menu.
// lang selects the wording: langEN for USSD, the rider's chosen language for SMS.
func (s *USSDService) ussdMenu(ctx context.Context, req domain.USSDRequest, lang string) string {
	phone := req.PhoneNumber

	// Lookup bikes registered to this caller's phone number
	bikes, err := s.repo.GetBikesByPhone(ctx, phone)
	if err != nil {
		slog.Error("USSD lookup bikes error", "phone", phone, "error", err)
		return "END " + ussdText(lang,
			"System temporarily unavailable. Please try again shortly.",
			"Huduma haipatikani kwa sasa. Tafadhali jaribu tena baadaye.")
	}

	// 1. If phone number is not registered on the platform, do NOT allow procession
	if len(bikes) == 0 {
		return "END " + ussdText(lang,
			"Phone number not registered on M-Guard. Please contact your administrator.",
			"Namba ya simu haijasajiliwa kwenye M-Guard. Tafadhali wasiliana na msimamizi wako.")
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
		return s.handleSingleBikeMenu(ctx, phone, trigger, lang, bikes[0], steps)
	}

	return s.handleMultiBikeMenu(ctx, phone, trigger, lang, bikes, steps)
}

func (s *USSDService) handleSingleBikeMenu(ctx context.Context, phone, trigger, lang string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return ussdText(lang,
			"CON Welcome to M-Guard\n1. Find bike\n2. Immobilize bike\n3. Reactivate bike\n4. Emergency phones",
			"CON Karibu M-Guard\n1. Tafuta pikipiki\n2. Zuia pikipiki\n3. Washa pikipiki\n4. Simu za dharura")
	}

	choice := steps[0]
	switch choice {
	case "1":
		return s.handleFindBike(bike, lang)
	case "2":
		return s.handleImmobilizeBike(ctx, phone, trigger, lang, bike, steps[1:])
	case "3":
		return s.handleReactivateBike(ctx, phone, trigger, lang, bike, steps[1:])
	case "4":
		return s.handleEmergencyPhones(lang, bike, steps[1:])
	default:
		return "END " + invalidOptionText(lang)
	}
}

func (s *USSDService) handleMultiBikeMenu(ctx context.Context, phone, trigger, lang string, bikes []domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		var sb strings.Builder
		sb.WriteString(ussdText(lang,
			"CON Welcome to M-Guard\nSelect bike:\n",
			"CON Karibu M-Guard\nChagua pikipiki:\n"))
		for i, b := range bikes {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, b.RegistrationNumber))
		}
		sb.WriteString(ussdText(lang, "0. Exit", "0. Toka"))
		return sb.String()
	}

	idxStr := steps[0]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 1 || idx > len(bikes) {
		return "END " + ussdText(lang, "Invalid selection. Session ended.", "Uchaguzi batili. Kipindi kimefungwa.")
	}

	selectedBike := bikes[idx-1]
	return s.handleSingleBikeMenu(ctx, phone, trigger, lang, selectedBike, steps[1:])
}

// invalidOptionText is the end-of-session text for an option the rider did not pick from the menu.
func invalidOptionText(lang string) string {
	return ussdText(lang, "Invalid option. Session ended.", "Chaguo batili. Kipindi kimefungwa.")
}

// 1. Find bike
func (s *USSDService) handleFindBike(bike domain.RiderBikeInfo, lang string) string {
	return s.renderBikeStatus(bike, lang)
}

func (s *USSDService) renderBikeStatus(bike domain.RiderBikeInfo, lang string) string {
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
			loc = ussdText(lang, "Location pending update", "Mahali kunasubiriwa")
		}
	}

	status := strings.TrimSpace(bike.Status)
	if status == "" {
		if bike.Ignition {
			status = ussdText(lang, "RUNNING", "INATEMBEA")
		} else {
			status = ussdText(lang, "STOPPED", "IMESIMAMA")
		}
	}

	return fmt.Sprintf(ussdText(lang,
		"CON Bike %s\nStatus: %s\nBattery: %d%%\nLocation: %s\n0. Back",
		"CON Pikipiki %s\nHali: %s\nBetri: %d%%\nMahali: %s\n0. Rudi"),
		bike.RegistrationNumber, status, bike.BatteryPercent, loc)
}

// 2. Immobilize bike
func (s *USSDService) handleImmobilizeBike(ctx context.Context, phone, trigger, lang string, bike domain.RiderBikeInfo, steps []string) string {
	if bike.IsImmobilized {
		return fmt.Sprintf(ussdText(lang,
			"END Bike %s is already immobilized.",
			"END Pikipiki %s tayari imezuiwa."), bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf(ussdText(lang,
			"CON Turn off engine for %s?\n1. Yes, shut down\n0. Back",
			"CON Zima injini ya %s?\n1. Ndiyo, zima\n0. Rudi"), bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		ts := time.Now().Unix()
		imei, userID, err := s.repo.CreatePendingCommand(ctx, bike.ID, true, ts)
		if errors.Is(err, repository.ErrCommandPending) {
			return fmt.Sprintf(ussdText(lang,
				"END A command for %s is still awaiting confirmation. Please wait.",
				"END Amri ya %s bado inasubiri uthibitisho. Tafadhali subiri."), bike.RegistrationNumber)
		}
		if err != nil {
			slog.Error("Failed to request immobilization via USSD", "bike_id", bike.ID, "error", err)
			return "END " + ussdText(lang,
				"Unable to immobilize bike at this time. Please try again.",
				"Haiwezekani kuzuia pikipiki kwa sasa. Tafadhali jaribu tena.")
		}

		if s.cmdPub != nil && imei != "" {
			if err := s.cmdPub.PublishCommand(imei, "IMMOBILIZE", ts); err != nil {
				slog.Warn("USSD immobilize command not sent, will retry", "bike_id", bike.ID, "error", err)
			}
		}

		if s.sseBroker != nil {
			s.sseBroker.BroadcastAlert(userID, map[string]any{
				"event_type":  "IMMOBILIZED",
				"bike_id":     bike.ID,
				"title":       "🔒 Engine Immobilization Requested via USSD",
				"activity":    fmt.Sprintf("Rider on %s requested engine immobilization. Awaiting bike confirmation.", phone),
				"recorded_at": time.Now().UTC(),
			})
		}

		if s.notifService != nil {
			_ = s.notifService.DispatchSecurityAlert(ctx, userID, notification.SecurityAlert{
				BikeID:     bike.ID,
				Title:      "🔒 Engine Immobilization Requested via USSD",
				Body:       fmt.Sprintf("Immobilization of motorcycle %s was requested via USSD by %s and is awaiting confirmation", bike.RegistrationNumber, phone),
				EventType:  "IMMOBILIZE",
				RecordedAt: time.Now().UTC(),
			})
		}

		return "END " + ussdText(lang,
			"Immobilization requested. The starter is disabled once the bike confirms.",
			"Ombi la kuzuia limetumwa. Kianzishi kitazimwa pikipiki itakapothibitisha.")
	}

	return "END " + invalidOptionText(lang)
}

// 3. Reactivate bike
func (s *USSDService) handleReactivateBike(ctx context.Context, phone, trigger, lang string, bike domain.RiderBikeInfo, steps []string) string {
	if !bike.IsImmobilized {
		return fmt.Sprintf(ussdText(lang,
			"END Bike %s is already active.",
			"END Pikipiki %s tayari inafanya kazi."), bike.RegistrationNumber)
	}

	if len(steps) == 0 {
		return fmt.Sprintf(ussdText(lang,
			"CON Restore engine for %s?\n1. Yes, reactivate\n0. Back",
			"CON Washa injini ya %s?\n1. Ndiyo, washa\n0. Rudi"), bike.RegistrationNumber)
	}

	if steps[0] == "1" {
		ts := time.Now().Unix()
		imei, userID, err := s.repo.CreatePendingCommand(ctx, bike.ID, false, ts)
		if errors.Is(err, repository.ErrCommandPending) {
			return fmt.Sprintf(ussdText(lang,
				"END A command for %s is still awaiting confirmation. Please wait.",
				"END Amri ya %s bado inasubiri uthibitisho. Tafadhali subiri."), bike.RegistrationNumber)
		}
		if err != nil {
			slog.Error("Failed to request reactivation via USSD", "bike_id", bike.ID, "error", err)
			return "END " + ussdText(lang,
				"Unable to reactivate bike at this time. Please try again.",
				"Haiwezekani kuwasha pikipiki kwa sasa. Tafadhali jaribu tena.")
		}

		if s.cmdPub != nil && imei != "" {
			if err := s.cmdPub.PublishCommand(imei, "REACTIVATE", ts); err != nil {
				slog.Warn("USSD reactivate command not sent, will retry", "bike_id", bike.ID, "error", err)
			}
		}

		if s.sseBroker != nil {
			s.sseBroker.BroadcastAlert(userID, map[string]any{
				"event_type":  "REACTIVATED",
				"bike_id":     bike.ID,
				"title":       "🔓 Engine Reactivation Requested via USSD",
				"activity":    fmt.Sprintf("Rider on %s requested engine restore. Awaiting bike confirmation.", phone),
				"recorded_at": time.Now().UTC(),
			})
		}

		if s.notifService != nil {
			_ = s.notifService.DispatchSecurityAlert(ctx, userID, notification.SecurityAlert{
				BikeID:     bike.ID,
				Title:      "🔓 Engine Reactivation Requested via USSD",
				Body:       fmt.Sprintf("Engine restore for motorcycle %s was requested via USSD by %s and is awaiting confirmation", bike.RegistrationNumber, phone),
				EventType:  "REACTIVATED",
				RecordedAt: time.Now().UTC(),
			})
		}

		return "END " + ussdText(lang,
			"Reactivation requested. The bike is ready to ride once it confirms.",
			"Ombi la kuwasha limetumwa. Pikipiki itakuwa tayari kutumika itakapothibitisha.")
	}

	return "END " + invalidOptionText(lang)
}

// 4. Emergency phones - searchable police & fleet directory
func (s *USSDService) handleEmergencyPhones(lang string, bike domain.RiderBikeInfo, steps []string) string {
	if len(steps) == 0 {
		return ussdText(lang,
			"CON Emergency Directory:\n1. Fleet Owner\n2. Search Police Station\n0. Back",
			"CON Orodha ya dharura:\n1. Mmiliki wa meli\n2. Tafuta kituo cha polisi\n0. Rudi")
	}

	switch steps[0] {
	case "1":
		var sb strings.Builder
		sb.WriteString(ussdText(lang, "CON Fleet Owner:\n", "CON Mmiliki wa meli:\n"))
		if strings.TrimSpace(bike.OwnerPhone) != "" {
			ownerLabel := strings.TrimSpace(bike.OwnerName)
			if ownerLabel == "" || strings.EqualFold(ownerLabel, "Fleet Owner") {
				ownerLabel = ussdText(lang, "Fleet Manager", "Msimamizi wa meli")
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", ownerLabel, strings.TrimSpace(bike.OwnerPhone)))
		} else {
			sb.WriteString(ussdText(lang,
				"No fleet owner phone on record.\n",
				"Hakuna namba ya simu ya mmiliki.\n"))
		}

		supportPhone := strings.TrimSpace(os.Getenv("EMERGENCY_SUPPORT_PHONE"))
		if supportPhone != "" {
			sb.WriteString(fmt.Sprintf(ussdText(lang, "Support: %s\n", "Msaada: %s\n"), supportPhone))
		}

		policeHotline := strings.TrimSpace(os.Getenv("POLICE_HOTLINE"))
		if policeHotline == "" {
			policeHotline = "999 / 112"
		}
		sb.WriteString(fmt.Sprintf(ussdText(lang, "National Police: %s\n", "Polisi wa Taifa: %s\n"), policeHotline))
		sb.WriteString(ussdText(lang, "0. Back", "0. Rudi"))
		return sb.String()

	case "2":
		if len(steps) == 1 {
			return ussdText(lang,
				"CON Enter town or station name (e.g. Rongai, Karen, Central):\n0. Back",
				"CON Andika jina la mji au kituo (k.m. Rongai, Karen, Central):\n0. Rudi")
		}
		searchTerm := strings.TrimSpace(steps[1])
		return searchPoliceStations(searchTerm, lang)

	default:
		return "END " + invalidOptionText(lang)
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

func searchPoliceStations(term, lang string) string {
	q := strings.ToLower(strings.TrimSpace(term))
	if q == "" {
		return ussdText(lang,
			"CON Enter a valid town or station name:\n0. Back",
			"CON Andika jina sahihi la mji au kituo:\n0. Rudi")
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
		return fmt.Sprintf(ussdText(lang,
			"CON No station found for %q.\nNational Hotline: 999 / 112\nDCI Hotline: 0800 722 203\n0. Back",
			"CON Hakuna kituo kilichopatikana kwa %q.\nNambari ya dharura ya taifa: 999 / 112\nDCI: 0800 722 203\n0. Rudi"), term)
	}

	var sb strings.Builder
	sb.WriteString(ussdText(lang, "CON Search Results:\n", "CON Matokeo ya utafutaji:\n"))
	for _, st := range matched {
		sb.WriteString(fmt.Sprintf("%s:\n%s\n", st.Name, st.Phone))
	}
	sb.WriteString(ussdText(lang, "0. Back", "0. Rudi"))
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
