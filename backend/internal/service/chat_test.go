package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/security"
)

type fakeChatBikes struct {
	bikes  []domain.BikeDTO
	events []domain.BikeEventDTO
}

func (f *fakeChatBikes) ListBikes(_ context.Context, _ *security.JWTClaims, _ string, _ int) (*domain.BikeListResponse, error) {
	return &domain.BikeListResponse{Data: f.bikes}, nil
}

func (f *fakeChatBikes) GetBike(_ context.Context, _ *security.JWTClaims, bikeID string) (*domain.BikeDTO, error) {
	for i := range f.bikes {
		if f.bikes[i].ID == bikeID {
			return &f.bikes[i], nil
		}
	}
	return nil, domain.NewNotFoundError("Motorcycle not found", "/v1/chat")
}

func (f *fakeChatBikes) GetEvents(_ context.Context, _ *security.JWTClaims, _, _, _ string, _ int64, _ int) (*domain.PaginatedEventsResponse, error) {
	return &domain.PaginatedEventsResponse{Data: f.events}, nil
}

type fakeRecovery struct {
	incidents []domain.TheftIncidentDTO
	contacts  []domain.UserContactDTO
}

func (f *fakeRecovery) ListIncidents(_ context.Context, _ string, _, _ int) (*IncidentListResult, error) {
	return &IncidentListResult{Data: f.incidents}, nil
}

func (f *fakeRecovery) ListContacts(_ context.Context, _ string) ([]domain.UserContactDTO, error) {
	return f.contacts, nil
}

type fakeHistory struct {
	items []domain.ChatHistoryItem
}

func (f *fakeHistory) Append(_ context.Context, _, role, text string) error {
	f.items = append(f.items, domain.ChatHistoryItem{ID: int64(len(f.items) + 1), Role: role, Text: text, CreatedAt: time.Now()})
	return nil
}

func (f *fakeHistory) Recent(_ context.Context, _ string, limit int) ([]domain.ChatHistoryItem, error) {
	if limit > len(f.items) {
		limit = len(f.items)
	}
	return f.items[len(f.items)-limit:], nil
}

func (f *fakeHistory) Clear(_ context.Context, _ string) error {
	f.items = nil
	return nil
}

func (f *fakeHistory) PurgeOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type fakeComposer struct {
	reply string
	err   error
	calls int
	lang  string
}

func (f *fakeComposer) Compose(_ context.Context, language string, _ any) (string, error) {
	f.calls++
	f.lang = language
	return f.reply, f.err
}

var testClaims = &security.JWTClaims{UserID: "user-1"}

func newTestBike(now time.Time, age time.Duration) domain.BikeDTO {
	seen := now.Add(-age)
	return domain.BikeDTO{
		ID:                 "bike-1",
		RegistrationNumber: "KDA 123A",
		Label:              "Nyayo",
		Make:               "Bajaj",
		Model:              "Boxer",
		Year:               2022,
		Color:              "Red",
		IMEI:               "863719040123456",
		CurrentAddress:     "Kawangware Road",
		SubCounty:          "Dagoretti",
		County:             "Nairobi",
		Latitude:           -1.2855,
		Longitude:          36.7428,
		Ignition:           true,
		BatteryPercent:     80,
		SignalQualityCSQ:   20,
		LastTelemetryAt:    &seen,
	}
}

type testRig struct {
	svc      *ChatService
	bikes    *fakeChatBikes
	recovery *fakeRecovery
	history  *fakeHistory
	composer *fakeComposer
}

func newRig(bikes []domain.BikeDTO, composer *fakeComposer) *testRig {
	rig := &testRig{
		bikes:    &fakeChatBikes{bikes: bikes},
		recovery: &fakeRecovery{},
		history:  &fakeHistory{},
		composer: composer,
	}
	var c ChatComposer
	if composer != nil {
		c = composer
	}
	rig.svc = NewChatService(rig.bikes, rig.recovery, rig.history, c)
	return rig
}

