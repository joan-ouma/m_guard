package sms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"0712 345 678", "+254712345678", true},
		{"0712-345-678", "+254712345678", true},
		{"+254 712 345 678", "+254712345678", true},
		{"254712345678", "+254712345678", true},
		{"00254712345678", "+254712345678", true},
		{"712345678", "+254712345678", true},
		{"", "", false},
		{"12", "", false},
		{"abc123456", "", false},
		{"+25471234567890123456", "", false},
	}
	for _, tc := range cases {
		got, ok := NormalizePhone(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("NormalizePhone(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestSendPostsToAllRecipientsAndParsesResults(t *testing.T) {
	var gotForm url.Values
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		gotKey = r.Header.Get("apiKey")
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"SMSMessageData":{"Message":"Sent","Recipients":[` +
			`{"statusCode":101,"number":"+254712345678","status":"Success","messageId":"ATXid_1"},` +
			`{"statusCode":406,"number":"+254700000000","status":"UserInBlacklist","messageId":"None"}]}}`))
	}))
	defer srv.Close()

	c := NewAfricasTalkingClient("myuser", "secret-key", "MGUARD")
	c.endpoint = srv.URL

	results, err := c.Send(context.Background(), []string{"+254712345678", "+254700000000"}, "Jane has lost their bike KDA 123A.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "secret-key" {
		t.Errorf("expected apiKey header, got %q", gotKey)
	}
	if gotForm.Get("username") != "myuser" || gotForm.Get("from") != "MGUARD" {
		t.Errorf("unexpected username/from: %v", gotForm)
	}
	if gotForm.Get("to") != "+254712345678,+254700000000" {
		t.Errorf("unexpected to: %q", gotForm.Get("to"))
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].IsAccepted() {
		t.Errorf("expected first recipient accepted")
	}
	if results[1].IsAccepted() {
		t.Errorf("expected blacklisted recipient rejected")
	}
}

func TestSendReturnsErrorOnHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "InvalidApiKey", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewAfricasTalkingClient("myuser", "bad", "")
	c.endpoint = srv.URL

	_, err := c.Send(context.Background(), []string{"+254712345678"}, "hi")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected a 401 error, got %v", err)
	}
}

func TestSendWithNoRecipientsDoesNothing(t *testing.T) {
	c := NewAfricasTalkingClient("myuser", "key", "")
	c.endpoint = "http://127.0.0.1:1" // would fail if a request were made
	results, err := c.Send(context.Background(), nil, "hi")
	if err != nil || results != nil {
		t.Fatalf("expected no-op, got %v, %v", results, err)
	}
}
