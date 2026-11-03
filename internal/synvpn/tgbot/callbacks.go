package tgbot

import "strings"

// IsStrangerRegistrationCallback keeps only the callbacks needed to complete
// or cancel registration and payment available before a client is registered.
func IsStrangerRegistrationCallback(data string) bool {
	switch data {
	case "register_user", "register_cancel", "subscription_confirm", "subscription_cancel", "client_view_back", "client_view_cancel", "client_payment_fee_info":
		return true
	}
	return strings.HasPrefix(data, "subscription_tariff_") || strings.HasPrefix(data, "subscription_period_")
}
