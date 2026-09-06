package yoomoney

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

func VerifyNotification(values url.Values, secret string) bool {
	sign := values.Get("sign")
	if sign == "" || secret == "" {
		return false
	}

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

	payload := strings.Join(data, "&")

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))

	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal(
		[]byte(strings.ToLower(sign)),
		[]byte(strings.ToLower(expected)),
	)
}

func rfc3986Encode(value string) string {
	encoded := url.QueryEscape(value)

	// QueryEscape uses '+' for spaces.
	// RFC 3986 requires %20.
	encoded = strings.ReplaceAll(encoded, "+", "%20")

	return encoded
}
