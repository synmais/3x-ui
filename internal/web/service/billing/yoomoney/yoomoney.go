package yoomoney

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// YooMoneyNotification contains the fields relevant to a YooMoney
// p2p-incoming HTTP notification.
type YooMoneyNotification struct {
	NotificationType string
	OperationID      string
	Amount           int64 // kopecks
	Currency         string
	Label            string
	Unaccepted       string
	Sign             string
}

// ParseYooMoneyNotification parses a YooMoney application/x-www-form-urlencoded
// notification and verifies its signature using the HTTP notification secret.
func ParseYooMoneyNotification(
	values url.Values,
	secret string,
) (*YooMoneyNotification, error) {
	if secret == "" {
		return nil, fmt.Errorf("YooMoney notification secret is empty")
	}

	sign := values.Get("sign")
	if sign == "" {
		return nil, fmt.Errorf("missing YooMoney notification signature")
	}

	if !VerifyNotification(values, secret) {
		return nil, fmt.Errorf("invalid YooMoney notification signature")
	}

	if sign == "" {
		return nil, fmt.Errorf("missing YooMoney notification signature")
	}

	notificationType := values.Get("notification_type")
	if notificationType != "p2p-incoming" {
		return nil, fmt.Errorf("unsupported YooMoney notification type: %q", notificationType)
	}

	operationID := values.Get("operation_id")
	if operationID == "" {
		return nil, fmt.Errorf("missing YooMoney operation_id")
	}

	amount, err := parseYooMoneyAmount(values.Get("amount"))
	if err != nil {
		return nil, fmt.Errorf("invalid YooMoney amount: %w", err)
	}

	currency := values.Get("currency")
	if currency == "" {
		return nil, fmt.Errorf("missing YooMoney currency")
	}

	return &YooMoneyNotification{
		NotificationType: notificationType,
		OperationID:      operationID,
		Amount:           amount,
		Currency:         currency,
		Label:            values.Get("label"),
		Unaccepted:       values.Get("unaccepted"),
		Sign:             sign,
	}, nil
}

func parseYooMoneyAmount(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("amount is empty")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal amount")
	}

	rubles, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || rubles < 0 {
		return 0, fmt.Errorf("invalid ruble amount")
	}

	var kopecks int64

	if len(parts) == 2 {
		fraction := parts[1]

		if len(fraction) == 1 {
			fraction += "0"
		}

		if len(fraction) != 2 {
			return 0, fmt.Errorf("invalid kopeck amount")
		}

		kopecks, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid kopeck amount")
		}
	}

	if rubles > (int64(^uint64(0)>>1)-kopecks)/100 {
		return 0, fmt.Errorf("amount overflow")
	}

	return rubles*100 + kopecks, nil
}
