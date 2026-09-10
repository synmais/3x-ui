package model

import "time"

type PaymentStatus string

const (
	PaymentPending    PaymentStatus = "pending"
	PaymentProcessing PaymentStatus = "processing"
	PaymentPaid       PaymentStatus = "paid"
	PaymentExpired    PaymentStatus = "expired"
	PaymentFailed     PaymentStatus = "failed"
)

type Payment struct {
	ID string `gorm:"primaryKey"`

	Label string `gorm:"uniqueIndex;not null"`

	ClientEmail string `gorm:"index"`
	TgID        int64  `gorm:"index;not null"`
	Comment     string `gorm:"not null;default:''"`

	TariffID string `gorm:"index;not null"`
	Months   int    `gorm:"not null"`

	// Amount is stored in kopecks.
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"not null;default:RUB"`

	Status PaymentStatus `gorm:"index;not null"`

	Provider string `gorm:"index;not null"`

	// YooMoney operation_id. NULL until the payment is confirmed.
	ProviderOperationID *string `gorm:"uniqueIndex"`

	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
	PaidAt    *time.Time
}
