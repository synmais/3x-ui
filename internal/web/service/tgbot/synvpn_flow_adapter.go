package tgbot

import (
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

func (t *Tgbot) synVPNFlow(view ...*clientViewRenderer) *synvpntgbot.Flow {
	sendMessage := t.SendMsgToTgbot
	showRegistrationPrompt := func(chatID int64) { t.showRegistrationPrompt(chatID) }
	if len(view) > 0 && view[0] != nil {
		sendMessage = view[0].send
		showRegistrationPrompt = func(chatID int64) { t.showRegistrationPrompt(chatID, view[0]) }
	}
	flow := &synvpntgbot.Flow{
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
				if err := copySynVPNSupportMessages(fromChatID, adminID, messageIDs); err != nil {
					logger.Warning("Failed to forward support attachment to admin:", err)
				}
			}
		},
		CopySupportMedia:       copySynVPNSupportMessages,
		SendMessageNoPreview:   t.SendMsgToTgbotNoPreview,
		RandomClientEmail:      t.randomLowerAndNum,
		ShowRegistrationPrompt: showRegistrationPrompt,
		SendSubscriptionLinks: func(chatID int64, email string) {
			// Payment confirmation and its subscription link are standalone
			// messages, so do not let later button taps edit an older client view.
			t.resetClientView(chatUser{chatID: chatID, userID: chatID})
			t.sendClientPostPaymentLinks(chatID, email)
		},
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
			if len(view) > 0 && view[0] != nil {
				view[0].schedulePaymentExpiry(paymentID, expiresAt)
			}
		},
	}
	if len(view) > 0 && view[0] != nil {
		flow.SendRegistrationMessage = view[0].sendBanner
	}
	return flow
}
