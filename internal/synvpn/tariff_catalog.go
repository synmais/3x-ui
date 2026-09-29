package synvpn

import "strings"

func Tariffs() []Tariff {
	return append([]Tariff(nil), tariffs...)
}

func TariffsForEmail(email string) []Tariff {
	catalog := Tariffs()
	if IsLoyaltyTariffUser(email) {
		catalog = append(catalog, loyaltyTariff)
	}
	return catalog
}

func RegistrationPromoTariff() Tariff {
	return registrationPromoTariff
}

func LoyaltyTariff() Tariff {
	return loyaltyTariff
}

func IsLoyaltyTariffUser(email string) bool {
	_, ok := loyaltyTariffUsers[strings.ToLower(strings.TrimSpace(email))]
	return ok
}

func FindTariff(id string) *Tariff {
	for i := range tariffs {
		if tariffs[i].ID == id {
			return &tariffs[i]
		}
	}
	if id == registrationPromoTariff.ID {
		return &registrationPromoTariff
	}
	if id == loyaltyTariff.ID {
		return &loyaltyTariff
	}
	return nil
}

func TariffPeriods() []TariffPeriod {
	return append([]TariffPeriod(nil), tariffPeriods...)
}

func FindPeriod(months int) *TariffPeriod {
	for i := range tariffPeriods {
		if tariffPeriods[i].Months == months {
			return &tariffPeriods[i]
		}
	}
	return nil
}

func CalculatePrice(tariff Tariff, period TariffPeriod) int64 {
	if tariff.Days > 0 {
		return tariff.Price
	}
	price := tariff.MonthlyPrice * int64(period.Months)
	return price * int64(100-period.Discount) / 100
}

func IsDayTariff(tariff Tariff) bool {
	return tariff.Days > 0
}
