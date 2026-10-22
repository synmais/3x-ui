package tgbot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func (f *Flow) AskSupportMessage(chatID int64) {
	f.SendMessage(chatID, "Опишите проблему или отправьте скриншоты (можно несколько одним альбомом; подпись можно добавить к фото).")
}

func (f *Flow) SendSupportRequest(userID int64, displayName, message string, mediaIDs ...int) {
	message = limitSupportText(message, 3500)
	if message == "" && len(mediaIDs) == 0 {
		return
	}
	text := fmt.Sprintf("🆘 <b>Обращение в тех. поддержку</b>\nПользователь: %s\nTelegram ID: <code>%d</code>\n\n%s",
		html.EscapeString(displayName), userID, html.EscapeString(strings.TrimSpace(message)))
	callback := "support_reply " + strconv.FormatInt(userID, 10)
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Ответить").WithCallbackData(callback),
	))
	if len(mediaIDs) > 0 && f.SendSupportMediaToAdmins != nil {
		f.SendToAdmins(text)
		f.SendSupportMediaToAdmins(userID, mediaIDs)
		// Telegram attaches inline keyboards to messages, not media groups. An
		// invisible placeholder keeps the reply button after the album without
		// adding another visible instruction to the admin chat.
		f.SendToAdmins("\u2063", keyboard)
	} else {
		f.SendToAdmins(text, keyboard)
	}
	f.SendMessage(userID, "Сообщение передано администраторам. Ответ придёт сюда.", supportBackToMenuKeyboard())
}

func (f *Flow) AskSupportReply(adminChatID, userID int64) {
	f.SendMessage(adminChatID, fmt.Sprintf("Введите ответ пользователю %d текстом или отправьте скриншоты одним альбомом (подпись можно добавить к фото).", userID))
}

func (f *Flow) SendSupportReply(adminChatID, userID int64, message string, mediaIDs ...int) {
	message = limitSupportText(message, 3500)
	if message == "" && len(mediaIDs) == 0 {
		f.SendMessage(adminChatID, "Ответ не может быть пустым. Введите текст или приложите скриншот ещё раз.")
		return
	}
	text := "💬 Ответ тех. поддержки:"
	if message != "" {
		text += "\n\n" + html.EscapeString(message)
	}
	if len(mediaIDs) > 0 {
		f.SendMessage(userID, text, supportBackToMenuKeyboard())
		if f.CopySupportMedia == nil {
			f.SendMessage(adminChatID, "Не удалось отправить вложение пользователю. Попробуйте ещё раз.")
			return
		}
		if err := f.CopySupportMedia(adminChatID, userID, mediaIDs); err != nil {
			f.SendMessage(adminChatID, "Не удалось отправить вложение пользователю. Попробуйте ещё раз.")
			return
		}
	} else {
		f.SendMessage(userID, text, supportBackToMenuKeyboard())
	}
	f.SendMessage(adminChatID, "Ответ отправлен пользователю.")
}

func supportBackToMenuKeyboard() telego.ReplyMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("⬅️ Назад в меню").WithCallbackData("client_view_home"),
	))
}

func limitSupportText(message string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(message))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}
