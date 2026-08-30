package tgbot

type Tariff struct {
	ID           string
	InboundID    int
	TotalGB      int64
	LimitHWID    int
	MonthlyPrice int64
}

type TariffPeriod struct {
	Months   int
	Discount int
}

const defaultRegistrationInboundID = 1

var tariffs = []Tariff{
	{
		ID:           "30gb_1",
		InboundID:    defaultRegistrationInboundID,
		TotalGB:      30,
		LimitHWID:    1,
		MonthlyPrice: 50,
	},
	{
		ID:           "50gb_3",
		InboundID:    defaultRegistrationInboundID,
		TotalGB:      50,
		LimitHWID:    3,
		MonthlyPrice: 100,
	},
	{
		ID:           "100gb_5",
		InboundID:    defaultRegistrationInboundID,
		TotalGB:      100,
		LimitHWID:    5,
		MonthlyPrice: 200,
	},
	{
		ID:           "300gb_10",
		InboundID:    defaultRegistrationInboundID,
		TotalGB:      300,
		LimitHWID:    10,
		MonthlyPrice: 300,
	},
}

var tariffPeriods = []TariffPeriod{
	{
		Months:   1,
		Discount: 0,
	},
	{
		Months:   3,
		Discount: 10,
	},
	{
		Months:   6,
		Discount: 20,
	},
	{
		Months:   12,
		Discount: 30,
	},
}

func (t Tariff) Price(period TariffPeriod) int64 {
	price := t.MonthlyPrice * int64(period.Months)
	return price * int64(100-period.Discount) / 100
}
