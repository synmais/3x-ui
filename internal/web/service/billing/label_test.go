package billing

import (
	"regexp"
	"testing"
)

func TestRandomLowerAndNum(t *testing.T) {
	const length = 16

	label, err := randomLowerAndNum(length)
	if err != nil {
		t.Fatalf("randomLowerAndNum() error = %v", err)
	}

	if len(label) != length {
		t.Fatalf("randomLowerAndNum() length = %d, want %d", len(label), length)
	}

	if !regexp.MustCompile(`^[a-z0-9]+$`).MatchString(label) {
		t.Fatalf("randomLowerAndNum() = %q, contains invalid characters", label)
	}
}

func TestRandomLowerAndNumProducesDifferentValues(t *testing.T) {
	first, err := randomLowerAndNum(16)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := randomLowerAndNum(16)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first == second {
		t.Fatalf("two generated labels are identical: %q", first)
	}
}
