package tgbot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// showClientDevices displays the devices registered for one of the caller's
// clients. The raw HWID is intentionally not displayed: it is stored only as a
// hash and is not recoverable after registration.
func (t *Tgbot) showClientDevices(chatID, tgUserID int64, email string, messageID ...int) {
	if !t.ownsClient(tgUserID, email) {
		t.sendCallbackError(chatID)
		return
	}

	record, err := t.clientService.GetRecordByEmail(nil, email)
	if err != nil {
		t.sendCallbackError(chatID)
		return
	}
	hwids, err := t.clientService.ListClientHwids(email)
	if err != nil {
		t.sendCallbackError(chatID)
		return
	}

	limit := "∞"
	if record.LimitHwid > 0 {
		limit = strconv.Itoa(record.LimitHwid)
	}
	text := t.I18nBot("tgbot.devices.list", "Used=="+strconv.Itoa(len(hwids)), "DeviceWord=="+russianDeviceWord(len(hwids)), "Limit=="+limit)
	if len(hwids) == 0 {
		text += "\r\n" + t.I18nBot("tgbot.devices.empty")
	}

	buttons := make([]telego.InlineKeyboardButton, 0, len(hwids))
	for i, device := range hwids {
		buttons = append(buttons, tu.InlineKeyboardButton(deviceName(device, i+1)).WithCallbackData(
			fmt.Sprintf("client_device %s %d", email, device.Id),
		))
	}
	keyboard := tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...))
	if len(messageID) > 0 {
		t.editMessageTgBot(chatID, messageID[0], text, keyboard)
		return
	}
	t.SendMsgToTgbot(chatID, text, keyboard)
}

func (t *Tgbot) showOwnDevices(chatID, tgUserID int64) {
	traffics, err := t.inboundService.GetClientTrafficTgBot(tgUserID)
	if err != nil {
		t.sendCallbackError(chatID)
		return
	}
	if len(traffics) == 0 {
		t.showRegistrationPrompt(chatID)
		return
	}
	if len(traffics) == 1 {
		t.showClientDevices(chatID, tgUserID, traffics[0].Email)
		return
	}

	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, tu.InlineKeyboardButton(traffic.Email).WithCallbackData("client_devices "+traffic.Email))
	}
	t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.devices.selectClient"), tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...)))
}

func (t *Tgbot) showDevice(chatID, tgUserID int64, email string, deviceID, messageID int) {
	if !t.ownsClient(tgUserID, email) {
		t.sendCallbackError(chatID)
		return
	}
	hwids, err := t.clientService.ListClientHwids(email)
	if err != nil {
		t.sendCallbackError(chatID)
		return
	}
	for i, device := range hwids {
		if device.Id != deviceID {
			continue
		}
		text := t.I18nBot("tgbot.devices.detail",
			"Name=="+html.EscapeString(deviceName(device, i+1)),
			"FirstSeen=="+formatDeviceTime(device.FirstSeen),
			"LastSeen=="+formatDeviceTime(device.LastSeen),
			"Details=="+t.deviceDetails(device),
		)
		keyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.deleteDevice")).WithCallbackData(fmt.Sprintf("client_device_remove %s %d", email, deviceID))),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.back")).WithCallbackData("client_devices "+email)),
		)
		t.editMessageTgBot(chatID, messageID, text, keyboard)
		return
	}
	t.showClientDevices(chatID, tgUserID, email, messageID)
}

func (t *Tgbot) confirmDeviceRemoval(chatID, tgUserID int64, email string, deviceID, messageID int) {
	if !t.ownsClient(tgUserID, email) {
		t.sendCallbackError(chatID)
		return
	}
	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("client_device "+email+" "+strconv.Itoa(deviceID))),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.confirmDeleteDevice")).WithCallbackData("client_device_delete "+email+" "+strconv.Itoa(deviceID))),
	)
	t.editMessageTgBot(chatID, messageID, t.I18nBot("tgbot.devices.confirmRemove"), keyboard)
}

func (t *Tgbot) deleteDevice(chatID, tgUserID int64, callbackID, email string, deviceID, messageID int) {
	if !t.ownsClient(tgUserID, email) || t.clientService.DeleteClientHwid(email, deviceID) != nil {
		t.sendCallbackError(chatID)
		return
	}
	t.sendCallbackAnswerTgBot(callbackID, t.I18nBot("tgbot.answers.deviceDeleted"))
	t.showClientDevices(chatID, tgUserID, email, messageID)
}

func (t *Tgbot) ownsClient(tgUserID int64, email string) bool {
	traffics, err := t.inboundService.GetClientTrafficTgBot(tgUserID)
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

func (t *Tgbot) sendCallbackError(chatID int64) {
	t.SendMsgToTgbot(chatID, t.I18nBot("tgbot.answers.errorOperation"))
}

func deviceName(device service.ClientHwidInfo, position int) string {
	for _, value := range []string{device.DeviceModel, strings.TrimSpace(device.DeviceOS + " " + device.OsVersion), device.UserAgent} {
		if value = strings.TrimSpace(value); value != "" {
			return truncateDeviceLabel(value, 48)
		}
	}
	return fmt.Sprintf("Device %d", position)
}

func (t *Tgbot) deviceDetails(device service.ClientHwidInfo) string {
	var details []string
	if device.DeviceOS != "" {
		details = append(details, "\r\n"+t.I18nBot("tgbot.devices.os", "Value=="+html.EscapeString(device.DeviceOS)))
	}
	if device.OsVersion != "" {
		details = append(details, "\r\n"+t.I18nBot("tgbot.devices.osVersion", "Value=="+html.EscapeString(device.OsVersion)))
	}
	if device.DeviceModel != "" {
		details = append(details, "\r\n"+t.I18nBot("tgbot.devices.model", "Value=="+html.EscapeString(device.DeviceModel)))
	}
	if device.UserAgent != "" {
		details = append(details, "\r\n"+t.I18nBot("tgbot.devices.userAgent", "Value=="+html.EscapeString(device.UserAgent)))
	}
	return strings.Join(details, "")
}

func formatDeviceTime(timestamp int64) string {
	if timestamp <= 0 {
		return "—"
	}
	return time.UnixMilli(timestamp).Format("02.01.2006 15:04")
}

func truncateDeviceLabel(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}
