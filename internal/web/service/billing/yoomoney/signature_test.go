package yoomoney

import (
	"net/url"
	"testing"
)

func TestVerifyNotificationOfficialExample(t *testing.T) {
	values := url.Values{
		"notification_type": {"p2p-incoming"},
		"operation_id":      {"441361714955017004"},
		"amount":            {"98.00"},
		"withdraw_amount":   {"100.00"},
		"currency":          {"643"},
		"datetime":          {"2013-12-26T08:28:34Z"},
		"sender":            {"41000000000"},
		"codepro":           {"false"},
		"label":             {"ML23045"},
		"unaccepted":        {"false"},
		"sha1_hash":         {"ac13833bd6ba9eff1fa9e4bed76f3d6ebb57f6c0"},
		"sign":              {"a452af731650e2c5b39abcdc7c28dd27db7b3b654c2230ad2c386e64afb98605"},
	}

	if !VerifyNotification(values, "secret123") {
		t.Fatal("official YooMoney notification signature must be valid")
	}
}

func TestNotificationPayloadRFC3986(t *testing.T) {
	values := url.Values{
		"label":             {"hello world"},
		"notification_type": {"p2p-incoming"},
		"sign":              {"ignored"},
	}

	got := notificationPayload(values)

	want := "label=hello%20world&notification_type=p2p-incoming"

	if got != want {
		t.Fatalf("payload mismatch:\ngot:  %s\nwant: %s", got, want)
	}
}
