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
		})
	}
}
