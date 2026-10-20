package tgbot

import (
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	"github.com/mymmrac/telego"
)

func (t *Tgbot) synvpnFlow(window ...*clientWindowRenderer) *synvpntgbot.Flow {
	sendMessage := t.SendMsgToTgbot
	showRegistrationPrompt := func(chatID int64) { t.showRegistrationPrompt(chatID) }
	if len(window) > 0 && window[0] != nil {
		sendMessage = window[0].send
		showRegistrationPrompt = func(chatID int64) { t.showRegistrationPrompt(chatID, window[0]) }
	}
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
		SendMessage:  sendMessage,
		SendToAdmins: t.SendMsgToTgbotAdmins,
		SendSupportMediaToAdmins: func(fromChatID int64, messageIDs []int) {
			for _, adminID := range adminSnapshot() {
				if err := copySupportMessages(fromChatID, adminID, messageIDs); err != nil {
					logger.Warning("Failed to forward support attachment to admin:", err)
				}
			}
		},
		CopySupportMedia:       copySupportMessages,
		SendMessageNoPreview:   t.SendMsgToTgbotNoPreview,
		RandomClientEmail:      t.randomLowerAndNum,
		ShowRegistrationPrompt: showRegistrationPrompt,
		SendSubscriptionLinks:  func(chatID int64, email string) { t.sendClientSubLinks(chatID, email) },
		ListDevices: func(email string) ([]synvpntgbot.Device, int, error) {
			record, err := t.clientService.GetRecordByEmail(nil, email)
			if err != nil {
				return nil, 0, err
			}
			hwids, err := t.clientService.ListClientHwids(email)
			if err != nil {
				return nil, 0, err
			}
			devices := make([]synvpntgbot.Device, 0, len(hwids))
			for _, hwid := range hwids {
				devices = append(devices, synvpntgbot.Device{
					ID:          hwid.Id,
					Fingerprint: hwid.Fingerprint,
					FirstSeen:   hwid.FirstSeen,
					LastSeen:    hwid.LastSeen,
					UserAgent:   hwid.UserAgent,
					DeviceOS:    hwid.DeviceOS,
					OsVersion:   hwid.OsVersion,
					DeviceModel: hwid.DeviceModel,
				})
			}
			limit := record.LimitHwid
			if status, found, statusErr := t.clientService.HwidSlotStatusForSubID(record.SubID); statusErr != nil {
				return nil, 0, statusErr
			} else if found {
				limit = status.Limit
			}
			return devices, limit, nil
		},
		DeleteDeviceByEmail: t.clientService.DeleteClientHwid,
		ListSubscriptions: func() ([]synvpntgbot.SubscriberSubscription, error) {
			var records []model.ClientRecord
			if err := database.GetDB().Where("tg_id <> 0").Find(&records).Error; err != nil {
				return nil, err
			}
			trafficByUser := make(map[int64][]*xray.ClientTraffic)
			out := make([]synvpntgbot.SubscriberSubscription, 0, len(records))
			for _, record := range records {
				if _, ok := trafficByUser[record.TgID]; !ok {
					traffic, err := t.inboundService.GetClientTrafficTgBot(record.TgID)
					if err == nil {
						trafficByUser[record.TgID] = traffic
					}
				}
				item := synvpntgbot.SubscriberSubscription{
					TelegramID: record.TgID,
					Email:      record.Email,
					ExpiryTime: record.ExpiryTime,
				}
				for _, traffic := range trafficByUser[record.TgID] {
					if traffic.Email == record.Email {
						item.TrafficUsed = traffic.Up + traffic.Down
						item.TrafficTotal = traffic.Total
						item.TrafficActive = traffic.Enable
						break
					}
				}
				out = append(out, item)
			}
			return out, nil
		},
		EncodeCallback: t.encodeQuery,
		SchedulePaymentExpiry: func(paymentID string, expiresAt time.Time) {
			if len(window) > 0 && window[0] != nil {
				window[0].schedulePaymentExpiry(paymentID, expiresAt)
			}
		},
	}
}

func (t *Tgbot) askClientForSupport(actor chatUser, window ...*clientWindowRenderer) {
	userStateMgr.set(actor, "awaiting_support_message")
	t.synvpnFlow(window...).AskSupportMessage(actor.chatID)
}

