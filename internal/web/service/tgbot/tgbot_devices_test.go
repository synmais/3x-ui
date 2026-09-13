package tgbot

import (
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func TestDeviceNamePrefersKnownDeviceMetadata(t *testing.T) {
	device := service.ClientHwidInfo{DeviceModel: "Pixel 9", DeviceOS: "Android", OsVersion: "16", UserAgent: "client/1"}
	if got := deviceName(device, 1); got != "Pixel 9" {
		t.Fatalf("deviceName() = %q, want model", got)
	}
	device.DeviceModel = ""
	if got := deviceName(device, 1); got != "Android 16" {
		t.Fatalf("deviceName() = %q, want OS and version", got)
	}
}

func TestDeviceNameFallsBackAndTruncates(t *testing.T) {
	if got := deviceName(service.ClientHwidInfo{}, 3); got != "Device 3" {
		t.Fatalf("deviceName() = %q", got)
	}
	long := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if got := deviceName(service.ClientHwidInfo{UserAgent: long}, 1); len([]rune(got)) != 48 {
		t.Fatalf("device label length = %d, want 48", len([]rune(got)))
	}
}

func TestFormatDeviceTime(t *testing.T) {
	if got := formatDeviceTime(0); got != "—" {
		t.Fatalf("formatDeviceTime(0) = %q", got)
	}
	stamp := time.Date(2026, time.September, 13, 12, 34, 0, 0, time.Local).UnixMilli()
	if got := formatDeviceTime(stamp); got != "13.09.2026 12:34" {
		t.Fatalf("formatDeviceTime() = %q", got)
	}
}
