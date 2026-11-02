package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIComposerSendsFactsAndReturnsReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected auth header %q", got)
		}

		var body chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.Model != "gpt-4o-mini" {
			t.Errorf("unexpected model %q", body.Model)
		}
		if len(body.Messages) != 2 || !strings.Contains(body.Messages[0].Content, "Kiswahili") {
			t.Errorf("expected Swahili system prompt, got %+v", body.Messages)
		}
		if !strings.Contains(body.Messages[1].Content, "KDA 123A") {
			t.Errorf("facts not sent to model: %s", body.Messages[1].Content)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "  Pikipiki yako iko Dagoretti.  "}}},
		})
	}))
	defer server.Close()

	c := NewOpenAIComposer("test-key", "", server.URL)
	reply, err := c.Compose(context.Background(), "sw", map[string]string{"registration_number": "KDA 123A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Pikipiki yako iko Dagoretti." {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestOpenAIComposerReturnsErrorOnNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c := NewOpenAIComposer("test-key", "gpt-4o-mini", server.URL)
	if _, err := c.Compose(context.Background(), "en", map[string]string{}); err == nil {
		t.Fatal("expected error on 429 response")
	}
}

func TestOpenAIComposerRejectsEmptyReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": "   "}}}})
	}))
	defer server.Close()

	c := NewOpenAIComposer("test-key", "", server.URL)
	if _, err := c.Compose(context.Background(), "en", map[string]string{}); err == nil {
		t.Fatal("expected error for empty reply")
	}
}
