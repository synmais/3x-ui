package tgbot

import (
	"context"
	"errors"
	"io/fs"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type clientScreen struct {
	text         string
	markup       *telego.InlineKeyboardMarkup
	mediaKey     string
	mediaCaption string
}

type clientWindow struct {
	mu          sync.Mutex
	message     int
	attachments []int
	current     clientScreen
	back        []clientScreen
	revision    uint64
	paymentID   string
}

type clientWindowRenderer struct {
	t      *Tgbot
	actor  chatUser
	source int
	window *clientWindow
}

var clientWindows = struct {
	sync.Mutex
	items map[chatUser]*clientWindow
}{items: make(map[chatUser]*clientWindow)}

var clientPaymentTimers = struct {
	sync.Mutex
	items map[string]*time.Timer
}{items: make(map[string]*time.Timer)}

func (t *Tgbot) clientWindowForCallback(actor chatUser, sourceMessageID int) *clientWindowRenderer {
	// A zero message ID is only used by unit level callback invocations, not by
	// Telegram. Keep those renderers isolated so their state cannot leak between
	// tests or affect a real user's open screen.
	if sourceMessageID == 0 {
		return &clientWindowRenderer{t: t, actor: actor, source: sourceMessageID, window: &clientWindow{}}
	}
	clientWindows.Lock()
	window := clientWindows.items[actor]
	if window == nil {
		window = &clientWindow{}
		clientWindows.items[actor] = window
	}
	clientWindows.Unlock()
	return &clientWindowRenderer{t: t, actor: actor, source: sourceMessageID, window: window}
}

func (r *clientWindowRenderer) send(chatID int64, text string, markup ...telego.ReplyMarkup) {
	if text == "" || !r.t.IsRunning() {
		return
	}
	var inline *telego.InlineKeyboardMarkup
	if len(markup) > 0 {
		inline, _ = markup[0].(*telego.InlineKeyboardMarkup)
	}
	r.render(clientScreen{text: text, markup: inline}, true)
}

func (r *clientWindowRenderer) sendInstruction(chatID int64, page synvpntgbot.InstructionPage) {
	if !r.t.IsRunning() {
		return
	}
	markup := instructionKeyboard(page.Buttons)
	screen := clientScreen{text: page.Text, markup: markup}
	if page.Guide != nil {
		screen.text = "📘 <b>" + page.Guide.Title + "</b>"
		screen.mediaKey = page.Guide.ID
		screen.mediaCaption = page.Guide.Title + "\n\n" + page.Guide.Description
	}
	r.render(screen, true)
}

func (r *clientWindowRenderer) render(screen clientScreen, push bool) {
	w := r.window
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.message == 0 {
		if !push {
			return
		}
		if screen.mediaKey != "" {
			attachments, err := sendClientInstructionMedia(r.actor.chatID, screen.mediaKey, screen.mediaCaption)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errNoInstructionScreenshots) {
					logger.Warning("Failed to send client instruction screenshots:", err)
				}
				screen.text = screen.mediaCaption + "\n\nСкриншоты пока не загружены."
				screen.mediaKey = ""
			} else {
				w.attachments = attachments
			}
		}
		message, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: tu.ID(r.actor.chatID), Text: screen.text, ParseMode: "HTML",
			ReplyMarkup: r.navigation(screen.markup, false),
		})
		if err != nil {
			for _, messageID := range w.attachments {
				r.t.deleteMessageTgBot(r.actor.chatID, messageID)
			}
			w.attachments = nil
			logger.Warning("Failed to open client bot screen:", err)
			return
		}
		w.message, w.current = message.MessageID, screen
		w.revision++
		return
	}
	if push && (screen.text != w.current.text || screen.mediaKey != w.current.mediaKey || !sameKeyboard(screen.markup, w.current.markup)) {
		w.back = append(w.back, w.current)
	}
	if screen.mediaKey != w.current.mediaKey {
		var newAttachments []int
		if screen.mediaKey != "" {
			attachments, err := sendClientInstructionMedia(r.actor.chatID, screen.mediaKey, screen.mediaCaption)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errNoInstructionScreenshots) {
					logger.Warning("Failed to send client instruction screenshots:", err)
				}
				screen.text = screen.mediaCaption + "\n\nСкриншоты пока не загружены."
				screen.mediaKey = ""
			} else {
				newAttachments = attachments
			}
		}
		if screen.mediaKey != "" {
			message, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: tu.ID(r.actor.chatID), Text: screen.text, ParseMode: "HTML",
				ReplyMarkup: r.navigation(screen.markup, len(w.back) > 0),
			})
			if err != nil {
				for _, messageID := range newAttachments {
					r.t.deleteMessageTgBot(r.actor.chatID, messageID)
				}
				logger.Warning("Failed to open client instruction controls:", err)
				return
			}
			oldMessage, oldAttachments := w.message, w.attachments
			if oldMessage != 0 {
				r.t.deleteMessageTgBot(r.actor.chatID, oldMessage)
			}
			for _, messageID := range oldAttachments {
				r.t.deleteMessageTgBot(r.actor.chatID, messageID)
			}
			w.message = message.MessageID
			w.attachments = newAttachments
			w.current = screen
			w.revision++
			return
		}
	}
	if len(w.attachments) > 0 && screen.mediaKey == "" {
		for _, messageID := range w.attachments {
			r.t.deleteMessageTgBot(r.actor.chatID, messageID)
		}
		w.attachments = nil
	}
	if w.message == 0 {
		message, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: tu.ID(r.actor.chatID), Text: screen.text, ParseMode: "HTML",
			ReplyMarkup: r.navigation(screen.markup, len(w.back) > 0),
		})
		if err != nil {
			logger.Warning("Failed to open client bot screen:", err)
			return
		}
		w.message, w.current = message.MessageID, screen
		w.revision++
		return
	}
	markup := r.navigation(screen.markup, len(w.back) > 0)
	params := telego.EditMessageTextParams{
		ChatID: tu.ID(r.actor.chatID), MessageID: w.message, Text: screen.text,
		ParseMode: "HTML", ReplyMarkup: markup,
	}
	if _, err := bot.EditMessageText(context.Background(), &params); err != nil && !isTelegramNotModifiedError(err) {
		logger.Warning("Failed to update client bot screen:", err)
		return
	}
	w.current = screen
	w.revision++
}

