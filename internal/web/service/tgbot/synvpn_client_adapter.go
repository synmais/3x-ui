package tgbot

import (
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mymmrac/telego"
)

func (t *Tgbot) prepareSynVPNClientStart(message *telego.Message, chatID int64) {
	actor := messageActor(*message)
	t.resetClientView(actor)
	t.clearPurchase(chatID)
	userStateMgr.clear(actor)
}

func (t *Tgbot) startRegistrationForStranger(chatID int64, user telego.User) bool {
	if t.levelOf(user.ID) != levelStranger {
		return false
	}
	actor := chatUser{userID: user.ID, chatID: chatID}
	t.startRegistration(chatID, user, t.clientViewForActor(actor))
	return true
}

func (t *Tgbot) askClientForSupport(actor chatUser, view ...*clientViewRenderer) {
	userStateMgr.set(actor, "awaiting_support_message")
	t.synVPNFlow(view...).AskSupportMessage(actor.chatID)
}

func (t *Tgbot) startSupportReply(actor chatUser, userID int64) {
	userStateMgr.set(actor, "awaiting_support_reply "+strconv.FormatInt(userID, 10))
	t.synVPNFlow().AskSupportReply(actor.chatID, userID)
}

func (t *Tgbot) startRegistration(chatID int64, user telego.User, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).StartRegistration(chatID, user)
}

func (t *Tgbot) startPurchase(chatID int64, user telego.User, kind synvpn.PurchaseKind, email string, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).StartPurchase(chatID, user, kind, email)
}

func (t *Tgbot) startNewPurchase(chatID int64, user telego.User, view ...*clientViewRenderer) {
	t.startPurchase(chatID, user, synvpn.PurchaseCreate, "", view...)
}

func (t *Tgbot) clearPurchase(chatID int64) {
	synvpntgbot.ClearPurchase(chatID)
}

func (t *Tgbot) purchaseTariff(chatID, tgUserID int64, tariffID string, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).PurchaseTariff(chatID, tgUserID, tariffID)
}

func (t *Tgbot) purchasePeriod(chatID, tgUserID int64, months int, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).PurchasePeriod(chatID, tgUserID, months)
}

func (t *Tgbot) handleTariffCallback(chatID, tgUserID int64, data string, view ...*clientViewRenderer) bool {
	if !strings.HasPrefix(data, "register_tariff_") && !strings.HasPrefix(data, "subscription_tariff_") {
		return false
	}

	tariffID := strings.TrimPrefix(strings.TrimPrefix(data, "register_tariff_"), "subscription_tariff_")
	t.purchaseTariff(chatID, tgUserID, tariffID, view...)
	return true
}

func (t *Tgbot) handlePeriodCallback(chatID, tgUserID int64, data string, view ...*clientViewRenderer) bool {
	if !strings.HasPrefix(data, "register_period_") && !strings.HasPrefix(data, "subscription_period_") {
		return false
	}

	monthsStr := strings.TrimPrefix(strings.TrimPrefix(data, "register_period_"), "subscription_period_")
	months, err := strconv.Atoi(monthsStr)
	if err != nil {
		if len(view) > 0 {
			view[0].send(chatID, "Некорректный срок подписки.")
		} else {
			t.SendMsgToTgbot(chatID, "Некорректный срок регистрации.")
		}
		return true
	}

	t.purchasePeriod(chatID, tgUserID, months, view...)
	return true
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).ConfirmPurchase(chatID, tgUserID)
}

func (t *Tgbot) startOwnRenewal(chatID int64, user telego.User, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).StartOwnRenewal(chatID, user)
}

func (t *Tgbot) showOwnDevices(chatID, tgUserID int64, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).ShowOwnDevices(chatID, tgUserID)
}

func (t *Tgbot) handleDeviceCallback(chatID, tgUserID int64, data string, view ...*clientViewRenderer) bool {
	parts := strings.Fields(data)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "client_device") {
		return false
	}
	if parts[0] == "client_devices" && len(parts) == 2 {
		t.synVPNFlow(view...).ShowDevices(chatID, tgUserID, parts[1])
		return true
	}
	if (parts[0] == "client_device" || parts[0] == "client_device_remove" || parts[0] == "client_device_delete") && len(parts) == 3 {
		id, err := strconv.Atoi(parts[2])
		if err != nil {
			t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.answers.errorOperation"))
			return true
		}
		flow := t.synVPNFlow(view...)
		switch parts[0] {
		case "client_device":
			flow.ShowDevice(chatID, tgUserID, parts[1], id)
		case "client_device_remove":
			flow.ConfirmDeviceDelete(chatID, tgUserID, parts[1], id)
		case "client_device_delete":
			flow.DeleteDevice(chatID, tgUserID, parts[1], id)
		}
		return true
	}
	t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.answers.errorOperation"))
	return true
}

func (t *Tgbot) startRenewal(chatID int64, user telego.User, email string, view ...*clientViewRenderer) {
	t.synVPNFlow(view...).StartRenewal(chatID, user, email)
}

func (t *Tgbot) NotifySynVPNUsers() { t.synVPNFlow().NotifySubscribers() }

func (t *Tgbot) ProcessYooMoneyPayment(payment *model.Payment) error {
	err := t.synVPNFlow().ProcessYooMoneyPayment(payment)
	if err == nil && payment != nil {
		cancelClientPaymentExpiry(payment.ID)
	}
	return err
}
