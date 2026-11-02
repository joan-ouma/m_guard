package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"motoguard-backend/internal/content"
	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/security"
)

const (
	chatInstance        = "/v1/chat"
	chatStaleAfter      = 15 * time.Minute
	chatMaxBikeChoices  = 50
	chatMaxIncidents    = 25
	chatMaxContacts     = 50
	chatRecentEventsMax = 5
	chatMaxPathLength   = 4
	chatMaxLabelRunes   = 120
	chatHistoryDefault  = 30
	chatHistoryMax      = 100
	chatRetention       = 90 * 24 * time.Hour
	chatPurgeInterval   = 24 * time.Hour
)

// chatLocation is East Africa Time, used for timestamps shown to riders.
var chatLocation = time.FixedZone("EAT", 3*60*60)

// ChatBikeReader is the subset of BikeService the chatbot needs. Every call is scoped to the signed-in user.
type ChatBikeReader interface {
	ListBikes(ctx context.Context, claims *security.JWTClaims, cursor string, limit int) (*domain.BikeListResponse, error)
	GetBike(ctx context.Context, claims *security.JWTClaims, bikeID string) (*domain.BikeDTO, error)
	GetEvents(ctx context.Context, claims *security.JWTClaims, bikeID, startDate, endDate string, cursor int64, limit int) (*domain.PaginatedEventsResponse, error)
}

// ChatRecoveryReader gives the chatbot the rider's open theft reports and saved contacts.
type ChatRecoveryReader interface {
	ListIncidents(ctx context.Context, userID string, limit, offset int) (*IncidentListResult, error)
	ListContacts(ctx context.Context, userID string) ([]domain.UserContactDTO, error)
}

// ChatHistoryStore persists the rider's chat history.
type ChatHistoryStore interface {
	Append(ctx context.Context, userID, role, text string) error
	Recent(ctx context.Context, userID string, limit int) ([]domain.ChatHistoryItem, error)
	Clear(ctx context.Context, userID string) error
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// ChatComposer turns verified bike facts into a short reply in the chosen language.
type ChatComposer interface {
	Compose(ctx context.Context, language string, facts any) (string, error)
}

// ChatService drives the guided menu. It only reads data; it never changes a bike or a report.
type ChatService struct {
	bikes    ChatBikeReader
	recovery ChatRecoveryReader
	history  ChatHistoryStore
	composer ChatComposer // optional; when nil or failing, built-in wording is used
}

func NewChatService(bikes ChatBikeReader, recovery ChatRecoveryReader, history ChatHistoryStore, composer ChatComposer) *ChatService {
	return &ChatService{bikes: bikes, recovery: recovery, history: history, composer: composer}
}

// Respond returns the next menu or the final answer for the path the rider has selected.
//
//	[]                                    -> language menu
//	[lang]                                -> top menu
//	[lang, section]                       -> that section's menu (bikes, incidents, contacts, emergency, terms, privacy)
//	[lang, section, item]                 -> answer, or a sub-menu for bikes (bike:<id>)
//	[lang, section:bikes, bike, action]   -> bike answer
func (s *ChatService) Respond(ctx context.Context, claims *security.JWTClaims, path []string, label string) (*domain.ChatResponse, error) {
	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", chatInstance)
	}
	if len(path) > chatMaxPathLength {
		return nil, invalidChatPath()
	}
	if len(path) == 0 {
		return languageMenu(), nil
	}

	lang, ok := strings.CutPrefix(path[0], "lang:")
	if !ok || !isSupportedChatLanguage(lang) {
		return nil, invalidChatPath()
	}
	if len(path) == 1 {
		return topMenu(lang), nil
	}

	section, ok := strings.CutPrefix(path[1], "section:")
	if !ok {
		return nil, invalidChatPath()
	}

	var (
		resp *domain.ChatResponse
		err  error
	)
	rest := path[2:]
	switch section {
	case "bikes":
		resp, err = s.bikesRoute(ctx, claims, lang, rest)
	case "incidents":
		resp, err = s.incidentsRoute(ctx, claims, lang, rest)
	case "contacts":
		resp, err = s.contactsRoute(ctx, claims, lang, rest)
	case "emergency":
		resp, err = emergencyRoute(lang, rest)
	case "terms":
		resp, err = clauseRoute(lang, content.Terms, rest)
	case "privacy":
		resp, err = clauseRoute(lang, content.Privacy, rest)
	default:
		return nil, invalidChatPath()
	}
	if err != nil {
		return nil, err
	}

	if resp.Type == "answer" {
		s.recordExchange(ctx, claims.UserID, label, resp.Message)
	}
	return resp, nil
}

