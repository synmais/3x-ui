package billing

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/billing/yoomoney"
)

func setupBillingTestDB(t *testing.T) {
	t.Helper()

	if err := database.InitDB(filepath.Join(t.TempDir(), "x-ui.db")); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := database.CloseDB(); err != nil {
			t.Fatal(err)
		}
	})
}

func createTestPayment(t *testing.T, status model.PaymentStatus) *model.Payment {
	t.Helper()

	payment := &model.Payment{
		ID:          "payment-test",
		Label:       "testlabel123456",
		ClientEmail: "client@example",
		TgID:        123456789,
		Comment:     "test user",
		TariffID:    "50gb",
		Months:      1,
		Amount:      5000,
		Currency:    "RUB",
		Status:      status,
		Provider:    "yoomoney",
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	if err := database.GetDB().Create(payment).Error; err != nil {
		t.Fatal(err)
	}

	return payment
}

func TestConfirmYooMoneyPayment(t *testing.T) {
	setupBillingTestDB(t)

	createTestPayment(t, model.PaymentPending)

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-123",
		Amount:      5000,
		Currency:    "RUB",
		Label:       "testlabel123456",
	}

	payment, err := (&BillingService{}).ConfirmYooMoneyPayment(notification)
	if err != nil {
		t.Fatalf("ConfirmYooMoneyPayment() error = %v", err)
	}

	if payment.Status != model.PaymentProcessing {
		t.Fatalf("payment status = %q, want %q", payment.Status, model.PaymentProcessing)
	}

	if payment.ProviderOperationID == nil ||
		*payment.ProviderOperationID != "operation-123" {
		t.Fatalf("operation ID = %v, want operation-123",
			payment.ProviderOperationID)
	}

	//if payment.PaidAt == nil {
	//	t.Fatal("PaidAt is nil")
	//}
}

func TestConfirmYooMoneyPaymentRejectsAmountMismatch(t *testing.T) {
	setupBillingTestDB(t)

	createTestPayment(t, model.PaymentPending)

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-123",
		Amount:      10000,
		Currency:    "RUB",
		Label:       "testlabel123456",
	}

	if _, err := (&BillingService{}).ConfirmYooMoneyPayment(notification); err == nil {
		t.Fatal("expected amount mismatch error")
	}
}

func TestConfirmYooMoneyPaymentRejectsCurrencyMismatch(t *testing.T) {
	setupBillingTestDB(t)

	createTestPayment(t, model.PaymentPending)

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-123",
		Amount:      5000,
		Currency:    "USD",
		Label:       "testlabel123456",
	}

	if _, err := (&BillingService{}).ConfirmYooMoneyPayment(notification); err == nil {
		t.Fatal("expected currency mismatch error")
	}
}

func TestConfirmYooMoneyPaymentIsIdempotent(t *testing.T) {
	setupBillingTestDB(t)

	payment := createTestPayment(t, model.PaymentPaid)

	operationID := "operation-original"
	payment.ProviderOperationID = &operationID

	if err := database.GetDB().Save(payment).Error; err != nil {
		t.Fatal(err)
	}

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-duplicate",
		Amount:      5000,
		Currency:    "RUB",
		Label:       "testlabel123456",
	}

	confirmed, err := (&BillingService{}).ConfirmYooMoneyPayment(notification)
	if err != nil {
		t.Fatalf("ConfirmYooMoneyPayment() error = %v", err)
	}

	if confirmed.Status != model.PaymentPaid {
		t.Fatalf("payment status = %q, want %q",
			confirmed.Status, model.PaymentPaid)
	}

	if confirmed.ProviderOperationID == nil ||
		*confirmed.ProviderOperationID != "operation-original" {
		t.Fatalf("operation ID changed on duplicate notification: %v",
			confirmed.ProviderOperationID)
	}
}

func TestConfirmYooMoneyPaymentExpiresPayment(t *testing.T) {
	setupBillingTestDB(t)

	payment := createTestPayment(t, model.PaymentPending)
	payment.ExpiresAt = time.Now().Add(-time.Minute)

	if err := database.GetDB().Save(payment).Error; err != nil {
		t.Fatal(err)
	}

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-123",
		Amount:      5000,
		Currency:    "RUB",
		Label:       "testlabel123456",
	}

	if _, err := (&BillingService{}).ConfirmYooMoneyPayment(notification); err == nil {
		t.Fatal("expected expired payment error")
	}

	var stored model.Payment
	if err := database.GetDB().
		Where("label = ?", "testlabel123456").
		First(&stored).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Status != model.PaymentExpired {
		t.Fatalf("payment status = %q, want %q",
			stored.Status, model.PaymentExpired)
	}
}

