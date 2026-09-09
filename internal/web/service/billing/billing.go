package billing

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// BillingService contains payment business logic.
type BillingService struct{}

// CreatePayment creates a pending payment and returns it.
func (s *BillingService) CreatePayment(
	tgID int64,
	tariffID string,
	months int,
	amountKopecks int64,
	provider string,
	expiresAt time.Time,
) (*model.Payment, error) {
	if tgID == 0 {
		return nil, fmt.Errorf("telegram ID is required")
	}
	if tariffID == "" {
		return nil, fmt.Errorf("tariff ID is required")
	}
	if months <= 0 {
		return nil, fmt.Errorf("months must be positive")
	}
	if amountKopecks <= 0 {
		return nil, fmt.Errorf("payment amount must be positive")
	}
	if provider == "" {
		return nil, fmt.Errorf("payment provider is required")
	}

	label, err := randomLowerAndNum(16)
	if err != nil {
		return nil, err
	}

	payment := &model.Payment{
		ID:        uuid.NewString(),
		Label:     label,
		TgID:      tgID,
		TariffID:  tariffID,
		Months:    months,
		Amount:    amountKopecks,
		Currency:  "RUB",
		Status:    model.PaymentPending,
		Provider:  provider,
		ExpiresAt: expiresAt,
	}

	if err := database.GetDB().Create(payment).Error; err != nil {
		return nil, err
	}

	return payment, nil
}

// YooMoneyLabel returns the internal label sent to YooMoney.
func YooMoneyLabel(paymentID string) string {
	return "synvpn:" + paymentID
}

// PaymentIDFromYooMoneyLabel extracts the internal payment ID.
func PaymentIDFromYooMoneyLabel(label string) (string, error) {
	const prefix = "synvpn:"

	if !strings.HasPrefix(label, prefix) {
		return "", fmt.Errorf("invalid payment label")
	}

	id := strings.TrimPrefix(label, prefix)
	if id == "" {
		return "", fmt.Errorf("empty payment ID")
	}

	return id, nil
}

// YooMoneyPaymentURL builds a YooMoney payment URL.
func YooMoneyPaymentURL(wallet string, payment *model.Payment, successURL string) (string, error) {
	if wallet == "" {
		return "", fmt.Errorf("YooMoney wallet is not configured")
	}
	if payment == nil {
		return "", fmt.Errorf("payment is nil")
	}
	if payment.Amount <= 0 {
		return "", fmt.Errorf("invalid payment amount")
	}

	values := url.Values{}
	values.Set("receiver", wallet)
	values.Set("quickpay-form", "shop")
	values.Set("targets", "synVPN subscription")
	values.Set("paymentType", "AC")
	values.Set("sum", formatRUB(payment.Amount))
	values.Set("label", payment.Label)

	if successURL != "" {
		values.Set("successURL", successURL)
	}

	return "https://yoomoney.ru/quickpay/confirm.xml?" + values.Encode(), nil
}

func formatRUB(amountKopecks int64) string {
	rubles := amountKopecks / 100
	kopecks := amountKopecks % 100

	return strconv.FormatInt(rubles, 10) + "." + fmt.Sprintf("%02d", kopecks)
}
