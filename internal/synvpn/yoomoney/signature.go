package yoomoney

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

func notificationPayload(values url.Values) string {
	data := make([]string, 0, len(values))

	for key, vals := range values {
		if key == "sign" {
			continue
		}

		for _, value := range vals {
			data = append(data, key+"="+rfc3986Encode(value))
		}
	}

	sort.Strings(data)

	return strings.Join(data, "&")
}

func VerifyNotification(values url.Values, secret string) bool {
	sign := values.Get("sign")
	if sign == "" || secret == "" {
		return false
	}

	payload := notificationPayload(values)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))

	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal(
		[]byte(strings.ToLower(sign)),
		[]byte(expected),
	)
}

func rfc3986Encode(value string) string {
	encoded := url.QueryEscape(value)

	// QueryEscape uses '+' for spaces.
	// RFC 3986 requires %20.
	return strings.ReplaceAll(encoded, "+", "%20")
}
