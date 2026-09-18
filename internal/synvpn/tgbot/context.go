package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

type ClientService interface {
	GetRecordByEmail(*gorm.DB, string) (*model.ClientRecord, error)
}

type InboundService interface {
	GetClientTrafficTgBot(int64) ([]*xray.ClientTraffic, error)
}

type SettingService interface {
	GetYooMoneyWallet() (string, error)
}

type Flow struct {
	ClientService  ClientService
	InboundService InboundService
	SettingService SettingService
	FulfillPayment func(*model.Payment) error

	SendMessage            func(int64, string, ...telego.ReplyMarkup)
	Translate              func(string, ...string) string
	RandomClientEmail      func(int) string
	ShowRegistrationPrompt func(int64)
	SendSubscriptionLinks  func(int64, string)
}