// History returns the rider's most recent stored messages, oldest first.
func (s *ChatService) History(ctx context.Context, claims *security.JWTClaims, limit int) ([]domain.ChatHistoryItem, error) {
	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", chatInstance)
	}
	if limit <= 0 {
		limit = chatHistoryDefault
	}
	if limit > chatHistoryMax {
		limit = chatHistoryMax
	}
	items, err := s.history.Recent(ctx, claims.UserID, limit)
	if err != nil {
		slog.Error("chat history read failed", "user_id", claims.UserID, "error", err)
		return nil, domain.NewInternalServerError(chatInstance)
	}
	return items, nil
}

// ClearHistory deletes all of the rider's stored chat messages.
func (s *ChatService) ClearHistory(ctx context.Context, claims *security.JWTClaims) error {
	if claims == nil || claims.UserID == "" {
		return domain.NewUnauthorizedError("Authentication token is missing", chatInstance)
	}
	if err := s.history.Clear(ctx, claims.UserID); err != nil {
		slog.Error("chat history clear failed", "user_id", claims.UserID, "error", err)
		return domain.NewInternalServerError(chatInstance)
	}
	return nil
}

// RunRetention deletes chat history older than the retention period, once at start and then daily.
// It stops when ctx is cancelled.
func (s *ChatService) RunRetention(ctx context.Context) {
	purge := func() {
		removed, err := s.history.PurgeOlderThan(ctx, time.Now().Add(-chatRetention))
		if err != nil {
			slog.Error("chat history purge failed", "error", err)
			return
		}
		if removed > 0 {
			slog.Info("chat history purged", "rows", removed)
		}
	}

	purge()
	ticker := time.NewTicker(chatPurgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}

// recordExchange stores the rider's tap and the answer. A storage failure never blocks the reply.
func (s *ChatService) recordExchange(ctx context.Context, userID, label, answer string) {
	label = truncateLabel(label)
	if label != "" {
		if err := s.history.Append(ctx, userID, "rider", label); err != nil {
			slog.Error("chat history write failed", "user_id", userID, "error", err)
		}
	}
	if err := s.history.Append(ctx, userID, "bot", answer); err != nil {
		slog.Error("chat history write failed", "user_id", userID, "error", err)
	}
}

// ---- Menus ----

func languageMenu() *domain.ChatResponse {
	return menuResponse(
		"Choose a language for this information. / Chagua lugha ya taarifa hii.",
		[]domain.ChatOption{
			{ID: "lang:en", Label: "English"},
			{ID: "lang:sw", Label: "Kiswahili"},
		},
	)
}

func topMenu(lang string) *domain.ChatResponse {
	sections := []string{"bikes", "incidents", "contacts", "emergency", "terms", "privacy"}
	options := make([]domain.ChatOption, 0, len(sections))
	for _, sec := range sections {
		options = append(options, domain.ChatOption{ID: "section:" + sec, Label: chatText(lang, "section:"+sec)})
	}
	return menuResponse(chatText(lang, "topPrompt"), options)
}

func (s *ChatService) bikeMenu(ctx context.Context, claims *security.JWTClaims, lang string) (*domain.ChatResponse, error) {
	list, err := s.bikes.ListBikes(ctx, claims, "", chatMaxBikeChoices)
	if err != nil {
		return nil, err
	}
	if len(list.Data) == 0 {
		return answerResponse(chatText(lang, "noBikes")), nil
	}

	options := make([]domain.ChatOption, 0, len(list.Data))
	for i := range list.Data {
		options = append(options, domain.ChatOption{
			ID:    "bike:" + list.Data[i].ID,
			Label: bikeDisplayName(&list.Data[i]),
		})
	}
	return menuResponse(chatText(lang, "bikePrompt"), options), nil
}

func actionMenu(lang string, bike *domain.BikeDTO) *domain.ChatResponse {
	actions := []string{"location", "status", "activity", "details"}
	options := make([]domain.ChatOption, 0, len(actions))
	for _, a := range actions {
		options = append(options, domain.ChatOption{ID: "action:" + a, Label: chatText(lang, "action:"+a)})
	}
	return menuResponse(fmt.Sprintf(chatText(lang, "actionPrompt"), bikeDisplayName(bike)), options)
}

// ---- Bikes ----

func (s *ChatService) bikesRoute(ctx context.Context, claims *security.JWTClaims, lang string, rest []string) (*domain.ChatResponse, error) {
	if len(rest) == 0 {
		return s.bikeMenu(ctx, claims, lang)
	}
	bikeID, ok := strings.CutPrefix(rest[0], "bike:")
	if !ok || bikeID == "" || len(rest) > 2 {
		return nil, invalidChatPath()
	}
	// GetBike enforces ownership: a rider cannot reach another rider's bike by guessing an ID.
	bike, err := s.bikes.GetBike(ctx, claims, bikeID)
	if err != nil {
		return nil, err
	}
	if len(rest) == 1 {
		return actionMenu(lang, bike), nil
	}

	action, ok := strings.CutPrefix(rest[1], "action:")
	if !ok {
		return nil, invalidChatPath()
	}
	if action == "details" {
		return answerResponse(bikeDetailsText(lang, bike)), nil
	}
	if !isBikeFactAction(action) {
		return nil, invalidChatPath()
	}
	return s.bikeAnswer(ctx, claims, lang, bike, action)
}

func (s *ChatService) bikeAnswer(ctx context.Context, claims *security.JWTClaims, lang string, bike *domain.BikeDTO, action string) (*domain.ChatResponse, error) {
	facts, err := s.buildFacts(ctx, claims, bike, action, time.Now())
	if err != nil {
		return nil, err
	}

	message := ""
	if s.composer != nil {
		text, composeErr := s.composer.Compose(ctx, lang, facts)
		if composeErr != nil {
			slog.Warn("chat composer failed, using built-in wording", "error", composeErr, "action", action)
		} else {
			message = strings.TrimSpace(text)
		}
	}
	if message == "" {
		message = fallbackAnswer(lang, facts)
	}
	return answerResponse(message), nil
}

func (s *ChatService) buildFacts(ctx context.Context, claims *security.JWTClaims, bike *domain.BikeDTO, action string, now time.Time) (any, error) {
	base := newChatBase(bike, action, now)

	switch action {
	case "location":
		return locationFacts{
			chatBase:  base,
			Address:   bike.CurrentAddress,
			SubCounty: bike.SubCounty,
			County:    bike.County,
			Latitude:  bike.Latitude,
			Longitude: bike.Longitude,
			Ignition:  bike.Ignition,
		}, nil

	case "status":
		return statusFacts{
			chatBase:         base,
			Ignition:         bike.Ignition,
			SpeedKMH:         bike.SpeedKMH,
			BatteryPercent:   bike.BatteryPercent,
			SignalQualityCSQ: bike.SignalQualityCSQ,
			IsImmobilized:    bike.IsImmobilized,
			Status:           bike.Status,
		}, nil

	case "activity":
		events, err := s.bikes.GetEvents(ctx, claims, bike.ID, "", "", 0, chatRecentEventsMax)
		if err != nil {
			return nil, err
		}
		recent := make([]chatEvent, 0, len(events.Data))
		for _, e := range events.Data {
			recent = append(recent, chatEvent{Type: e.EventType, RecordedAt: formatChatTime(e.RecordedAt)})
		}
		return activityFacts{chatBase: base, RecentEvents: recent}, nil
	}

	return nil, invalidChatPath()
}

// ---- Theft reports and contacts ----

func (s *ChatService) incidentsRoute(ctx context.Context, claims *security.JWTClaims, lang string, rest []string) (*domain.ChatResponse, error) {
	result, err := s.recovery.ListIncidents(ctx, claims.UserID, chatMaxIncidents, 0)
	if err != nil {
		return nil, err
	}

	if len(rest) == 0 {
		if len(result.Data) == 0 {
			return answerResponse(chatText(lang, "noIncidents")), nil
		}
		options := make([]domain.ChatOption, 0, len(result.Data))
		for _, it := range result.Data {
			options = append(options, domain.ChatOption{
				ID:    "incident:" + it.ID,
				Label: incidentLabel(lang, it),
			})
		}
		return menuResponse(chatText(lang, "incidentPrompt"), options), nil
	}

	id, ok := singleToken(rest, "incident:")
	if !ok {
		return nil, invalidChatPath()
	}
	for _, it := range result.Data {
		if it.ID == id {
			return answerResponse(incidentText(lang, it)), nil
		}
	}
	return nil, domain.NewNotFoundError("Theft report not found", chatInstance)
}

func (s *ChatService) contactsRoute(ctx context.Context, claims *security.JWTClaims, lang string, rest []string) (*domain.ChatResponse, error) {
	contacts, err := s.recovery.ListContacts(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if len(contacts) > chatMaxContacts {
		contacts = contacts[:chatMaxContacts]
	}

	if len(rest) == 0 {
		if len(contacts) == 0 {
			return answerResponse(chatText(lang, "noContacts")), nil
		}
		options := make([]domain.ChatOption, 0, len(contacts))
		for _, c := range contacts {
			options = append(options, domain.ChatOption{
				ID:    "contact:" + c.ID,
				Label: c.Name + " · " + contactCategoryLabel(lang, c.Category),
			})
		}
		return menuResponse(chatText(lang, "contactPrompt"), options), nil
	}

	id, ok := singleToken(rest, "contact:")
	if !ok {
		return nil, invalidChatPath()
	}
	for _, c := range contacts {
		if c.ID == id {
			return answerResponse(contactText(lang, c)), nil
		}
	}
	return nil, domain.NewNotFoundError("Contact not found", chatInstance)
}

// ---- Emergency numbers and police stations ----

func emergencyRoute(lang string, rest []string) (*domain.ChatResponse, error) {
	if len(rest) == 0 {
		options := []domain.ChatOption{
			{ID: "number:national", Label: chatText(lang, "number:national")},
			{ID: "number:redcross", Label: chatText(lang, "number:redcross")},
		}
		if support := strings.TrimSpace(os.Getenv("EMERGENCY_SUPPORT_PHONE")); support != "" {
			options = append(options, domain.ChatOption{ID: "number:support", Label: chatText(lang, "number:support")})
		}
		for i, st := range kenyaPoliceStations {
			options = append(options, domain.ChatOption{ID: "station:" + strconv.Itoa(i), Label: st.Name})
		}
		return menuResponse(chatText(lang, "emergencyPrompt"), options), nil
	}

	token := rest[0]
	if len(rest) != 1 {
		return nil, invalidChatPath()
	}

	if key, ok := strings.CutPrefix(token, "number:"); ok {
		switch key {
		case "national", "redcross":
			return answerResponse(chatText(lang, "answer:"+key)), nil
		case "support":
			support := strings.TrimSpace(os.Getenv("EMERGENCY_SUPPORT_PHONE"))
			if support == "" {
				return nil, invalidChatPath()
			}
			return answerResponse(fmt.Sprintf(chatText(lang, "answer:support"), support)), nil
		}
		return nil, invalidChatPath()
	}

	if idx, ok := strings.CutPrefix(token, "station:"); ok {
		i, err := strconv.Atoi(idx)
		if err != nil || i < 0 || i >= len(kenyaPoliceStations) {
			return nil, invalidChatPath()
		}
		st := kenyaPoliceStations[i]
		return answerResponse(fmt.Sprintf(chatText(lang, "answer:station"), st.Name, st.Phone)), nil
	}

	return nil, invalidChatPath()
}

// ---- Terms and Privacy Policy ----

func clauseRoute(lang string, clauses []content.Clause, rest []string) (*domain.ChatResponse, error) {
	if len(rest) == 0 {
		options := make([]domain.ChatOption, 0, len(clauses))
		for i, c := range clauses {
			options = append(options, domain.ChatOption{
				ID:    "clause:" + strconv.Itoa(i+1),
				Label: fmt.Sprintf("%d. %s", i+1, c.Title),
			})
		}
		return menuResponse(chatText(lang, "clausePrompt"), options), nil
	}

	n, ok := singleToken(rest, "clause:")
	if !ok {
		return nil, invalidChatPath()
	}
	i, err := strconv.Atoi(n)
	if err != nil || i < 1 || i > len(clauses) {
		return nil, invalidChatPath()
	}
	c := clauses[i-1]
	title, body := c.Title, c.Body
	if lang == "sw" {
		title, body = c.TitleSW, c.BodySW
	}
	text := fmt.Sprintf("%d. %s\n\n%s", i, title, body)
	if lang == "sw" {
		text = chatText(lang, "legalNote") + "\n\n" + text
	}
	return answerResponse(text), nil
}

// ---- Facts and fallback wording for bike answers ----

// chatBase is the information every bike answer shares, including how fresh the data is.
type chatBase struct {
	RequestedInformation string `json:"requested_information"`
	RegistrationNumber   string `json:"registration_number"`
	Label                string `json:"label,omitempty"`
	LastUpdate           string `json:"last_update,omitempty"`
	MinutesSinceUpdate   *int   `json:"minutes_since_last_update"`
	MayBeOutdated        bool   `json:"may_be_outdated"`
}

func newChatBase(bike *domain.BikeDTO, action string, now time.Time) chatBase {
	base := chatBase{
		RequestedInformation: action,
		RegistrationNumber:   bike.RegistrationNumber,
		Label:                bike.Label,
		MayBeOutdated:        true, // unknown update time counts as outdated
	}
	if bike.LastTelemetryAt != nil {
		age := now.Sub(*bike.LastTelemetryAt)
		minutes := int(age.Minutes())
		base.LastUpdate = formatChatTime(*bike.LastTelemetryAt)
		base.MinutesSinceUpdate = &minutes
		base.MayBeOutdated = age > chatStaleAfter
	}
	return base
}

type locationFacts struct {
	chatBase
	Address   string  `json:"address"`
	SubCounty string  `json:"sub_county"`
	County    string  `json:"county"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Ignition  bool    `json:"ignition_on"`
}

type statusFacts struct {
	chatBase
	Ignition         bool    `json:"ignition_on"`
	SpeedKMH         float64 `json:"speed_kmh"`
	BatteryPercent   int     `json:"battery_percent"`
	SignalQualityCSQ int     `json:"signal_quality_csq"`
	IsImmobilized    bool    `json:"engine_immobilized"`
	Status           string  `json:"status"`
}

type chatEvent struct {
	Type       string `json:"event_type"`
	RecordedAt string `json:"recorded_at"`
}

type activityFacts struct {
	chatBase
	RecentEvents []chatEvent `json:"recent_events"`
}

// fallbackAnswer is used when OpenAI is not configured or fails. It states only the same facts.
func fallbackAnswer(lang string, facts any) string {
	sw := lang == "sw"
	switch f := facts.(type) {
	case locationFacts:
		place := f.Address
		if place == "" {
			place = fmt.Sprintf("GPS %.5f, %.5f", f.Latitude, f.Longitude)
		}
		if area := joinNonEmpty(", ", f.SubCounty, f.County); area != "" && f.Address != "" {
			place += ", " + area
		}
		msg := pick(sw, fmt.Sprintf("%s was last reported at %s.", bikeNameFor(f.chatBase, sw), place),
			fmt.Sprintf("%s ilionekana mara ya mwisho katika %s.", bikeNameFor(f.chatBase, sw), place))
		return msg + updateNote(sw, f.chatBase)

	case statusFacts:
		ignition := pick(sw, "off", "imezimwa")
		if f.Ignition {
			ignition = pick(sw, "on", "imewashwa")
		}
		var msg string
		if sw {
			msg = fmt.Sprintf("%s: injini %s, kasi %.0f km/h, betri %d%%, mawimbi %d/31.",
				bikeNameFor(f.chatBase, sw), ignition, f.SpeedKMH, f.BatteryPercent, f.SignalQualityCSQ)
			if f.IsImmobilized {
				msg += " Injini imezuiwa."
			}
		} else {
			msg = fmt.Sprintf("%s: ignition is %s, speed %.0f km/h, battery %d%%, signal %d/31.",
				bikeNameFor(f.chatBase, sw), ignition, f.SpeedKMH, f.BatteryPercent, f.SignalQualityCSQ)
			if f.IsImmobilized {
				msg += " Engine is immobilized."
			}
		}
		return msg + updateNote(sw, f.chatBase)

	case activityFacts:
		name := bikeNameFor(f.chatBase, sw)
		if len(f.RecentEvents) == 0 {
			return pick(sw, fmt.Sprintf("No recent activity recorded for %s.", name),
				fmt.Sprintf("Hakuna shughuli za hivi karibuni kwa %s.", name))
		}
		items := make([]string, 0, len(f.RecentEvents))
		for _, e := range f.RecentEvents {
			items = append(items, fmt.Sprintf("%s (%s)", e.Type, e.RecordedAt))
		}
		return pick(sw, fmt.Sprintf("Recent activity for %s: %s.", name, strings.Join(items, "; ")),
			fmt.Sprintf("Shughuli za hivi karibuni za %s: %s.", name, strings.Join(items, "; ")))
	}
	return pick(sw, "Sorry, I could not get that information.", "Samahani, sikuweza kupata taarifa hiyo.")
}

func bikeDetailsText(lang string, b *domain.BikeDTO) string {
	sw := lang == "sw"
	model := joinNonEmpty(" ", b.Make, b.Model)
	year := ""
	if b.Year > 0 {
		year = strconv.Itoa(b.Year)
	}
	tracker := ""
	if len(b.IMEI) >= 4 {
		tracker = "..." + b.IMEI[len(b.IMEI)-4:]
	}
	status := b.Status
	if b.IsImmobilized {
		status = joinNonEmpty(" · ", status, pick(sw, "engine immobilized", "injini imezuiwa"))
	}
	return fmt.Sprintf(pick(sw,
		"Registration: %s\nMake and model: %s\nYear: %s\nColour: %s\nTracker ID ends in: %s\nStatus: %s",
		"Usajili: %s\nAina: %s\nMwaka: %s\nRangi: %s\nKifaa kinaishia na: %s\nHali: %s"),
		orUnknown(sw, b.RegistrationNumber), orUnknown(sw, model), orUnknown(sw, year),
		orUnknown(sw, b.Color), orUnknown(sw, tracker), orUnknown(sw, status))
}

func incidentLabel(lang string, it domain.TheftIncidentDTO) string {
	return modeLabel(lang, it.Mode) + " · " + it.LostAt.In(chatLocation).Format("02 Jan 2006")
}

func incidentText(lang string, it domain.TheftIncidentDTO) string {
	sw := lang == "sw"
	ob := pick(sw, "not yet added", "bado haijawekwa")
	if it.OBNumber != "" {
		ob = it.OBNumber
	}
	return fmt.Sprintf(pick(sw,
		"Theft report: %s\nLost: %s\nPlace: %s\nOB number: %s\nStatus: open (not yet found)",
		"Ripoti ya wizi: %s\nIlipotea: %s\nMahali: %s\nNambari ya OB: %s\nHali: bado haijapatikana"),
		modeLabel(lang, it.Mode), formatChatTime(it.LostAt), orUnknown(sw, it.LocationLost), ob)
}

func modeLabel(lang, mode string) string {
	labels := map[string][2]string{
		"violent_robbery":    {"Violent robbery", "Wizi kwa nguvu"},
		"theft_from_parking": {"Theft from parking", "Wizi kutoka maegesho"},
		"theft_unattended":   {"Theft while unattended", "Wizi wakati hayupo"},
		"hijacking":          {"Hijacking", "Utekaji"},
		"other":              {"Other", "Nyingine"},
	}
	if l, ok := labels[mode]; ok {
		return pick(lang == "sw", l[0], l[1])
	}
	return mode
}

func contactText(lang string, c domain.UserContactDTO) string {
	return fmt.Sprintf("%s\n%s: %s", c.Name, contactCategoryLabel(lang, c.Category), c.Phone)
}

func contactCategoryLabel(lang, category string) string {
	if category == "insurance" {
		return pick(lang == "sw", "Insurance", "Bima")
	}
	return pick(lang == "sw", "Emergency contact", "Mtu wa dharura")
}

func updateNote(sw bool, base chatBase) string {
	if base.LastUpdate == "" {
		return pick(sw, " The bike has not reported its location yet.", " Pikipiki bado haijatuma taarifa zake.")
	}
	note := pick(sw, fmt.Sprintf(" Last update: %s.", base.LastUpdate), fmt.Sprintf(" Sasisho la mwisho: %s.", base.LastUpdate))
	if base.MayBeOutdated {
		note += pick(sw, " This may be out of date.", " Taarifa hii huenda isiwe ya sasa.")
	}
	return note
}

func bikeNameFor(base chatBase, sw bool) string {
	if base.RegistrationNumber != "" {
		return base.RegistrationNumber
	}
	return pick(sw, "Your bike", "Pikipiki yako")
}

// bikeDisplayName is the label riders see in menus, e.g. "Nyayo · KDA 123A".
func bikeDisplayName(b *domain.BikeDTO) string {
	name := b.Label
	if name == "" {
		name = b.RegistrationNumber
	}
	if b.RegistrationNumber != "" && name != b.RegistrationNumber {
		return name + " · " + b.RegistrationNumber
	}
	if name == "" {
		return b.ID
	}
	return name
}

// ---- Shared helpers ----

func menuResponse(message string, options []domain.ChatOption) *domain.ChatResponse {
	if options == nil {
		options = []domain.ChatOption{}
	}
	return &domain.ChatResponse{Type: "menu", Message: message, Options: options}
}

func answerResponse(message string) *domain.ChatResponse {
	return &domain.ChatResponse{Type: "answer", Message: message, Options: []domain.ChatOption{}}
}

func singleToken(rest []string, prefix string) (string, bool) {
	if len(rest) != 1 {
		return "", false
	}
	value, ok := strings.CutPrefix(rest[0], prefix)
	return value, ok && value != ""
}

func orUnknown(sw bool, value string) string {
	if strings.TrimSpace(value) == "" {
		return pick(sw, "not recorded", "haijarekodiwa")
	}
	return value
}

func truncateLabel(label string) string {
	label = strings.TrimSpace(label)
	if utf8.RuneCountInString(label) <= chatMaxLabelRunes {
		return label
	}
	return string([]rune(label)[:chatMaxLabelRunes])
}

func formatChatTime(t time.Time) string {
	return t.In(chatLocation).Format("02 Jan 2006 15:04")
}

func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func pick(sw bool, en, swahili string) string {
	if sw {
		return swahili
	}
	return en
}

func languageOf(lang string) string {
	if isSupportedChatLanguage(lang) {
		return lang
	}
	return "en"
}

var chatMessages = map[string]map[string]string{
	"en": {
		"topPrompt":         "What would you like to check?",
		"section:bikes":     "My bikes",
		"section:incidents": "Theft reports",
		"section:contacts":  "My contacts",
		"section:emergency": "Emergency numbers & police stations",
		"section:terms":     "Terms & conditions",
		"section:privacy":   "Privacy policy",
		"noBikes":           "No motorcycles are linked to your account yet.",
		"bikePrompt":        "Which motorcycle would you like information about?",
		"actionPrompt":      "What would you like to know about %s?",
		"action:location":   "Where is it now?",
		"action:status":     "Bike status (ignition, battery, signal)",
		"action:activity":   "Recent activity",
		"action:details":    "Bike details",
		"noIncidents":       "You have no open theft reports.",
		"incidentPrompt":    "Which theft report would you like to see?",
		"noContacts":        "You have not saved any contacts yet. You can add them in the app under Contacts.",
		"contactPrompt":     "Which contact would you like to see?",
		"emergencyPrompt":   "Choose an emergency number or a police station.",
		"number:national":   "Police emergency (999 / 112)",
		"number:redcross":   "Ambulance (1199)",
		"number:support":    "MGuard support",
		"answer:national":   "Police emergency: call 999 or 112.",
		"answer:redcross":   "Ambulance: call 1199 (Kenya Red Cross).",
		"answer:support":    "MGuard support: %s.",
		"answer:station":    "%s: %s.",
		"clausePrompt":      "Choose a section to read.",
		"legalNote":         "This text is currently provided in English only.",
	},
	"sw": {
		"topPrompt":         "Ungependa kuangalia nini?",
		"section:bikes":     "Pikipiki zangu",
		"section:incidents": "Ripoti za wizi",
		"section:contacts":  "Watu wangu",
		"section:emergency": "Namba za dharura na vituo vya polisi",
		"section:terms":     "Masharti ya matumizi",
		"section:privacy":   "Sera ya faragha",
		"noBikes":           "Hakuna pikipiki iliyounganishwa na akaunti yako bado.",
		"bikePrompt":        "Ungependa taarifa ya pikipiki ipi?",
		"actionPrompt":      "Ungependa kujua nini kuhusu %s?",
		"action:location":   "Iko wapi sasa?",
		"action:status":     "Hali ya pikipiki (injini, betri, mawimbi)",
		"action:activity":   "Shughuli za hivi karibuni",
		"action:details":    "Maelezo ya pikipiki",
		"noIncidents":       "Huna ripoti za wizi zilizo wazi.",
		"incidentPrompt":    "Ungependa kuona ripoti ipi ya wizi?",
		"noContacts":        "Bado hujahifadhi watu wowote. Unaweza kuwaongeza kwenye programu chini ya Contacts.",
		"contactPrompt":     "Ungependa kuona mtu yupi?",
		"emergencyPrompt":   "Chagua namba ya dharura au kituo cha polisi.",
		"number:national":   "Polisi dharura (999 / 112)",
		"number:redcross":   "Ambulensi (1199)",
		"number:support":    "Msaada wa MGuard",
		"answer:national":   "Polisi dharura: piga 999 au 112.",
		"answer:redcross":   "Ambulensi: piga 1199 (Msalaba Mwekundu wa Kenya).",
		"answer:support":    "Msaada wa MGuard: %s.",
		"answer:station":    "%s: %s.",
		"clausePrompt":      "Chagua sehemu ya kusoma.",
		"legalNote":         "Tafsiri hii ni rasimu na inasubiri ukaguzi wa kisheria.",
	},
}

func chatText(lang, key string) string {
	if msg, ok := chatMessages[languageOf(lang)][key]; ok {
		return msg
	}
	return chatMessages["en"][key]
}

func isSupportedChatLanguage(lang string) bool {
	return lang == "en" || lang == "sw"
}

func isBikeFactAction(action string) bool {
	switch action {
	case "location", "status", "activity":
		return true
	}
	return false
}

func invalidChatPath() error {
	return domain.NewBadRequestError("Invalid menu selection", chatInstance, nil)
}
