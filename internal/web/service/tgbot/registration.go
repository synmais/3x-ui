package tgbot

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type registrationState struct {
	TgID      int64
	Comment   string
	TariffID  string
	UpdatedAt time.Time
}

type registrationStore struct {
	mu    sync.Mutex
	items map[int64]registrationState
}

const registrationTTL = time.Hour

var registrationMgr = &registrationStore{
	items: make(map[int64]registrationState),
}

func (s *registrationStore) set(chatID int64, state registrationState) {
	s.mu.Lock()
	s.items[chatID] = state
	s.mu.Unlock()
}

func (s *registrationStore) get(chatID int64) (registrationState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.items[chatID]
	if !ok {
		return registrationState{}, false
	}

	if time.Since(state.UpdatedAt) > registrationTTL {
		delete(s.items, chatID)
		return registrationState{}, false
	}

	return state, true
}

func (s *registrationStore) clear(chatID int64) {
	s.mu.Lock()
	delete(s.items, chatID)
	s.mu.Unlock()
}

func (t *Tgbot) startRegistration(chatID int64, user telego.User) {
	registrationMgr.set(chatID, registrationState{
		TgID:      user.ID,
		Comment:   telegramUserComment(user),
		UpdatedAt: time.Now(),
	})

	inlineKeyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("30 ГБ · 1 устройство · 50 ₽/мес").
				WithCallbackData("register_tariff_30gb_1"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("50 ГБ · 3 устройства · 100 ₽/мес ⭐").
				WithCallbackData("register_tariff_50gb_3"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("100 ГБ · 5 устройств · 200 ₽/мес").
				WithCallbackData("register_tariff_100gb_5"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("300 ГБ · 10 устройств · 300 ₽/мес").
				WithCallbackData("register_tariff_300gb_10"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("❌ Отмена").
				WithCallbackData("register_cancel"),
		),
	)

	t.SendMsgToTgbot(
		chatID,
		fmt.Sprintf("👇 <i>%s</i>, выберите тариф:", html.EscapeString(user.FirstName)),
		inlineKeyboard,
	)
}

func telegramUserComment(user telego.User) string {
	name := strings.TrimSpace(
		strings.TrimSpace(user.FirstName) + " " +
			strings.TrimSpace(user.LastName),
	)

	if user.Username != "" {
		if name != "" {
			return fmt.Sprintf("%s (@%s)", name, user.Username)
		}
		return "@" + user.Username
	}

	if name != "" {
		return name
	}

	return fmt.Sprintf("Telegram user %d", user.ID)
}

func (t *Tgbot) registrationTariff(chatID int64, tgUserID int64, tariffID string) {
	state, ok := registrationMgr.get(chatID)

	if !ok {
		t.SendMsgToTgbot(chatID, "Регистрация не найдена. Начните её заново.")
		return
	}

	if state.TgID != tgUserID {
		t.SendMsgToTgbot(chatID, "Регистрация принадлежит другому пользователю.")
		return
	}

	for _, tariff := range tariffs {
		if tariff.ID != tariffID {
			continue
		}

		state.TariffID = tariffID
		state.UpdatedAt = time.Now()
		registrationMgr.set(chatID, state)

		t.SendMsgToTgbot(
			chatID,
			fmt.Sprintf(
				"Вы выбрали:\n\n📊 %d ГБ\n📱 %d устройств\n💰 %d ₽/мес.\n\nВыберите срок подписки:",
				tariff.TotalGB,
				tariff.LimitHWID,
				tariff.MonthlyPrice,
			),
		)

		return
	}

	t.SendMsgToTgbot(chatID, "Не удалось определить тариф. Попробуйте ещё раз.")
}
