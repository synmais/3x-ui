package tgbot

import (
	"strconv"
	"strings"

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
		EncodeCallback:      t.encodeQuery,
	}
}

func (t *Tgbot) startRegistration(chatID int64, user telego.User) {
	t.synvpnFlow().StartRegistration(chatID, user)
}

func (t *Tgbot) startPurchase(chatID int64, user telego.User, kind synvpn.PurchaseKind, email string) {
	t.synvpnFlow().StartPurchase(chatID, user, kind, email)
}

func (t *Tgbot) startNewPurchase(chatID int64, user telego.User) {
	t.startPurchase(chatID, user, synvpn.PurchaseCreate, "")
}

func (t *Tgbot) clearPurchase(chatID int64) {
	synvpntgbot.ClearPurchase(chatID)
}

func (t *Tgbot) purchaseTariff(chatID, tgUserID int64, tariffID string) {
	t.synvpnFlow().PurchaseTariff(chatID, tgUserID, tariffID)
}

func (t *Tgbot) purchasePeriod(chatID, tgUserID int64, months int) {
	t.synvpnFlow().PurchasePeriod(chatID, tgUserID, months)
}

func (t *Tgbot) handleTariffCallback(chatID, tgUserID int64, data string) bool {
	if !strings.HasPrefix(data, "register_tariff_") && !strings.HasPrefix(data, "subscription_tariff_") {
		return false
	}

	tariffID := strings.TrimPrefix(strings.TrimPrefix(data, "register_tariff_"), "subscription_tariff_")
	t.purchaseTariff(chatID, tgUserID, tariffID)
	return true
}

func (t *Tgbot) handlePeriodCallback(chatID, tgUserID int64, data string) bool {
	if !strings.HasPrefix(data, "register_period_") && !strings.HasPrefix(data, "subscription_period_") {
		return false
	}

	monthsStr := strings.TrimPrefix(strings.TrimPrefix(data, "register_period_"), "subscription_period_")
	months, err := strconv.Atoi(monthsStr)
	if err != nil {
		t.SendMsgToTgbot(chatID, "Некорректный срок регистрации.")
		return true
	}

	t.purchasePeriod(chatID, tgUserID, months)
	return true
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64) {
	t.synvpnFlow().ConfirmPurchase(chatID, tgUserID)
}

func (t *Tgbot) startOwnRenewal(chatID int64, user telego.User) {
	t.synvpnFlow().StartOwnRenewal(chatID, user)
}

func (t *Tgbot) showOwnDevices(chatID, tgUserID int64) {
	t.synvpnFlow().ShowOwnDevices(chatID, tgUserID)
}

func (t *Tgbot) handleDeviceCallback(chatID, tgUserID int64, data string) bool {
	parts := strings.Fields(data)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "client_device") {
		return false
	}
	if parts[0] == "client_devices" && len(parts) == 2 {
		t.synvpnFlow().ShowDevices(chatID, tgUserID, parts[1])
		return true
	}
	if (parts[0] == "client_device" || parts[0] == "client_device_remove" || parts[0] == "client_device_delete") && len(parts) == 3 {
		id, err := strconv.Atoi(parts[2])
		if err != nil {
			t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.answers.errorOperation"))
			return true
		}
		flow := t.synvpnFlow()
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

func (t *Tgbot) startRenewal(chatID int64, user telego.User, email string) {
	t.synvpnFlow().StartRenewal(chatID, user, email)
}

func (t *Tgbot) ProcessYooMoneyPayment(payment *model.Payment) error {
	return t.synvpnFlow().ProcessYooMoneyPayment(payment)
}
