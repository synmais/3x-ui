package tgbot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

var noticeState = struct {
	sync.Mutex
	traffic map[string]bool
	expiry  map[string]bool
}{traffic: make(map[string]bool), expiry: make(map[string]bool)}

func (f *Flow) NotifyNewDevice(telegramID int64, email string, registered, limit int) {
	if telegramID == 0 {
		return
	}
	message := fmt.Sprintf("📱 К вашей подписке <b>%s</b> подключено новое устройство.", html.EscapeString(email))
	var keyboard telego.ReplyMarkup
	if limit > 0 {
		message += fmt.Sprintf("\nУстройств: %d/%d.", registered, limit)
		if registered >= limit {
			message += "\n\nДостигнут лимит устройств."
			keyboard = renewalKeyboard(email, "Улучшить тариф")
		}
	} else {
		message += fmt.Sprintf("\nЗарегистрировано устройств: %d.", registered)
	}
	if keyboard == nil {
		f.SendMessage(telegramID, message)
		return
	}
	f.SendMessage(telegramID, message, keyboard)
}

func (f *Flow) NotifyDeviceLimitReached(telegramID int64, email string, registered, limit int) {
	if telegramID == 0 {
		return
	}
	message := fmt.Sprintf("⛔ Достигнут лимит устройств по подписке <b>%s</b>: %d/%d.\nУдалите ненужное устройство, чтобы подключить новое.",
		html.EscapeString(email), registered, limit)
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("📱 Управление устройствами").WithCallbackData(f.callback("client_devices " + email)),
	))
	f.SendMessage(telegramID, message, keyboard)
}

func (f *Flow) NotifySubscribers() {
	if f.ListSubscriptions == nil {
		return
	}
	subscriptions, err := f.ListSubscriptions()
	if err != nil {
		return
	}
	for _, sub := range subscriptions {
		if sub.TelegramID == 0 {
			continue
		}
		f.notifyTraffic(sub)
		f.notifyExpiry(sub)
	}
}

func (f *Flow) notifyTraffic(sub SubscriberSubscription) {
	if !sub.TrafficActive || sub.TrafficTotal <= 0 {
		return
	}
	noticeState.Lock()
	if float64(sub.TrafficUsed)/float64(sub.TrafficTotal) < .9 {
		delete(noticeState.traffic, sub.Email)
		noticeState.Unlock()
		return
	}
	if noticeState.traffic[sub.Email] {
		noticeState.Unlock()
		return
	}
	noticeState.traffic[sub.Email] = true
	noticeState.Unlock()

	percent := int(float64(sub.TrafficUsed) / float64(sub.TrafficTotal) * 100)
	message := fmt.Sprintf("⚠️ Использовано %d%% трафика по подписке <b>%s</b>: %s из %s.",
		percent, html.EscapeString(sub.Email), formatNoticeTraffic(sub.TrafficUsed), formatNoticeTraffic(sub.TrafficTotal))
	f.SendMessage(sub.TelegramID, message, renewalKeyboard(sub.Email, "Улучшить тариф"))
}

func formatNoticeTraffic(bytes int64) string {
	units := []string{"Б", "КБ", "МБ", "ГБ", "ТБ", "ПБ"}
	size, unit := float64(bytes), 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	value := strconv.FormatFloat(size, 'f', 1, 64)
	value = strings.TrimSuffix(strings.TrimSuffix(value, "0"), ".")
	return strings.ReplaceAll(value, ".", ",") + " " + units[unit]
}

func (f *Flow) notifyExpiry(sub SubscriberSubscription) {
	if sub.ExpiryTime <= 0 {
		return
	}
	today := time.Now()
	expiry := time.UnixMilli(sub.ExpiryTime).In(today.Location())
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 0, 0, 0, 0, time.UTC)
	days := int(expiryDate.Sub(todayDate).Hours() / 24)
	if days != 7 && days != 3 && days != 2 && days != 1 && days != 0 {
		return
	}
	key := fmt.Sprintf("%s:%d:%d", sub.Email, sub.ExpiryTime, days)
	noticeState.Lock()
	if noticeState.expiry[key] {
		noticeState.Unlock()
		return
	}
	noticeState.expiry[key] = true
	noticeState.Unlock()

	message := fmt.Sprintf("⏳ До окончания подписки <b>%s</b> осталось %d %s.", html.EscapeString(sub.Email), days, russianDayWord(days))
	f.SendMessage(sub.TelegramID, message, renewalKeyboard(sub.Email, "Продлить подписку"))
}

func renewalKeyboard(email, label string) telego.ReplyMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(label).WithCallbackData("client_renew " + email),
	))
}
