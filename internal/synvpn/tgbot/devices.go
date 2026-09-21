package tgbot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type Device struct {
	ID          int
	Fingerprint string
	FirstSeen   int64
	LastSeen    int64
	UserAgent   string
	DeviceOS    string
	OsVersion   string
	DeviceModel string
}

func (f *Flow) ShowOwnDevices(chatID, tgUserID int64) {
	traffics, err := f.InboundService.GetClientTrafficTgBot(tgUserID)
	if err != nil {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	if len(traffics) == 0 {
		f.ShowRegistrationPrompt(chatID)
		return
	}
	if len(traffics) == 1 {
		f.ShowDevices(chatID, tgUserID, traffics[0].Email)
		return
	}

	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, tu.InlineKeyboardButton(traffic.Email).
			WithCallbackData(f.callback("client_devices "+traffic.Email)))
	}
	f.SendMessage(chatID, "Выберите подписку:", tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...)))
}

func (f *Flow) ShowDevices(chatID, tgUserID int64, email string) {
	if !f.ownsClient(tgUserID, email) {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}

	devices, limit, err := f.listDevices(email)
	if err != nil {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}

	limitText := "∞"
	if limit > 0 {
		limitText = strconv.Itoa(limit)
	}
	message := fmt.Sprintf("📱 <b>Устройства</b>\n\nИспользуется устройств: %d/%s", len(devices), limitText)
	buttons := make([]telego.InlineKeyboardButton, 0, len(devices))
	for _, device := range devices {
		buttons = append(buttons, tu.InlineKeyboardButton(deviceLabel(device)).
			WithCallbackData(f.callback(fmt.Sprintf("client_device %s %d", email, device.ID))))
	}
	if len(buttons) == 0 {
		message += "\n\nНет зарегистрированных устройств."
		f.SendMessage(chatID, message)
		return
	}
	f.SendMessage(chatID, message, tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...)))
}

func (f *Flow) ShowDevice(chatID, tgUserID int64, email string, id int) {
	if !f.ownsClient(tgUserID, email) {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	devices, _, err := f.listDevices(email)
	if err != nil {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	for _, device := range devices {
		if device.ID != id {
			continue
		}
		message := fmt.Sprintf("📱 <b>Устройство</b>\n\n%s\nFingerprint: <code>%s</code>\nПодключено: %s\nПоследний вход: %s",
			deviceLabel(device), device.Fingerprint, formatDeviceTime(device.FirstSeen), formatDeviceTime(device.LastSeen))
		keyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("🗑 Удалить устройство").
				WithCallbackData(f.callback(fmt.Sprintf("client_device_remove %s %d", email, id)))),
		)
		f.SendMessage(chatID, message, keyboard)
		return
	}
	f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
}

func (f *Flow) ConfirmDeviceDelete(chatID, tgUserID int64, email string, id int) {
	if !f.ownsClient(tgUserID, email) {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Да, удалить").
				WithCallbackData(f.callback(fmt.Sprintf("client_device_delete %s %d", email, id))),
			tu.InlineKeyboardButton("Отмена").
				WithCallbackData(f.callback(fmt.Sprintf("client_device %s %d", email, id))),
		),
	)
	f.SendMessage(chatID, "Удалить это устройство?", keyboard)
}

func (f *Flow) DeleteDevice(chatID, tgUserID int64, email string, id int) {
	if !f.ownsClient(tgUserID, email) {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	if err := f.DeleteDeviceByEmail(email, id); err != nil {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	f.ShowDevices(chatID, tgUserID, email)
}

func (f *Flow) ownsClient(tgUserID int64, email string) bool {
	traffics, err := f.InboundService.GetClientTrafficTgBot(tgUserID)
	if err != nil {
		return false
	}
	for _, traffic := range traffics {
		if traffic.Email == email {
			return true
		}
	}
	return false
}

func (f *Flow) listDevices(email string) ([]Device, int, error) {
	return f.ListDevices(email)
}

func (f *Flow) callback(data string) string {
	if f.EncodeCallback != nil {
		return f.EncodeCallback(data)
	}
	return data
}

func deviceLabel(device Device) string {
	name := strings.TrimSpace(device.DeviceModel)
	if name == "" {
		name = strings.TrimSpace(device.DeviceOS)
	}
	if name == "" {
		name = strings.TrimSpace(device.UserAgent)
	}
	if name == "" {
		name = device.Fingerprint
	}
	if name == "" {
		name = strconv.Itoa(device.ID)
	}
	icon := "📱"
	if strings.Contains(strings.ToLower(name), "windows") ||
		strings.Contains(strings.ToLower(device.DeviceOS), "windows") {
		icon = "💻"
	}
	return icon + " " + name
}

func formatDeviceTime(value int64) string {
	if value <= 0 {
		return "—"
	}
	return time.UnixMilli(value).Format("2006-01-02 15:04:05")
}
