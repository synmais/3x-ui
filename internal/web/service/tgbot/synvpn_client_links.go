package tgbot

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/skip2/go-qrcode"
)

// sendClientSubLinks sends the subscription links for the client to the chat.
func (t *Tgbot) sendClientSubLinks(chatId int64, email string, view ...*clientViewRenderer) {
	subURL, subJsonURL, err := t.buildSubscriptionURLs(email)
	if err != nil {
		t.sendSynVPNClientViewMessage(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error(), view...)
		return
	}
	msg := "Subscription URL:\r\n<code>" + subURL + "</code>"
	if subJsonURL != "" {
		msg += "\r\n\r\nJSON URL:\r\n<code>" + subJsonURL + "</code>"
	}
	inlineKeyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(t.I18nBot("qrCode")).WithCallbackData(t.encodeQuery("client_qr_links "+email)),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🔄 Продлить подписку").WithCallbackData("client_renew "+email),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("➕ Добавить подписку").WithCallbackData("client_add_subscription"),
		),
	)
	if len(view) > 0 && view[0] != nil {
		view[0].send(chatId, msg, inlineKeyboard)
	} else {
		t.SendMsgToTgbot(chatId, msg, inlineKeyboard)
	}
}

// clientSubLinks builds the subscription's links in-process for the host subURL
// names; fetching subURL instead fails whenever that host does not resolve here.
func (t *Tgbot) clientSubLinks(email, subURL string) ([]string, error) {
	u, err := url.Parse(subURL)
	if err != nil {
		return nil, err
	}
	_, client, err := t.inboundService.GetClientByEmail(email)
	if err != nil || client == nil {
		return nil, errors.New("client not found")
	}
	return t.inboundService.GetSubLinks(u.Hostname(), client.SubID)
}

// sendClientIndividualLinks sends the subscription's individual links to the user
func (t *Tgbot) sendClientIndividualLinks(chatId int64, email string, view ...*clientViewRenderer) {
	subURL, _, err := t.buildSubscriptionURLs(email)
	if err != nil {
		t.sendSynVPNClientViewMessage(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error(), view...)
		return
	}
	links, err := t.clientSubLinks(email, subURL)
	if err != nil {
		t.sendSynVPNClientViewMessage(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error(), view...)
		return
	}
	if len(links) == 0 {
		t.sendSynVPNClientViewMessage(chatId, t.I18nBot("tgbot.noResult"), view...)
		return
	}

	// Send in chunks to respect message length; use monospace formatting
	const maxPerMessage = 50
	for i := 0; i < len(links); i += maxPerMessage {
		j := min(i+maxPerMessage, len(links))
		chunk := links[i:j]
		var msg strings.Builder
		msg.WriteString(t.I18nBot("subscription.individualLinks"))
		msg.WriteString(":\r\n")
		for _, link := range chunk {
			// wrap each link in <code>
			msg.WriteString("<code>")
			msg.WriteString(link)
			msg.WriteString("</code>\r\n")
		}
		if len(view) > 0 && view[0] != nil && i == 0 {
			view[0].send(chatId, msg.String())
		} else {
			t.SendMsgToTgbot(chatId, msg.String())
		}
	}
}

func (t *Tgbot) sendSynVPNClientViewMessage(chatID int64, message string, view ...*clientViewRenderer) {
	if len(view) > 0 && view[0] != nil {
		view[0].send(chatID, message)
		return
	}
	t.SendMsgToTgbot(chatID, message)
}

func (t *Tgbot) sendSynVPNClientViewWithMarkup(chatID int64, message string, view *clientViewRenderer, markup telego.ReplyMarkup) {
	if view != nil {
		view.send(chatID, message, markup)
		return
	}
	t.SendMsgToTgbot(chatID, message, markup)
}

// sendClientQRLinks generates QR images for subscription URL, JSON URL, and a few individual links, then sends them
func (t *Tgbot) sendClientQRLinks(chatId int64, email string, view ...*clientViewRenderer) {
	subURL, subJsonURL, err := t.buildSubscriptionURLs(email)
	if err != nil {
		t.sendSynVPNClientViewMessage(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error(), view...)
		return
	}

	// Helper to create QR PNG bytes from content
	createQR := func(content string, size int) ([]byte, error) {
		if size <= 0 {
			size = 256
		}
		return qrcode.Encode(content, qrcode.Medium, size)
	}

	type qrDocument struct {
		filename string
		content  []byte
	}
	documents := make([]qrDocument, 0, 7)
	appendQR := func(content, filename string) {
		png, err := createQR(content, 320)
		if err != nil {
			t.SendMsgToTgbot(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error())
			return
		}
		documents = append(documents, qrDocument{filename: filename, content: png})
	}
	appendQR(subURL, "sub.png")

	// Send JSON URL QR (filename: subjson.png) when available
	if subJsonURL != "" {
		appendQR(subJsonURL, "subjson.png")
	}

	// Also generate a few individual links' QRs (first up to 5)
	if links, err := t.clientSubLinks(email, subURL); err == nil {
		max := min(len(links), 5)
		for i := range max {
			appendQR(links[i], email+".png")
		}
	}

	hasClientView := len(view) > 0 && view[0] != nil
	if hasClientView {
		view[0].cancel()
	}
	t.SendMsgToTgbot(chatId, t.I18nBot("tgbot.answers.qrCodeForClient", "Email=="+email))
	if len(documents) == 0 {
		if hasClientView {
			t.SendMsgToTgbot(chatId, "Не удалось подготовить QR-коды.", clientBackToMenuKeyboard())
		} else {
			t.SendMsgToTgbot(chatId, "Не удалось подготовить QR-коды.")
		}
		return
	}
	lastSent := false
	for i, qr := range documents {
		document := tu.Document(tu.ID(chatId), tu.FileFromBytes(qr.content, qr.filename))
		if hasClientView && i == len(documents)-1 {
			document.ReplyMarkup = clientBackToMenuKeyboard()
		}
		if _, err := bot.SendDocument(context.Background(), document); err == nil {
			lastSent = i == len(documents)-1
		} else {
			lastSent = false
			t.SendMsgToTgbot(chatId, t.I18nBot("tgbot.answers.errorOperation")+"\r\n"+err.Error())
		}
		if i < len(documents)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if hasClientView && !lastSent {
		t.SendMsgToTgbot(chatId, "QR-коды готовы.", clientBackToMenuKeyboard())
	}
}

func clientBackToMenuKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("⬅️ Назад в меню").WithCallbackData("client_view_home"),
	))
}
