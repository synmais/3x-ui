package tgbot

import "strings"

// IsClientSelfCallback reports whether a callback belongs to a client flow.
// The Telegram adapter must still verify that any referenced client is owned
// by the callback sender.
func IsClientSelfCallback(data string) bool {
	switch data {
	case "register_user", "register_cancel", "subscription_cancel", "subscription_confirm", "client_renew", "client_add_subscription", "client_support":
		return true
	case "client_traffic", "client_commands", "client_sub_links",
		"client_individual_links", "client_qr_links", "client_devices",
		"register_confirm", "client_view_back", "client_view_cancel",
		"client_view_home", "client_payment_fee_info":
		return true
	}
	if strings.HasPrefix(data, "register_tariff_") || strings.HasPrefix(data, "subscription_tariff_") {
		return true
	}
	if strings.HasPrefix(data, "register_period_") || strings.HasPrefix(data, "subscription_period_") {
		return true
	}
	if strings.HasPrefix(data, "client_instruction") {
		return true
	}
	if strings.HasPrefix(data, "client_traffic ") ||
		strings.HasPrefix(data, "client_devices ") ||
		strings.HasPrefix(data, "client_device ") ||
		strings.HasPrefix(data, "client_device_remove ") ||
		strings.HasPrefix(data, "client_device_delete ") ||
		strings.HasPrefix(data, "client_renew ") {
		return true
	}
	_, _, ok := SplitClientLinkCallback(data)
	return ok
}

// SplitClientLinkCallback parses a callback that identifies a client's own
// subscription link or renewal action.
func SplitClientLinkCallback(data string) (action, email string, ok bool) {
	for _, candidate := range []string{"client_sub_links", "client_individual_links", "client_qr_links", "client_renew"} {
		if rest, found := strings.CutPrefix(data, candidate+" "); found && rest != "" {
			return candidate, rest, true
		}
	}
	return "", "", false
}

// IsStrangerRegistrationCallback keeps only the callbacks needed to complete
// or cancel registration and payment available before a client is registered.
func IsStrangerRegistrationCallback(data string) bool {
	switch data {
	case "register_user", "register_cancel", "subscription_confirm", "subscription_cancel", "client_view_back", "client_view_cancel", "client_payment_fee_info":
		return true
	}
	return strings.HasPrefix(data, "subscription_tariff_") || strings.HasPrefix(data, "subscription_period_")
}
