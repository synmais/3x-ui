package synvpn

import "time"

func CalculateSubscriptionExpiry(currentExpiry int64, months int, now time.Time) int64 {
	base := now

	if currentExpiry > now.UnixMilli() {
		base = time.UnixMilli(currentExpiry)
	}

	return base.AddDate(0, months, 0).UnixMilli()
}

func FindTariffForRecord(totalGB int64, limitHWID int) *Tariff {
	for i := range tariffs {
		if tariffs[i].TotalGB*1024*1024*1024 == totalGB &&
			tariffs[i].LimitHWID == limitHWID {
			return &tariffs[i]
		}
	}

	return nil
}

// ConvertedTariffDays converts remaining subscription days from the old tariff
// into an equivalent number of days at the new tariff.
// A month is priced as 30 days so the rule is predictable to subscribers.
func ConvertedTariffDays(expiry int64, oldTariff, newTariff Tariff, now time.Time) int {
	if expiry <= now.UnixMilli() || oldTariff.MonthlyPrice <= 0 || newTariff.MonthlyPrice <= 0 {
		return 0
	}

	remaining := time.UnixMilli(expiry).Sub(now)
	days := int((remaining + 24*time.Hour - 1) / (24 * time.Hour))

	return days * int(oldTariff.MonthlyPrice) / int(newTariff.MonthlyPrice)
}

func CalculatePurchaseCarryover(
	kind PurchaseKind,
	expiry int64,
	currentTariff, newTariff *Tariff,
	now time.Time,
) int {
	if kind != PurchaseRenew || currentTariff == nil || newTariff == nil {
		return 0
	}

	if currentTariff.ID == newTariff.ID {
		return 0
	}

	return ConvertedTariffDays(expiry, *currentTariff, *newTariff, now)
}
