package tgbot

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
)

const synVPNSupportAlbumDebounce = 900 * time.Millisecond

func (t *Tgbot) handleSynVPNSupportInput(message *telego.Message, actor chatUser, state string) bool {
	if state != "awaiting_support_message" && !strings.HasPrefix(state, "awaiting_support_reply ") {
		return false
	}
	t.processSynVPNSupportInput(message, actor, state)
	return true
}

func (t *Tgbot) handleSynVPNSupportReply(query *telego.CallbackQuery, actor chatUser, userIDText string) {
	userID, err := strconv.ParseInt(userIDText, 10, 64)
	if err != nil || userID <= 0 {
		t.sendCallbackAnswerTgBot(query.ID, t.I18nBot("tgbot.answers.errorOperation"))
		return
	}
	t.startSupportReply(actor, userID)
	t.sendCallbackAnswerTgBot(query.ID, "Введите ответ пользователю")
}

type synVPNSupportAlbum struct {
	groupID    string
	message    telego.Message
	messageIDs []int
	state      string
	timer      *time.Timer
}

var synVPNSupportAlbums = struct {
	sync.Mutex
	items map[chatUser]*synVPNSupportAlbum
}{items: make(map[chatUser]*synVPNSupportAlbum)}

func (t *Tgbot) processSynVPNSupportInput(message *telego.Message, actor chatUser, state string) {
	if message.MediaGroupID != "" && isSynVPNSupportAttachment(message) {
		t.bufferSynVPNSupportAlbum(message, actor, state)
		return
	}
	if !isSynVPNSupportAttachment(message) && strings.TrimSpace(message.Text) == "" {
		if state == "awaiting_support_message" {
			t.SendMsgToTgbot(message.Chat.ID, "Отправьте текст или скриншот. Несколько скриншотов можно прислать одним альбомом.")
		} else {
			t.SendMsgToTgbot(message.Chat.ID, "Отправьте текстовый ответ или скриншот.")
		}
		return
	}
	ids := []int(nil)
	if isSynVPNSupportAttachment(message) {
		ids = []int{message.MessageID}
	}
	t.submitSynVPNSupportInput(message, actor, state, ids)
}

func isSynVPNSupportAttachment(message *telego.Message) bool {
	return len(message.Photo) > 0 || message.Document != nil
}

func (t *Tgbot) bufferSynVPNSupportAlbum(message *telego.Message, actor chatUser, state string) {
	var previousGroup string
	synVPNSupportAlbums.Lock()
	album := synVPNSupportAlbums.items[actor]
	if album != nil && album.groupID != message.MediaGroupID {
		previousGroup = album.groupID
	}
	if previousGroup != "" {
		synVPNSupportAlbums.Unlock()
		t.finalizeSynVPNSupportAlbum(actor, previousGroup)
		return
	}
	if album == nil {
		first := *message
		album = &synVPNSupportAlbum{groupID: message.MediaGroupID, message: first, state: state}
		album.timer = time.AfterFunc(synVPNSupportAlbumDebounce, func() {
			t.finalizeSynVPNSupportAlbum(actor, message.MediaGroupID)
		})
		synVPNSupportAlbums.items[actor] = album
	}
	album.messageIDs = append(album.messageIDs, message.MessageID)
	album.timer.Reset(synVPNSupportAlbumDebounce)
	synVPNSupportAlbums.Unlock()
}

func (t *Tgbot) finalizeSynVPNSupportAlbum(actor chatUser, groupID string) {
	synVPNSupportAlbums.Lock()
	album := synVPNSupportAlbums.items[actor]
	if album == nil || album.groupID != groupID {
		synVPNSupportAlbums.Unlock()
		return
	}
	if album.timer != nil {
		album.timer.Stop()
	}
	delete(synVPNSupportAlbums.items, actor)
	message, state := album.message, album.state
	ids := slices.Clone(album.messageIDs)
	synVPNSupportAlbums.Unlock()

	current, ok := userStateMgr.get(actor)
	if !ok || current != state {
		return
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	t.submitSynVPNSupportInput(&message, actor, state, ids)
}

func (t *Tgbot) submitSynVPNSupportInput(message *telego.Message, actor chatUser, state string, messageIDs []int) {
	userStateMgr.clear(actor)
	switch state {
	case "awaiting_support_message":
		t.clientViewForCallback(actor, -1).cancel()
		name := "Пользователь"
		if message.From != nil {
			name = strings.TrimSpace(strings.TrimSpace(message.From.FirstName) + " " + strings.TrimSpace(message.From.LastName))
			if message.From.Username != "" {
				name += " (@" + message.From.Username + ")"
			}
		}
		t.synVPNFlow().SendSupportRequest(actor.userID, name, message.Text, messageIDs...)
	default:
		if userIDText, ok := strings.CutPrefix(state, "awaiting_support_reply "); ok {
			userID, err := strconv.ParseInt(userIDText, 10, 64)
			if err != nil || userID <= 0 {
				t.SendMsgToTgbot(message.Chat.ID, "Не удалось определить пользователя для ответа.")
				return
			}
			t.synVPNFlow().SendSupportReply(message.Chat.ID, userID, message.Text, messageIDs...)
		}
	}
}
