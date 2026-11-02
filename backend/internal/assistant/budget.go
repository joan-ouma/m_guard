package assistant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// ErrDailyBudgetExhausted is returned once the service-wide daily OpenAI allowance is used up.
var ErrDailyBudgetExhausted = errors.New("daily AI budget exhausted")

// Composer is anything that can phrase a chatbot reply from facts.
type Composer interface {
	Compose(ctx context.Context, language string, facts any) (string, error)
}

// UsageStore counts OpenAI calls per day. TakeAIUsage reserves one call if today's total is below limit.
type UsageStore interface {
	TakeAIUsage(ctx context.Context, day string, limit int) (bool, error)
}

var eat = time.FixedZone("EAT", 3*60*60)

// BudgetedComposer caps how many OpenAI calls the whole service makes per day, across all riders
// and across restarts. The count is stored in the database. If the count cannot be read, no call is made.
type BudgetedComposer struct {
	inner Composer
	store UsageStore
	limit int
	clock func() time.Time
}

func NewBudgetedComposer(inner Composer, store UsageStore, dailyLimit int) *BudgetedComposer {
	return &BudgetedComposer{
		inner: inner,
		store: store,
		limit: dailyLimit,
		clock: time.Now,
	}
}

func (b *BudgetedComposer) Compose(ctx context.Context, language string, facts any) (string, error) {
	day := b.clock().In(eat).Format("2006-01-02")
	ok, err := b.store.TakeAIUsage(ctx, day, b.limit)
	if err != nil {
		slog.Error("chat AI budget check failed; using built-in wording", "error", err)
		return "", fmt.Errorf("budget check: %w", err)
	}
	if !ok {
		return "", ErrDailyBudgetExhausted
	}
	return b.inner.Compose(ctx, language, facts)
}
