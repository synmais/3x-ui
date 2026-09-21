package tgbot

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	"github.com/mymmrac/telego"
)

type deviceTestInbound struct {
	traffics []*xray.ClientTraffic
}

func (s deviceTestInbound) GetClientTrafficTgBot(int64) ([]*xray.ClientTraffic, error) {
	return s.traffics, nil
}

func TestShowDevicesRefusesForeignClient(t *testing.T) {
	listed := false
	sent := false
	flow := &Flow{
		InboundService: deviceTestInbound{
			traffics: []*xray.ClientTraffic{{Email: "owner@example.com"}},
		},
		ListDevices: func(string) ([]Device, int, error) {
			listed = true
			return nil, 1, nil
		},
		SendMessage: func(int64, string, ...telego.ReplyMarkup) {
			sent = true
		},
		Translate: func(string, ...string) string { return "error" },
	}

	flow.ShowDevices(1, 42, "foreign@example.com")

	if listed {
		t.Fatal("foreign client reached device listing")
	}
	if !sent {
		t.Fatal("foreign client did not receive an error")
	}
}
