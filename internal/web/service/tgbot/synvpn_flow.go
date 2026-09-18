package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mymmrac/telego"
)

func (t *Tgbot) synvpnFlow() *synvpntgbot.Flow {
	return &synvpntgbot.Flow{
		ClientService:  &t.clientService,
		InboundService: &t.inboundService,
		SettingService: &t.settingService,
		FulfillPayment: func(payment *model.Payment) error {
			fulfillment := synvpn.PaymentFulfillmentService{
				ClientService:  t.clientService,
				InboundService: t.inboundService,
				XrayService:    t.xrayService,
			}
			return fulfillment.CreateClientFromPayment(payment)
		},
		SendMessage:            t.SendMsgToTgbot,
		Translate:              t.I18nBot,
		RandomClientEmail:      t.randomLowerAndNum,
		ShowRegistrationPrompt: t.showRegistrationPrompt,
		SendSubscriptionLinks:  t.sendClientSubLinks,
	}
}

func (t *Tgbot) startRegistration(chatID int64, user telego.User) {
	t.synvpnFlow().StartRegistration(chatID, user)
}

func (t *Tgbot) startPurchase(chatID int64, user telego.User, kind synvpn.PurchaseKind, email string) {
	t.synvpnFlow().StartPurchase(chatID, user, kind, email)
}

func (t *Tgbot) purchaseTariff(chatID, tgUserID int64, tariffID string) {
	t.synvpnFlow().PurchaseTariff(chatID, tgUserID, tariffID)
}

func (t *Tgbot) purchasePeriod(chatID, tgUserID int64, months int) {
	t.synvpnFlow().PurchasePeriod(chatID, tgUserID, months)
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64) {
	t.synvpnFlow().ConfirmPurchase(chatID, tgUserID)
}

func (t *Tgbot) startOwnRenewal(chatID int64, user telego.User) {
	t.synvpnFlow().StartOwnRenewal(chatID, user)
}

func (t *Tgbot) startRenewal(chatID int64, user telego.User, email string) {
	t.synvpnFlow().StartRenewal(chatID, user, email)
}

func (t *Tgbot) ProcessYooMoneyPayment(payment *model.Payment) error {
	return t.synvpnFlow().ProcessYooMoneyPayment(payment)
}
