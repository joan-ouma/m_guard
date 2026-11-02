package assistant

import (
	"context"
	"errors"
	"testing"
	"time"
)

type countingComposer struct{ calls int }

func (c *countingComposer) Compose(_ context.Context, _ string, _ any) (string, error) {
	c.calls++
	return "ok", nil
}

// memoryUsage mirrors the SQL in repository.TakeAIUsage: one counter per day, refused at the limit.
type memoryUsage struct {
	counts map[string]int
	err    error
}

func (m *memoryUsage) TakeAIUsage(_ context.Context, day string, limit int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if limit <= 0 || m.counts[day] >= limit {
		return false, nil
	}
	m.counts[day]++
	return true, nil
}

func TestBudgetedComposerStopsAtDailyLimitAndResetsNextDay(t *testing.T) {
	inner := &countingComposer{}
	store := &memoryUsage{counts: map[string]int{}}
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	b := NewBudgetedComposer(inner, store, 2)
	b.clock = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if _, err := b.Compose(context.Background(), "en", nil); err != nil {
			t.Fatalf("call %d within budget failed: %v", i, err)
		}
	}
	if _, err := b.Compose(context.Background(), "en", nil); !errors.Is(err, ErrDailyBudgetExhausted) {
		t.Fatalf("expected budget exhausted, got %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("inner composer should only be called within budget, got %d", inner.calls)
	}

	now = now.Add(24 * time.Hour)
	if _, err := b.Compose(context.Background(), "en", nil); err != nil {
		t.Fatalf("budget should reset the next day: %v", err)
	}
}

func TestBudgetedComposerMakesNoCallWhenStoreFails(t *testing.T) {
	inner := &countingComposer{}
	b := NewBudgetedComposer(inner, &memoryUsage{err: errors.New("db down")}, 10)

	if _, err := b.Compose(context.Background(), "en", nil); err == nil {
		t.Fatal("expected an error when the budget cannot be checked")
	}
	if inner.calls != 0 {
		t.Fatalf("no OpenAI call should be made when the budget is unknown, got %d", inner.calls)
	}
}
