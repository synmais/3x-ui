package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	"github.com/mymmrac/telego"
	"gorm.io/gorm"
	"time"
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

type SubscriberSubscription struct {
	TelegramID    int64
	Email         string
	ExpiryTime    int64
	TrafficUsed   int64
	TrafficTotal  int64
	TrafficActive bool
}

type Flow struct {
	ClientService  ClientService
	InboundService InboundService
	SettingService SettingService
	FulfillPayment func(*model.Payment) error

	SendMessage              func(int64, string, ...telego.ReplyMarkup)
	SendRegistrationMessage  func(int64, string, ...telego.ReplyMarkup)
	SendToAdmins             func(string, ...telego.ReplyMarkup)
	SendSupportMediaToAdmins func(int64, []int)
	CopySupportMedia         func(int64, int64, []int) error
	SendMessageNoPreview     func(int64, string, ...telego.ReplyMarkup)
	RandomClientEmail        func(int) string
	ShowRegistrationPrompt   func(int64)
	SendSubscriptionLinks    func(int64, string)
	ListDevices              func(string) ([]Device, int, error)
	DeleteDeviceByEmail      func(string, int) error
	ListSubscriptions        func() ([]SubscriberSubscription, error)
	EncodeCallback           func(string) string
	SchedulePaymentExpiry    func(string, time.Time)
}