func (r *clientWindowRenderer) schedulePaymentExpiry(paymentID string, expiresAt time.Time) {
	w := r.window
	w.mu.Lock()
	messageID, revision := w.message, w.revision
	w.paymentID = paymentID
	w.mu.Unlock()
	if messageID == 0 {
		return
	}

	clientPaymentTimers.Lock()
	if existing := clientPaymentTimers.items[paymentID]; existing != nil {
		existing.Stop()
	}
	clientPaymentTimers.items[paymentID] = time.AfterFunc(time.Until(expiresAt), func() {
		clientPaymentTimers.Lock()
		delete(clientPaymentTimers.items, paymentID)
		clientPaymentTimers.Unlock()

		w.mu.Lock()
		if w.paymentID != paymentID {
			w.mu.Unlock()
			return
		}
		if w.message != messageID || w.revision != revision {
			w.paymentID = ""
			w.mu.Unlock()
			r.t.SendMsgToTgbot(r.actor.chatID, "⏳ Время оплаты вышло. Откройте меню «Подписка», чтобы создать новый платёж.")
			return
		}
		w.message = 0
		w.current = clientScreen{}
		w.back = nil
		w.paymentID = ""
		w.mu.Unlock()

		clientWindows.Lock()
		delete(clientWindows.items, r.actor)
		clientWindows.Unlock()
		r.t.deleteMessageTgBot(r.actor.chatID, messageID)
		r.t.SendMsgToTgbot(r.actor.chatID, "⏳ Время оплаты вышло. Откройте меню «Подписка», чтобы создать новый платёж.")
	})
	clientPaymentTimers.Unlock()
}

func cancelClientPaymentExpiry(paymentID string) {
	if paymentID == "" {
		return
	}
	clientPaymentTimers.Lock()
	if timer := clientPaymentTimers.items[paymentID]; timer != nil {
		timer.Stop()
		delete(clientPaymentTimers.items, paymentID)
	}
	clientPaymentTimers.Unlock()
}

func (r *clientWindowRenderer) navigation(markup *telego.InlineKeyboardMarkup, withBack bool) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0)
	if markup != nil {
		rows = append(rows, markup.InlineKeyboard...)
	}
	controls := make([]telego.InlineKeyboardButton, 0, 2)
	if withBack {
		controls = append(controls, tu.InlineKeyboardButton("⬅️ Назад").WithCallbackData("client_window_back"))
	}
	controls = append(controls, tu.InlineKeyboardButton("✖️ Отмена").WithCallbackData("client_window_cancel"))
	rows = append(rows, controls)
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func sameKeyboard(a, b *telego.InlineKeyboardMarkup) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.InlineKeyboard) != len(b.InlineKeyboard) {
		return false
	}
	for i := range a.InlineKeyboard {
		if len(a.InlineKeyboard[i]) != len(b.InlineKeyboard[i]) {
			return false
		}
		for j := range a.InlineKeyboard[i] {
			if a.InlineKeyboard[i][j].Text != b.InlineKeyboard[i][j].Text || a.InlineKeyboard[i][j].CallbackData != b.InlineKeyboard[i][j].CallbackData {
				return false
			}
		}
	}
	return true
}

func (r *clientWindowRenderer) backToPrevious() bool {
	w := r.window
	w.mu.Lock()
	if len(w.back) == 0 {
		w.mu.Unlock()
		return false
	}
	last := len(w.back) - 1
	screen := w.back[last]
	w.back = w.back[:last]
	w.mu.Unlock()
	r.render(screen, false)
	return true
}

func (r *clientWindowRenderer) cancel() {
	w := r.window
	w.mu.Lock()
	messageID := w.message
	attachments := w.attachments
	paymentID := w.paymentID
	w.message = 0
	w.current = clientScreen{}
	w.back = nil
	w.paymentID = ""
	w.revision++
	w.mu.Unlock()
	cancelClientPaymentExpiry(paymentID)
	clientWindows.Lock()
	delete(clientWindows.items, r.actor)
	clientWindows.Unlock()
	if messageID != 0 {
		r.t.deleteMessageTgBot(r.actor.chatID, messageID)
	}
	for _, attachmentID := range attachments {
		r.t.deleteMessageTgBot(r.actor.chatID, attachmentID)
	}
}
