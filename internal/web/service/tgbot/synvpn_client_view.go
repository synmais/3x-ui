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
	photo        bool
}

type clientViewState struct {
	mu          sync.Mutex
	message     int
	attachments []int
	current     clientScreen
	back        []clientScreen
	revision    uint64
	paymentID   string
}

type clientViewRenderer struct {
	t     *Tgbot
	actor chatUser
	state *clientViewState
}

var clientViewStates = struct {
	sync.Mutex
	items map[chatUser]*clientViewState
}{items: make(map[chatUser]*clientViewState)}

var clientPaymentTimers = struct {
	sync.Mutex
	items map[string]*time.Timer
}{items: make(map[string]*time.Timer)}

func (t *Tgbot) clientViewForCallback(actor chatUser, sourceMessageID int) *clientViewRenderer {
	// A zero message ID is only used by unit level callback invocations, not by
	// Telegram. Keep those renderers isolated so their state cannot leak between
	// tests or affect a real user's open screen.
	if sourceMessageID == 0 {
		return &clientViewRenderer{t: t, actor: actor, state: &clientViewState{}}
	}
	return t.clientViewForActor(actor)
}

func (t *Tgbot) clientViewForActor(actor chatUser) *clientViewRenderer {
	clientViewStates.Lock()
	view := clientViewStates.items[actor]
	if view == nil {
		view = &clientViewState{}
		clientViewStates.items[actor] = view
	}
	clientViewStates.Unlock()
	return &clientViewRenderer{t: t, actor: actor, state: view}
}

func (t *Tgbot) resetClientView(actor chatUser) {
	clientViewStates.Lock()
	view := clientViewStates.items[actor]
	clientViewStates.Unlock()
	if view == nil {
		return
	}
	(&clientViewRenderer{t: t, actor: actor, state: view}).cancel()
}

func (r *clientViewRenderer) send(chatID int64, text string, markup ...telego.ReplyMarkup) {
	if text == "" || !r.t.IsRunning() {
		return
	}
	var inline *telego.InlineKeyboardMarkup
	if len(markup) > 0 {
		inline, _ = markup[0].(*telego.InlineKeyboardMarkup)
	}
	r.render(clientScreen{text: text, markup: inline}, true)
}

func (r *clientViewRenderer) sendBanner(chatID int64, text string, markup ...telego.ReplyMarkup) {
	if text == "" || !r.t.IsRunning() {
		return
	}
	var inline *telego.InlineKeyboardMarkup
	if len(markup) > 0 {
		inline, _ = markup[0].(*telego.InlineKeyboardMarkup)
	}
	r.render(clientScreen{text: text, markup: inline, photo: true}, true)
}

func (r *clientViewRenderer) sendInstruction(chatID int64, page synvpntgbot.InstructionPage) {
	if !r.t.IsRunning() {
		return
	}
	markup := synVPNInstructionKeyboard(page.Buttons)
	screen := clientScreen{text: page.Text, markup: markup}
	if page.Guide != nil {
		screen.text = "📘 <b>" + page.Guide.Title + "</b>"
		screen.mediaKey = page.Guide.ID
		screen.mediaCaption = page.Guide.Description
	}
	r.render(screen, true)
}

