package synvpn

import (
	"crypto/rand"
	"fmt"
)

const labelAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomLowerAndNum(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("label length must be positive")
	}

	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random label: %w", err)
	}

	for i, b := range randomBytes {
		result[i] = labelAlphabet[int(b)%len(labelAlphabet)]
	}

	return string(result), nil
}
