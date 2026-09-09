package tgbot

import "time"

func calculateSubscriptionExpiry(currentExpiry int64, months int, now time.Time) int64 {
	base := now

	if currentExpiry > now.UnixMilli() {
		base = time.UnixMilli(currentExpiry)
	}

	return base.AddDate(0, months, 0).UnixMilli()
}