func (t *Tgbot) startSupportReply(actor chatUser, userID int64) {
	userStateMgr.set(actor, "awaiting_support_reply "+strconv.FormatInt(userID, 10))
	t.synvpnFlow().AskSupportReply(actor.chatID, userID)
}

func (t *Tgbot) startRegistration(chatID int64, user telego.User, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).StartRegistration(chatID, user)
}

func (t *Tgbot) startPurchase(chatID int64, user telego.User, kind synvpn.PurchaseKind, email string, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).StartPurchase(chatID, user, kind, email)
}

func (t *Tgbot) startNewPurchase(chatID int64, user telego.User, window ...*clientWindowRenderer) {
	t.startPurchase(chatID, user, synvpn.PurchaseCreate, "", window...)
}

func (t *Tgbot) clearPurchase(chatID int64) {
	synvpntgbot.ClearPurchase(chatID)
}

func (t *Tgbot) purchaseTariff(chatID, tgUserID int64, tariffID string, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).PurchaseTariff(chatID, tgUserID, tariffID)
}

func (t *Tgbot) purchasePeriod(chatID, tgUserID int64, months int, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).PurchasePeriod(chatID, tgUserID, months)
}

func (t *Tgbot) handleTariffCallback(chatID, tgUserID int64, data string, window ...*clientWindowRenderer) bool {
	if !strings.HasPrefix(data, "register_tariff_") && !strings.HasPrefix(data, "subscription_tariff_") {
		return false
	}

	tariffID := strings.TrimPrefix(strings.TrimPrefix(data, "register_tariff_"), "subscription_tariff_")
	t.purchaseTariff(chatID, tgUserID, tariffID, window...)
	return true
}

func (t *Tgbot) handlePeriodCallback(chatID, tgUserID int64, data string, window ...*clientWindowRenderer) bool {
	if !strings.HasPrefix(data, "register_period_") && !strings.HasPrefix(data, "subscription_period_") {
		return false
	}

	monthsStr := strings.TrimPrefix(strings.TrimPrefix(data, "register_period_"), "subscription_period_")
	months, err := strconv.Atoi(monthsStr)
	if err != nil {
		if len(window) > 0 {
			window[0].send(chatID, "Некорректный срок подписки.")
		} else {
			t.SendMsgToTgbot(chatID, "Некорректный срок регистрации.")
		}
		return true
	}

	t.purchasePeriod(chatID, tgUserID, months, window...)
	return true
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).ConfirmPurchase(chatID, tgUserID)
}

func (t *Tgbot) startOwnRenewal(chatID int64, user telego.User, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).StartOwnRenewal(chatID, user)
}

func (t *Tgbot) showOwnDevices(chatID, tgUserID int64, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).ShowOwnDevices(chatID, tgUserID)
}

func (t *Tgbot) handleDeviceCallback(chatID, tgUserID int64, data string, window ...*clientWindowRenderer) bool {
	parts := strings.Fields(data)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "client_device") {
		return false
	}
	if parts[0] == "client_devices" && len(parts) == 2 {
		t.synvpnFlow(window...).ShowDevices(chatID, tgUserID, parts[1])
		return true
	}
	if (parts[0] == "client_device" || parts[0] == "client_device_remove" || parts[0] == "client_device_delete") && len(parts) == 3 {
		id, err := strconv.Atoi(parts[2])
		if err != nil {
			t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.answers.errorOperation"))
			return true
		}
		flow := t.synvpnFlow(window...)
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

func (t *Tgbot) startRenewal(chatID int64, user telego.User, email string, window ...*clientWindowRenderer) {
	t.synvpnFlow(window...).StartRenewal(chatID, user, email)
}

func (t *Tgbot) NotifySynVPNUsers() { t.synvpnFlow().NotifySubscribers() }

func (t *Tgbot) ProcessYooMoneyPayment(payment *model.Payment) error {
	err := t.synvpnFlow().ProcessYooMoneyPayment(payment)
	if err == nil && payment != nil {
		cancelClientPaymentExpiry(payment.ID)
	}
	return err
}
