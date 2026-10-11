package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseLimit(t *testing.T) {
	q := url.Values{}
	if got := parseLimit(q, 1000); got != 1000 {
		t.Errorf("expected 1000, got %d", got)
	}

	q.Set("limit", "50")
	if got := parseLimit(q, 1000); got != 50 {
		t.Errorf("expected 50, got %d", got)
	}

	q.Set("limit", "invalid")
	if got := parseLimit(q, 1000); got != 1000 {
		t.Errorf("expected 1000 for invalid, got %d", got)
	}
}

func TestParseDateRange(t *testing.T) {
	q := url.Values{}
	q.Set("date", "2026-09-29")
	start, end := parseDateRange(q)
	if start != "2026-09-29T00:00:00Z" || end != "2026-09-29T23:59:59Z" {
		t.Errorf("unexpected date range: %s to %s", start, end)
	}

	q2 := url.Values{}
	q2.Set("startDate", "2026-09-01")
	q2.Set("endDate", "2026-09-10")
	start2, end2 := parseDateRange(q2)
	if start2 != "2026-09-01T00:00:00Z" || end2 != "2026-09-10T23:59:59Z" {
		t.Errorf("unexpected custom range: %s to %s", start2, end2)
	}
}

func TestGetBikeID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/bikes/bik_123/breadcrumbs", nil)
	id := getBikeID(req)
	if id != "bik_123" {
		t.Errorf("expected bik_123, got %s", id)
	}
}
