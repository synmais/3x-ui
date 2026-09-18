package synvpn

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

const DefaultRegistrationInboundID = 1

var tariffs = []Tariff{
	{
		ID:           "30gb_1",
		InboundID:    DefaultRegistrationInboundID,
		TotalGB:      30,
		LimitHWID:    1,
		MonthlyPrice: 50,
	},
	{
		ID:           "50gb_3",
		InboundID:    DefaultRegistrationInboundID,
		TotalGB:      50,
		LimitHWID:    3,
		MonthlyPrice: 100,
	},
	{
		ID:           "100gb_5",
		InboundID:    DefaultRegistrationInboundID,
		TotalGB:      100,
		LimitHWID:    5,
		MonthlyPrice: 200,
	},
	{
		ID:           "300gb_10",
		InboundID:    DefaultRegistrationInboundID,
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
