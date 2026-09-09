package tgbot

import (
	"testing"
	"time"
)

func TestCalculateSubscriptionExpiry(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	t.Run("active subscription extends from current expiry", func(t *testing.T) {
		currentExpiry := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

		got := calculateSubscriptionExpiry(
			currentExpiry.UnixMilli(),
			3,
			now,
		)

		want := time.Date(2026, 12, 20, 12, 0, 0, 0, time.UTC).UnixMilli()

		if got != want {
			t.Fatalf("expected %d, got %d", want, got)
		}
	})

	t.Run("expired subscription starts from now", func(t *testing.T) {
		currentExpiry := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

		got := calculateSubscriptionExpiry(
			currentExpiry.UnixMilli(),
			3,
			now,
		)

		want := time.Date(2026, 12, 9, 12, 0, 0, 0, time.UTC).UnixMilli()

		if got != want {
			t.Fatalf("expected %d, got %d", want, got)
		}
	})

	t.Run("zero expiry starts from now", func(t *testing.T) {
		got := calculateSubscriptionExpiry(0, 1, now)

		want := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC).UnixMilli()

		if got != want {
			t.Fatalf("expected %d, got %d", want, got)
		}
	})
}