func (r *testRig) respond(t *testing.T, path []string, label string) *domain.ChatResponse {
	t.Helper()
	resp, err := r.svc.Respond(context.Background(), testClaims, path, label)
	if err != nil {
		t.Fatalf("unexpected error for %v: %v", path, err)
	}
	return resp
}

func (r *testRig) expectError(t *testing.T, path []string) {
	t.Helper()
	if _, err := r.svc.Respond(context.Background(), testClaims, path, ""); err == nil {
		t.Errorf("expected an error for path %v", path)
	}
}

func TestLanguageAndTopMenus(t *testing.T) {
	rig := newRig(nil, nil)

	lang := rig.respond(t, nil, "")
	if lang.Type != "menu" || len(lang.Options) != 2 || lang.Options[0].ID != "lang:en" {
		t.Fatalf("unexpected language menu: %+v", lang)
	}

	top := rig.respond(t, []string{"lang:en"}, "")
	if len(top.Options) != 6 || top.Options[0].ID != "section:bikes" || top.Options[4].ID != "section:terms" {
		t.Fatalf("unexpected top menu: %+v", top.Options)
	}
}

func TestBikeMenuAndNoBikes(t *testing.T) {
	rig := newRig([]domain.BikeDTO{newTestBike(time.Now(), time.Minute)}, nil)

	resp := rig.respond(t, []string{"lang:en", "section:bikes"}, "")
	if resp.Type != "menu" || len(resp.Options) != 1 || resp.Options[0].ID != "bike:bike-1" {
		t.Fatalf("expected one bike option, got %+v", resp)
	}
	if resp.Options[0].Label != "Nyayo · KDA 123A" {
		t.Fatalf("unexpected bike label: %q", resp.Options[0].Label)
	}

	empty := newRig(nil, nil)
	resp = empty.respond(t, []string{"lang:sw", "section:bikes"}, "")
	if resp.Type != "answer" || !strings.Contains(resp.Message, "Hakuna pikipiki") {
		t.Fatalf("expected Swahili no-bikes answer, got %+v", resp)
	}
}

func TestBikeLocationUsesComposerInRequestedLanguage(t *testing.T) {
	composer := &fakeComposer{reply: "Nyayo ilionekana Kawangware."}
	rig := newRig([]domain.BikeDTO{newTestBike(time.Now(), time.Minute)}, composer)

	resp := rig.respond(t, []string{"lang:sw", "section:bikes", "bike:bike-1", "action:location"}, "Iko wapi sasa?")
	if resp.Type != "answer" || resp.Message != "Nyayo ilionekana Kawangware." {
		t.Fatalf("expected composer reply, got %+v", resp)
	}
	if composer.calls != 1 || composer.lang != "sw" {
		t.Fatalf("expected one composer call in sw, got calls=%d lang=%q", composer.calls, composer.lang)
	}
}

func TestBikeAnswerFallsBackWhenComposerFails(t *testing.T) {
	composer := &fakeComposer{err: errors.New("openai down")}
	rig := newRig([]domain.BikeDTO{newTestBike(time.Now(), time.Minute)}, composer)

	resp := rig.respond(t, []string{"lang:en", "section:bikes", "bike:bike-1", "action:location"}, "")
	if !strings.Contains(resp.Message, "KDA 123A") || !strings.Contains(resp.Message, "Kawangware Road") {
		t.Fatalf("expected English fallback with bike and address, got %q", resp.Message)
	}
}

