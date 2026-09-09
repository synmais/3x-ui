package billing

import (
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
		ID:        "payment-test",
		Label:     "testlabel123456",
		TgID:      123456789,
		Comment:   "test user",
		TariffID:  "50gb",
		Months:    1,
		Amount:    5000,
		Currency:  "RUB",
		Status:    status,
		Provider:  "yoomoney",
		ExpiresAt: time.Now().Add(time.Hour),
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

	if payment.Status != model.PaymentPaid {
		t.Fatalf("payment status = %q, want %q",
			payment.Status, model.PaymentPaid)
	}

	if payment.ProviderOperationID == nil ||
		*payment.ProviderOperationID != "operation-123" {
		t.Fatalf("operation ID = %v, want operation-123",
			payment.ProviderOperationID)
	}

	if payment.PaidAt == nil {
		t.Fatal("PaidAt is nil")
	}
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