func TestConfirmYooMoneyPaymentProcessingIsIdempotent(t *testing.T) {
	setupBillingTestDB(t)

	createTestPayment(t, model.PaymentPending)

	notification := &yoomoney.YooMoneyNotification{
		OperationID: "operation-processing",
		Amount:      5000,
		Currency:    "RUB",
		Label:       "testlabel123456",
	}

	service := &BillingService{}

	first, err := service.ConfirmYooMoneyPayment(notification)
	if err != nil {
		t.Fatalf("first ConfirmYooMoneyPayment() error = %v", err)
	}

	if first.Status != model.PaymentProcessing {
		t.Fatalf("first payment status = %q, want %q",
			first.Status, model.PaymentProcessing)
	}

	second, err := service.ConfirmYooMoneyPayment(notification)
	if err != nil {
		t.Fatalf("second ConfirmYooMoneyPayment() error = %v", err)
	}

	if second.Status != model.PaymentProcessing {
		t.Fatalf("second payment status = %q, want %q",
			second.Status, model.PaymentProcessing)
	}

	if second.ProviderOperationID == nil ||
		*second.ProviderOperationID != "operation-processing" {
		t.Fatalf("operation ID = %v, want operation-processing",
			second.ProviderOperationID)
	}
}

func TestCompletePayment(t *testing.T) {
	setupBillingTestDB(t)

	payment := createTestPayment(t, model.PaymentProcessing)

	completed, err := (&BillingService{}).CompletePayment(payment.ID)
	if err != nil {
		t.Fatalf("CompletePayment() error = %v", err)
	}

	if completed.Status != model.PaymentPaid {
		t.Fatalf("payment status = %q, want %q",
			completed.Status, model.PaymentPaid)
	}

	if completed.ClientEmail != "client@example" {
		t.Fatalf("client email = %q, want client@example",
			completed.ClientEmail)
	}

	if completed.PaidAt == nil {
		t.Fatal("PaidAt is nil")
	}
}

func TestCompletePaymentIsIdempotent(t *testing.T) {
	setupBillingTestDB(t)

	payment := createTestPayment(t, model.PaymentPaid)

	clientEmail := "existing-client"
	payment.ClientEmail = clientEmail

	paidAt := time.Now().Add(-time.Minute)
	payment.PaidAt = &paidAt

	if err := database.GetDB().Save(payment).Error; err != nil {
		t.Fatal(err)
	}

	completed, err := (&BillingService{}).CompletePayment(payment.ID)

	if err != nil {
		t.Fatalf("CompletePayment() error = %v", err)
	}

	if completed.Status != model.PaymentPaid {
		t.Fatalf("payment status = %q, want %q",
			completed.Status, model.PaymentPaid)
	}

	if completed.ClientEmail != clientEmail {
		t.Fatalf("client email changed from %q to %q",
			clientEmail, completed.ClientEmail)
	}

	if completed.PaidAt == nil ||
		!completed.PaidAt.Equal(paidAt) {
		t.Fatal("PaidAt changed on repeated completion")
	}
}

func TestCompletePaymentRejectsPendingPayment(t *testing.T) {
	setupBillingTestDB(t)

	payment := createTestPayment(t, model.PaymentPending)

	if _, err := (&BillingService{}).CompletePayment(payment.ID); err == nil {
		t.Fatal("expected error for pending payment")
	}
}

func TestYooMoneyPaymentURL(t *testing.T) {
	payment := &model.Payment{
		Label:  "testlabel123456",
		Amount: 10050,
	}

	const (
		wallet     = "4100111122233344"
		targets    = "Подписка SynVPN"
		successURL = "https://example.com/success"
	)

	paymentURL, err := YooMoneyPaymentURL(
		wallet,
		targets,
		payment,
		successURL,
	)
	if err != nil {
		t.Fatalf("YooMoneyPaymentURL() error = %v", err)
	}

	parsedURL, err := url.Parse(paymentURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Fatalf("URL scheme = %q, want https", parsedURL.Scheme)
	}

	if parsedURL.Host != "yoomoney.ru" {
		t.Fatalf("URL host = %q, want yoomoney.ru", parsedURL.Host)
	}

	if parsedURL.Path != "/quickpay/confirm.xml" {
		t.Fatalf("URL path = %q, want /quickpay/confirm.xml", parsedURL.Path)
	}

	values := parsedURL.Query()

	tests := map[string]string{
		"receiver":      wallet,
		"quickpay-form": "shop",
		"targets":       targets,
		"paymentType":   "AC",
		"sum":           "100.50",
		"label":         payment.Label,
		"successURL":    successURL,
	}

	for key, want := range tests {
		if got := values.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestYooMoneyPaymentURLRejectsInvalidInput(t *testing.T) {
	payment := &model.Payment{
		Label:  "testlabel123456",
		Amount: 10050,
	}

	tests := []struct {
		name    string
		wallet  string
		targets string
		payment *model.Payment
	}{
		{
			name:    "missing wallet",
			wallet:  "",
			targets: "Подписка SynVPN",
			payment: payment,
		},
		{
			name:    "missing targets",
			wallet:  "4100111122233344",
			targets: "",
			payment: payment,
		},
		{
			name:    "nil payment",
			wallet:  "4100111122233344",
			targets: "Подписка SynVPN",
			payment: nil,
		},
		{
			name:    "invalid amount",
			wallet:  "4100111122233344",
			targets: "Подписка SynVPN",
			payment: &model.Payment{
				Label:  "testlabel123456",
				Amount: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := YooMoneyPaymentURL(
				tt.wallet,
				tt.targets,
				tt.payment,
				"",
			); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
