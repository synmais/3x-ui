package synvpn

func Tariffs() []Tariff {
	return append([]Tariff(nil), tariffs...)
}

func FindTariff(id string) *Tariff {
	for i := range tariffs {
		if tariffs[i].ID == id {
			return &tariffs[i]
		}
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
	price := tariff.MonthlyPrice * int64(period.Months)
	return price * int64(100-period.Discount) / 100
}
