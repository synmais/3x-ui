package synvpn

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"

	"github.com/mhsanaei/3x-ui/v3/internal/synvpn/yoomoney"
)

// BillingService contains payment business logic.
type BillingService struct{}

// CreatePayment creates a pending payment and returns it.
func (s *BillingService) CreatePayment(
	tgID int64,
	clientEmail string,
	comment string,
	tariffID string,
	months int,
	amountKopecks int64,
	provider string,
	expiresAt time.Time,
) (*model.Payment, error) {
	if tgID == 0 {
		return nil, fmt.Errorf("telegram ID is required")
	}
	if clientEmail == "" {
		return nil, fmt.Errorf("client email is required")
	}
	if comment == "" {
		return nil, fmt.Errorf("comment is required")
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
		ID:          uuid.NewString(),
		Label:       label,
		ClientEmail: clientEmail,
		TgID:        tgID,
		Comment:     comment,
		TariffID:    tariffID,
		Months:      months,
		Amount:      amountKopecks,
		Currency:    "RUB",
		Status:      model.PaymentPending,
		Provider:    provider,
		ExpiresAt:   expiresAt,
	}

	if err := database.GetDB().Create(payment).Error; err != nil {
		return nil, err
	}

	return payment, nil
}

// YooMoneyPaymentURL builds a YooMoney payment URL.
func YooMoneyPaymentURL(
	wallet string,
	payment *model.Payment,
	successURL string,
) (string, error) {
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
	values.Set("quickpay-form", "button")
	values.Set("paymentType", "AC")
	values.Set("sum", formatRUB(payment.Amount))
	values.Set("label", payment.Label)

	if successURL != "" {
		values.Set("successURL", successURL)
	}

	return "https://yoomoney.ru/quickpay/confirm?" + values.Encode(), nil
}

func formatRUB(amountKopecks int64) string {
	rubles := amountKopecks / 100
	kopecks := amountKopecks % 100

	return strconv.FormatInt(rubles, 10) + "." + fmt.Sprintf("%02d", kopecks)
}

// ConfirmYooMoneyPayment confirms a pending YooMoney payment.
func (s *BillingService) ConfirmYooMoneyPayment(
	notification *yoomoney.YooMoneyNotification,
) (*model.Payment, error) {
	if notification == nil {
		return nil, fmt.Errorf("YooMoney notification is nil")
	}

	if notification.Label == "" {
		return nil, fmt.Errorf("YooMoney payment label is empty")
	}

	var payment model.Payment
	if err := database.GetDB().
		Where("label = ?", notification.Label).
		First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	if payment.Provider != "yoomoney" {
		return nil, fmt.Errorf("unexpected payment provider: %q", payment.Provider)
	}

	if payment.Currency != yoomoney.CurrencyName(notification.Currency) {
		return nil, fmt.Errorf(
			"payment currency mismatch: expected %s, got %s",
			payment.Currency,
			notification.Currency,
		)
	}

	receivedAmount := notification.Amount

	if notification.NotificationType == "card-incoming" &&
		notification.WithdrawAmount > 0 {
		receivedAmount = notification.WithdrawAmount
	}

	if payment.Amount != receivedAmount {
		return nil, fmt.Errorf(
			"payment amount mismatch: expected %d, got %d",
			payment.Amount,
			receivedAmount,
		)
	}

	// YooMoney may send the same notification more than once.
	if payment.Status == model.PaymentPaid {
		return &payment, nil
	}

	if payment.Status == model.PaymentProcessing {
		return &payment, nil
	}

	if payment.Status != model.PaymentPending {
		return nil, fmt.Errorf(
			"payment has unexpected status: %s",
			payment.Status,
		)
	}

	if !payment.ExpiresAt.IsZero() && time.Now().After(payment.ExpiresAt) {
		payment.Status = model.PaymentExpired

		if err := database.GetDB().Save(&payment).Error; err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("payment has expired")
	}

	operationID := notification.OperationID

	result := database.GetDB().
		Model(&model.Payment{}).
		Where("id = ? AND status = ?", payment.ID, model.PaymentPending).
		Updates(map[string]interface{}{
			"status":                model.PaymentProcessing,
			"provider_operation_id": operationID,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		var current model.Payment
		if err := database.GetDB().
			Where("id = ?", payment.ID).
			First(&current).Error; err != nil {
			return nil, err
		}

		if current.Status == model.PaymentPaid ||
			current.Status == model.PaymentProcessing {
			return &current, nil
		}

		return nil, fmt.Errorf(
			"payment has unexpected status: %s",
			current.Status,
		)
	}

	if err := database.GetDB().
		Where("id = ?", payment.ID).
		First(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

func (s *BillingService) CompletePayment(
	paymentID string,
) (*model.Payment, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	var payment model.Payment
	if err := database.GetDB().
		Where("id = ?", paymentID).
		First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	if payment.Status == model.PaymentPaid {
		return &payment, nil
	}

	if payment.Status != model.PaymentProcessing {
		return nil, fmt.Errorf(
			"payment has unexpected status: %s",
			payment.Status,
		)
	}

	now := time.Now()

	result := database.GetDB().
		Model(&model.Payment{}).
		Where("id = ? AND status = ?", paymentID, model.PaymentProcessing).
		Updates(map[string]interface{}{
			"status":  model.PaymentPaid,
			"paid_at": now,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		var current model.Payment
		if err := database.GetDB().
			Where("id = ?", paymentID).
			First(&current).Error; err != nil {
			return nil, err
		}

		if current.Status == model.PaymentPaid {
			return &current, nil
		}

		return nil, fmt.Errorf(
			"payment has unexpected status: %s",
			current.Status,
		)
	}

	if err := database.GetDB().
		Where("id = ?", paymentID).
		First(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

// CreateYooMoneyPayment creates a pending YooMoney payment and returns its URL.
func (s *BillingService) CreateYooMoneyPayment(
	tgID int64,
	clientEmail string,
	comment string,
	tariffID string,
	months int,
	amountKopecks int64,
	expiresAt time.Time,
	wallet string,
) (string, error) {
	payment, err := s.CreatePayment(
		tgID,
		clientEmail,
		comment,
		tariffID,
		months,
		amountKopecks,
		"yoomoney",
		expiresAt,
	)
	if err != nil {
		return "", err
	}

	paymentURL, err := YooMoneyPaymentURL(wallet, payment, "")
	if err != nil {
		return "", err
	}

	return paymentURL, nil
}
