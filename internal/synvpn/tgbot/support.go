package tgbot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tu "github.com/mymmrac/telego/telegoutil"
)

func (f *Flow) AskSupportMessage(chatID int64) {
	f.SendMessage(chatID, "Опишите проблему одним сообщением. Я передам его администраторам.")
}

func (f *Flow) SendSupportRequest(userID int64, displayName, message string) {
	message = limitSupportText(message, 3500)
	text := fmt.Sprintf("🆘 <b>Обращение в тех. поддержку</b>\nПользователь: %s\nTelegram ID: <code>%d</code>\n\n%s",
		html.EscapeString(displayName), userID, html.EscapeString(strings.TrimSpace(message)))
	callback := "support_reply " + strconv.FormatInt(userID, 10)
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Ответить").WithCallbackData(callback),
	))
	f.SendToAdmins(text, keyboard)
	f.SendMessage(userID, "Сообщение передано администраторам. Ответ придёт сюда.")
}

func (f *Flow) AskSupportReply(adminChatID, userID int64) {
	f.SendMessage(adminChatID, fmt.Sprintf("Введите ответ пользователю %d одним сообщением.", userID))
}

func (f *Flow) SendSupportReply(adminChatID, userID int64, message string) {
	message = limitSupportText(message, 3500)
	if message == "" {
		f.SendMessage(adminChatID, "Ответ не может быть пустым. Введите сообщение ещё раз.")
		return
	}
	f.SendMessage(userID, "💬 Ответ тех. поддержки:\n\n"+html.EscapeString(message))
	f.SendMessage(adminChatID, "Ответ отправлен пользователю.")
}

func limitSupportText(message string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(message))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}
