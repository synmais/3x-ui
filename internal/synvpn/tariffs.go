package synvpn

type Tariff struct {
	ID           string
	InboundIDs   []int
	TotalGB      int64
	LimitHWID    int
	MonthlyPrice int64
	Price        int64
	Days         int
}

type TariffPeriod struct {
	Months   int
	Discount int
}

const (
	DefaultRegistrationInboundID = 1
	RegistrationPromoTariffID    = "registration_2rub"
	LoyaltyTariffID              = "loyalty_300gb_10"
)

var DefaultRegistrationInboundIDs = []int{1, 4}

var registrationPromoTariff = Tariff{
	ID:         RegistrationPromoTariffID,
	InboundIDs: DefaultRegistrationInboundIDs,
	TotalGB:    30,
	LimitHWID:  3,
	Price:      2,
	Days:       3,
}

var loyaltyTariff = Tariff{
	ID:           LoyaltyTariffID,
	InboundIDs:   DefaultRegistrationInboundIDs,
	TotalGB:      300,
	LimitHWID:    10,
	MonthlyPrice: 200,
}

// Add loyalty user emails here in lowercase.
var loyaltyTariffUsers = map[string]struct{}{
	"nvslov":     {},
	"synviktor2": {},
}

var tariffs = []Tariff{
	{
		ID:           "30gb_1",
		InboundIDs:   DefaultRegistrationInboundIDs,
		TotalGB:      30,
		LimitHWID:    1,
		MonthlyPrice: 50,
	},
	{
		ID:           "50gb_3",
		InboundIDs:   DefaultRegistrationInboundIDs,
		TotalGB:      50,
		LimitHWID:    3,
		MonthlyPrice: 100,
	},
	{
		ID:           "100gb_5",
		InboundIDs:   DefaultRegistrationInboundIDs,
		TotalGB:      100,
		LimitHWID:    5,
		MonthlyPrice: 200,
	},
	{
		ID:           "300gb_10",
		InboundIDs:   DefaultRegistrationInboundIDs,
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