func (r *clientViewRenderer) render(screen clientScreen, push bool) {
	w := r.state
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.message == 0 {
		if !push {
			return
		}
		message, err := r.sendHeader(&screen, false)
		if err != nil {
			logger.Warning("Failed to open client bot screen:", err)
			return
		}
		w.message, w.current = message.MessageID, screen
		w.revision++
		if screen.mediaKey != "" {
			attachments, mediaErr := sendSynVPNInstructionMedia(r.actor.chatID, screen.mediaKey, screen.mediaCaption)
			if mediaErr != nil {
				if !errors.Is(mediaErr, fs.ErrNotExist) && !errors.Is(mediaErr, errNoSynVPNInstructionScreenshots) {
					logger.Warning("Failed to send client instruction screenshots:", mediaErr)
				}
				screen.text += "\n\n" + screen.mediaCaption + "\n\nСкриншоты пока не загружены."
				screen.mediaKey = ""
				if editErr := r.editHeader(w.message, screen, false); editErr != nil {
					logger.Warning("Failed to update instruction without screenshots:", editErr)
				}
			} else {
				w.attachments = attachments
			}
			w.current = screen
		}
		return
	}
	// Keep the registration banner as the view's photo while subsequent
	// registration and payment screens update its caption in place.
	if w.current.photo && screen.mediaKey == "" {
		screen.photo = true
	}
	if push && (screen.text != w.current.text || screen.mediaKey != w.current.mediaKey || screen.photo != w.current.photo || !sameKeyboard(screen.markup, w.current.markup)) {
		w.back = append(w.back, w.current)
	}
	if screen.mediaKey == "" && screen.photo != w.current.photo {
		message, err := r.sendHeader(&screen, len(w.back) > 0)
		if err != nil {
			logger.Warning("Failed to replace client bot screen:", err)
			return
		}
		r.t.deleteMessageTgBot(r.actor.chatID, w.message)
		for _, attachmentID := range w.attachments {
			r.t.deleteMessageTgBot(r.actor.chatID, attachmentID)
		}
		w.message, w.current = message.MessageID, screen
		w.attachments = nil
		w.revision++
		return
	}
	if screen.mediaKey != "" && screen.mediaKey != w.current.mediaKey {
		if !w.current.photo {
			if err := r.editHeader(w.message, screen, len(w.back) > 0); err != nil {
				if !isTelegramUnavailableEditError(err) {
					logger.Warning("Failed to update client instruction heading:", err)
					return
				}
				message, sendErr := r.sendHeader(&screen, len(w.back) > 0)
				if sendErr != nil {
					logger.Warning("Failed to recreate client instruction heading:", sendErr)
					return
				}
				r.t.deleteMessageTgBot(r.actor.chatID, w.message)
				w.message = message.MessageID
			}
		} else {
			message, err := r.sendHeader(&screen, len(w.back) > 0)
			if err != nil {
				logger.Warning("Failed to open client instruction heading:", err)
				return
			}
			r.t.deleteMessageTgBot(r.actor.chatID, w.message)
			w.message = message.MessageID
		}
		for _, messageID := range w.attachments {
			r.t.deleteMessageTgBot(r.actor.chatID, messageID)
		}
		w.attachments = nil
		attachments, err := sendSynVPNInstructionMedia(r.actor.chatID, screen.mediaKey, screen.mediaCaption)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errNoSynVPNInstructionScreenshots) {
				logger.Warning("Failed to send client instruction screenshots:", err)
			}
			screen.text += "\n\n" + screen.mediaCaption + "\n\nСкриншоты пока не загружены."
			screen.mediaKey = ""
			if editErr := r.editHeader(w.message, screen, len(w.back) > 0); editErr != nil {
				logger.Warning("Failed to show instruction without screenshots:", editErr)
			}
		} else {
			w.attachments = attachments
		}
		w.current = screen
		w.revision++
		return
	}

	if err := r.editHeader(w.message, screen, len(w.back) > 0); err != nil {
		logger.Warning("Failed to update client bot screen:", err)
		if !isTelegramUnavailableEditError(err) {
			return
		}
		message, sendErr := r.sendHeader(&screen, len(w.back) > 0)
		if sendErr != nil {
			logger.Warning("Failed to recreate client bot screen:", sendErr)
			return
		}
		r.t.deleteMessageTgBot(r.actor.chatID, w.message)
		w.message = message.MessageID
	}
	if screen.mediaKey == "" && len(w.attachments) > 0 {
		for _, messageID := range w.attachments {
			r.t.deleteMessageTgBot(r.actor.chatID, messageID)
		}
		w.attachments = nil
	}
	w.current = screen
	w.revision++
}

func (r *clientViewRenderer) sendHeader(screen *clientScreen, withBack bool) (*telego.Message, error) {
	markup := r.navigation(screen.markup, withBack)
	if screen.photo {
		message, err := bot.SendPhoto(context.Background(), &telego.SendPhotoParams{
			ChatID: tu.ID(r.actor.chatID), Photo: tu.FileFromBytes(synvpntgbot.MainMenuBanner(), "main-menu-banner.jpg"),
			Caption: screen.text, ParseMode: "HTML", ReplyMarkup: markup,
		})
		if err == nil {
			return message, nil
		}
		logger.Warning("Failed to send client bot banner:", err)
		screen.photo = false
	}
	return bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID: tu.ID(r.actor.chatID), Text: screen.text, ParseMode: "HTML", ReplyMarkup: markup,
	})
}

func (r *clientViewRenderer) editHeader(messageID int, screen clientScreen, withBack bool) error {
	markup := r.navigation(screen.markup, withBack)
	if screen.photo {
		_, err := bot.EditMessageCaption(context.Background(), &telego.EditMessageCaptionParams{
			ChatID: tu.ID(r.actor.chatID), MessageID: messageID, Caption: screen.text,
			ParseMode: "HTML", ReplyMarkup: markup,
		})
		if isTelegramNotModifiedError(err) {
			return nil
		}
		return err
	}
	params := telego.EditMessageTextParams{
		ChatID: tu.ID(r.actor.chatID), MessageID: messageID, Text: screen.text,
		ParseMode: "HTML", ReplyMarkup: markup,
	}
	_, err := bot.EditMessageText(context.Background(), &params)
	if isTelegramNotModifiedError(err) {
		return nil
	}
	return err
}

func (r *clientViewRenderer) schedulePaymentExpiry(paymentID string, expiresAt time.Time) {
	w := r.state
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

		clientViewStates.Lock()
		delete(clientViewStates.items, r.actor)
		clientViewStates.Unlock()
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

func (r *clientViewRenderer) navigation(markup *telego.InlineKeyboardMarkup, withBack bool) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0)
	if markup != nil {
		rows = append(rows, markup.InlineKeyboard...)
	}
	controls := make([]telego.InlineKeyboardButton, 0, 2)
	if withBack {
		controls = append(controls, tu.InlineKeyboardButton("⬅️ Назад").WithCallbackData("client_view_back"))
	}
	controls = append(controls, tu.InlineKeyboardButton("✖️ Отмена").WithCallbackData("client_view_cancel"))
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

func (r *clientViewRenderer) backToPrevious() bool {
	w := r.state
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

func (r *clientViewRenderer) cancel() {
	w := r.state
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
	clientViewStates.Lock()
	delete(clientViewStates.items, r.actor)
	clientViewStates.Unlock()
	if messageID != 0 {
		r.t.deleteMessageTgBot(r.actor.chatID, messageID)
	}
	for _, attachmentID := range attachments {
		r.t.deleteMessageTgBot(r.actor.chatID, attachmentID)
	}
}