func TestSwahiliFallbackMarksOutdatedData(t *testing.T) {
	now := time.Now()
	bike := newTestBike(now, 2*time.Hour)
	facts, err := NewChatService(&fakeChatBikes{}, &fakeRecovery{}, &fakeHistory{}, nil).buildFacts(context.Background(), testClaims, &bike, "status", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := fallbackAnswer("sw", facts)
	if !strings.Contains(msg, "Taarifa hii huenda isiwe ya sasa.") || !strings.Contains(msg, "injini imewashwa") {
		t.Fatalf("expected outdated warning and ignition state in Swahili, got %q", msg)
	}
}

func TestBikeActivityAndDetails(t *testing.T) {
	now := time.Now()
	rig := newRig([]domain.BikeDTO{newTestBike(now, time.Minute)}, nil)
	rig.bikes.events = []domain.BikeEventDTO{{EventType: "GEOFENCE_BREACH", RecordedAt: now}}

	resp := rig.respond(t, []string{"lang:en", "section:bikes", "bike:bike-1", "action:activity"}, "")
	if !strings.Contains(resp.Message, "GEOFENCE_BREACH") {
		t.Fatalf("expected event in activity answer, got %q", resp.Message)
	}

	resp = rig.respond(t, []string{"lang:en", "section:bikes", "bike:bike-1", "action:details"}, "")
	if !strings.Contains(resp.Message, "Bajaj Boxer") || !strings.Contains(resp.Message, "...3456") {
		t.Fatalf("expected make, model and tracker suffix in details, got %q", resp.Message)
	}
}

func TestIncidentsOnlyShowOwnedReports(t *testing.T) {
	rig := newRig(nil, nil)
	rig.recovery.incidents = []domain.TheftIncidentDTO{{
		ID: "inc-1", Mode: "theft_unattended", LostAt: time.Now(), LocationLost: "Gate 4", OBNumber: "OB 12/11/2026",
	}}

	menu := rig.respond(t, []string{"lang:en", "section:incidents"}, "")
	if len(menu.Options) != 1 || menu.Options[0].ID != "incident:inc-1" {
		t.Fatalf("expected one incident option, got %+v", menu.Options)
	}

	answer := rig.respond(t, []string{"lang:en", "section:incidents", "incident:inc-1"}, "")
	if !strings.Contains(answer.Message, "OB 12/11/2026") || !strings.Contains(answer.Message, "Gate 4") {
		t.Fatalf("expected OB number and place, got %q", answer.Message)
	}

	rig.expectError(t, []string{"lang:en", "section:incidents", "incident:someone-else"})
}

func TestContactsMenuAndAnswer(t *testing.T) {
	rig := newRig(nil, nil)
	rig.recovery.contacts = []domain.UserContactDTO{{ID: "c-1", Name: "Wanjiru", Phone: "0712 000 111", Category: "emergency"}}

	menu := rig.respond(t, []string{"lang:en", "section:contacts"}, "")
	if len(menu.Options) != 1 || menu.Options[0].ID != "contact:c-1" {
		t.Fatalf("unexpected contacts menu: %+v", menu.Options)
	}

	answer := rig.respond(t, []string{"lang:sw", "section:contacts", "contact:c-1"}, "")
	if !strings.Contains(answer.Message, "0712 000 111") || !strings.Contains(answer.Message, "Mtu wa dharura") {
		t.Fatalf("unexpected contact answer: %q", answer.Message)
	}
}

func TestEmergencyNumbersAndStations(t *testing.T) {
	rig := newRig(nil, nil)

	menu := rig.respond(t, []string{"lang:en", "section:emergency"}, "")
	if menu.Options[0].ID != "number:national" || len(menu.Options) < 3+len(kenyaPoliceStations)-1 {
		t.Fatalf("expected numbers followed by stations, got %d options", len(menu.Options))
	}

	answer := rig.respond(t, []string{"lang:en", "section:emergency", "number:national"}, "")
	if !strings.Contains(answer.Message, "999") {
		t.Fatalf("expected 999 in answer, got %q", answer.Message)
	}

	station := rig.respond(t, []string{"lang:en", "section:emergency", "station:0"}, "")
	if !strings.Contains(station.Message, kenyaPoliceStations[0].Name) {
		t.Fatalf("expected first station in answer, got %q", station.Message)
	}

	rig.expectError(t, []string{"lang:en", "section:emergency", "station:999"})
}

func TestTermsAndPrivacyClauses(t *testing.T) {
	rig := newRig(nil, nil)

	menu := rig.respond(t, []string{"lang:en", "section:terms"}, "")
	if len(menu.Options) != 10 || menu.Options[0].ID != "clause:1" {
		t.Fatalf("expected ten terms clauses, got %d", len(menu.Options))
	}

	clause := rig.respond(t, []string{"lang:en", "section:terms", "clause:1"}, "")
	if !strings.Contains(clause.Message, "Acceptance of Terms") {
		t.Fatalf("unexpected clause text: %q", clause.Message)
	}

	sw := rig.respond(t, []string{"lang:sw", "section:privacy", "clause:4"}, "")
	if !strings.Contains(sw.Message, "MGuard Assistant") || !strings.Contains(sw.Message, "kwa Kiingereza pekee") {
		t.Fatalf("expected assistant paragraph and Swahili note, got %q", sw.Message)
	}

	rig.expectError(t, []string{"lang:en", "section:terms", "clause:11"})
	rig.expectError(t, []string{"lang:en", "section:terms", "clause:0"})
}

func TestAnswersAreRecordedAndMenusAreNot(t *testing.T) {
	rig := newRig(nil, nil)

	rig.respond(t, []string{"lang:en", "section:terms"}, "Terms & conditions")
	if len(rig.history.items) != 0 {
		t.Fatalf("menus should not be stored, got %d rows", len(rig.history.items))
	}

	rig.respond(t, []string{"lang:en", "section:terms", "clause:1"}, "Acceptance of Terms")
	if len(rig.history.items) != 2 || rig.history.items[0].Role != "rider" || rig.history.items[1].Role != "bot" {
		t.Fatalf("expected rider label then bot answer, got %+v", rig.history.items)
	}
	if rig.history.items[0].Text != "Acceptance of Terms" {
		t.Fatalf("rider label not stored: %q", rig.history.items[0].Text)
	}
}

func TestHistoryLimitAndClear(t *testing.T) {
	rig := newRig(nil, nil)
	for i := 0; i < 5; i++ {
		rig.respond(t, []string{"lang:en", "section:emergency", "number:national"}, "Police")
	}

	items, err := rig.svc.History(context.Background(), testClaims, 3)
	if err != nil || len(items) != 3 {
		t.Fatalf("expected 3 items, got %d (err %v)", len(items), err)
	}

	if err := rig.svc.ClearHistory(context.Background(), testClaims); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if items, _ := rig.svc.History(context.Background(), testClaims, 10); len(items) != 0 {
		t.Fatalf("expected empty history after clear, got %d", len(items))
	}
}

func TestRespondRejectsInvalidPaths(t *testing.T) {
	rig := newRig([]domain.BikeDTO{newTestBike(time.Now(), time.Minute)}, nil)

	cases := [][]string{
		{"lang:fr"},
		{"bike:bike-1"},
		{"lang:en", "bike:bike-1"},
		{"lang:en", "section:unknown"},
		{"lang:en", "section:bikes", "bike:"},
		{"lang:en", "section:bikes", "bike:someone-else"},
		{"lang:en", "section:bikes", "bike:bike-1", "action:immobilize"},
		{"lang:en", "section:bikes", "bike:bike-1", "action:location", "extra"},
		{"lang:en", "section:bikes", "bike:bike-1", "action:location", "extra", "more"},
	}
	for _, path := range cases {
		rig.expectError(t, path)
	}
}

func TestRespondRequiresAuthenticatedUser(t *testing.T) {
	rig := newRig(nil, nil)
	if _, err := rig.svc.Respond(context.Background(), nil, nil, ""); err == nil {
		t.Fatal("expected unauthorized error for nil claims")
	}
}
