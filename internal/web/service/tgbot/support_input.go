package tgbot

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
)

const supportAlbumDebounce = 900 * time.Millisecond

type supportAlbum struct {
	groupID    string
	message    telego.Message
	messageIDs []int
	state      string
	timer      *time.Timer
}

var supportAlbums = struct {
	sync.Mutex
	items map[chatUser]*supportAlbum
}{items: make(map[chatUser]*supportAlbum)}

func (t *Tgbot) handleSupportInput(message *telego.Message, actor chatUser, state string) {
	if message.MediaGroupID != "" && isSupportAttachment(message) {
		t.bufferSupportAlbum(message, actor, state)
		return
	}
	if !isSupportAttachment(message) && strings.TrimSpace(message.Text) == "" {
		if state == "awaiting_support_message" {
			t.SendMsgToTgbot(message.Chat.ID, "Отправьте текст или скриншот. Несколько скриншотов можно прислать одним альбомом.")
		} else {
			t.SendMsgToTgbot(message.Chat.ID, "Отправьте текстовый ответ или скриншот.")
		}
		return
	}
	ids := []int(nil)
	if isSupportAttachment(message) {
		ids = []int{message.MessageID}
	}
	t.submitSupportInput(message, actor, state, ids)
}

func isSupportAttachment(message *telego.Message) bool {
	return len(message.Photo) > 0 || message.Document != nil
}

func (t *Tgbot) bufferSupportAlbum(message *telego.Message, actor chatUser, state string) {
	var previousGroup string
	supportAlbums.Lock()
	album := supportAlbums.items[actor]
	if album != nil && album.groupID != message.MediaGroupID {
		previousGroup = album.groupID
	}
	if previousGroup != "" {
		supportAlbums.Unlock()
		t.finalizeSupportAlbum(actor, previousGroup)
		return
	}
	if album == nil {
		first := *message
		album = &supportAlbum{groupID: message.MediaGroupID, message: first, state: state}
		album.timer = time.AfterFunc(supportAlbumDebounce, func() {
			t.finalizeSupportAlbum(actor, message.MediaGroupID)
		})
		supportAlbums.items[actor] = album
	}
	album.messageIDs = append(album.messageIDs, message.MessageID)
	album.timer.Reset(supportAlbumDebounce)
	supportAlbums.Unlock()
}

func (t *Tgbot) finalizeSupportAlbum(actor chatUser, groupID string) {
	supportAlbums.Lock()
	album := supportAlbums.items[actor]
	if album == nil || album.groupID != groupID {
		supportAlbums.Unlock()
		return
	}
	if album.timer != nil {
		album.timer.Stop()
	}
	delete(supportAlbums.items, actor)
	message, state := album.message, album.state
	ids := slices.Clone(album.messageIDs)
	supportAlbums.Unlock()

	current, ok := userStateMgr.get(actor)
	if !ok || current != state {
		return
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	t.submitSupportInput(&message, actor, state, ids)
}

func (t *Tgbot) submitSupportInput(message *telego.Message, actor chatUser, state string, messageIDs []int) {
	userStateMgr.clear(actor)
	switch state {
	case "awaiting_support_message":
		t.clientWindowForCallback(actor, -1).cancel()
		name := "Пользователь"
		if message.From != nil {
			name = strings.TrimSpace(strings.TrimSpace(message.From.FirstName) + " " + strings.TrimSpace(message.From.LastName))
			if message.From.Username != "" {
				name += " (@" + message.From.Username + ")"
			}
		}
		t.synvpnFlow().SendSupportRequest(actor.userID, name, message.Text, messageIDs...)
	default:
		if userIDText, ok := strings.CutPrefix(state, "awaiting_support_reply "); ok {
			userID, err := strconv.ParseInt(userIDText, 10, 64)
			if err != nil || userID <= 0 {
				t.SendMsgToTgbot(message.Chat.ID, "Не удалось определить пользователя для ответа.")
				return
			}
			t.synvpnFlow().SendSupportReply(message.Chat.ID, userID, message.Text, messageIDs...)
		}
	}
}
