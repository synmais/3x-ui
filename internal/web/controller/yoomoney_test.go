package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"path/filepath"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func signYooMoneyNotification(values url.Values, secret string) string {
	var parts []string

	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "sign" || key == "sha256_hash" {
			continue
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range values[key] {
			encoded := strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
			parts = append(parts, key+"="+encoded)
		}
	}

	payload := strings.Join(parts, "&")

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}

func TestYooMoneyNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-secret"

	values := url.Values{
		"notification_type": {"p2p-incoming"},
		"operation_id":      {"test-operation-123"},
		"amount":            {"100.00"},
		"currency":          {"643"},
		"label":             {"test-label"},
		"unaccepted":        {"false"},
	}

	values.Set("sign", signYooMoneyNotification(values, secret))

	body := values.Encode()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid notification",
			body:       body,
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid signature",
			body: func() string {
				invalid := values.Clone()
				invalid.Set("sign", "invalid-signature")
				return invalid.Encode()
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid notification type",
			body: func() string {
				invalid := values.Clone()
				invalid.Set("notification_type", "unknown")
				invalid.Set("sign", signYooMoneyNotification(invalid, secret))
				return invalid.Encode()
			}(),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid notification" {
				dbPath := filepath.Join(t.TempDir(), "test.db")

				if err := database.InitDB(dbPath); err != nil {
					t.Fatalf("failed to init test database: %v", err)
				}

				payment := &model.Payment{
					ID:       "test-payment",
					Label:    "test-label",
					TgID:     123456789,
					TariffID: "30gb",
					Months:   1,
					Amount:   10000,
					Currency: "RUB",
					Status:   model.PaymentPending,
					Provider: "yoomoney",
				}

				if err := database.GetDB().Create(payment).Error; err != nil {
					t.Fatalf("failed to create test payment: %v", err)
				}
			}

			router := gin.New()

			controller := &YooMoneyController{
				getSecret: func() (string, error) {
					return secret, nil
				},
			}

			controller.initRouter(router.Group(""))

			req := httptest.NewRequest(
				http.MethodPost,
				"/yoomoney/notification",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}

			if tt.name == "valid notification" {
				var payment model.Payment

				if err := database.GetDB().
					Where("label = ?", "test-label").
					First(&payment).Error; err != nil {
					t.Fatalf("failed to load payment: %v", err)
				}

				if payment.Status != model.PaymentPaid {
					t.Fatalf("expected payment status %q, got %q",
						model.PaymentPaid, payment.Status)
				}

				if payment.ProviderOperationID == nil {
					t.Fatal("expected provider operation ID to be set")
				}

				if *payment.ProviderOperationID != "test-operation-123" {
					t.Fatalf("expected provider operation ID %q, got %q",
						"test-operation-123", *payment.ProviderOperationID)
				}

				if payment.PaidAt == nil {
					t.Fatal("expected paid_at to be set")
				}
			}
		})
	}
}

func TestYooMoneyNotificationDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "test-secret"

	dbPath := filepath.Join(t.TempDir(), "test.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("failed to init test database: %v", err)
	}

	payment := &model.Payment{
		ID:       "duplicate-test-payment",
		Label:    "duplicate-test-label",
		TgID:     123456789,
		TariffID: "30gb",
		Months:   1,
		Amount:   10000,
		Currency: "RUB",
		Status:   model.PaymentPending,
		Provider: "yoomoney",
	}

	if err := database.GetDB().Create(payment).Error; err != nil {
		t.Fatalf("failed to create test payment: %v", err)
	}

	values := url.Values{
		"notification_type": {"p2p-incoming"},
		"operation_id":      {"duplicate-operation-123"},
		"amount":            {"100.00"},
		"currency":          {"643"},
		"label":             {"duplicate-test-label"},
		"unaccepted":        {"false"},
	}

	values.Set("sign", signYooMoneyNotification(values, secret))
	body := values.Encode()

	router := gin.New()

	controller := &YooMoneyController{
		getSecret: func() (string, error) {
			return secret, nil
		},
	}

	controller.initRouter(router.Group(""))

	sendNotification := func() int {
		req := httptest.NewRequest(
			http.MethodPost,
			"/yoomoney/notification",
			strings.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		return recorder.Code
	}

	if status := sendNotification(); status != http.StatusOK {
		t.Fatalf("first notification: expected status %d, got %d",
			http.StatusOK, status)
	}

	if status := sendNotification(); status != http.StatusOK {
		t.Fatalf("duplicate notification: expected status %d, got %d",
			http.StatusOK, status)
	}

	var savedPayment model.Payment
	if err := database.GetDB().
		Where("label = ?", "duplicate-test-label").
		First(&savedPayment).Error; err != nil {
		t.Fatalf("failed to load payment: %v", err)
	}

	if savedPayment.Status != model.PaymentPaid {
		t.Fatalf("expected payment status %q, got %q",
			model.PaymentPaid, savedPayment.Status)
	}

	if savedPayment.ProviderOperationID == nil {
		t.Fatal("expected provider operation ID to be set")
	}

	if *savedPayment.ProviderOperationID != "duplicate-operation-123" {
		t.Fatalf("expected provider operation ID %q, got %q",
			"duplicate-operation-123", *savedPayment.ProviderOperationID)
	}

	if savedPayment.PaidAt == nil {
		t.Fatal("expected paid_at to be set")
	}
}
