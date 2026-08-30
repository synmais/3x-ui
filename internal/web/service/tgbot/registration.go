package tgbot

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type registrationState struct {
	TgID      int64
	Comment   string
	TariffID  string
	Months    int
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

		inlineKeyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					fmt.Sprintf("1 месяц · %d ₽", tariff.Price(tariffPeriods[0])),
				).WithCallbackData("register_period_1"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					fmt.Sprintf(
						"3 месяца · %d ₽ (%d ₽/мес · −%d%%)",
						tariff.Price(tariffPeriods[1]),
						tariff.Price(tariffPeriods[1])/int64(tariffPeriods[1].Months),
						tariffPeriods[1].Discount,
					),
				).WithCallbackData("register_period_3"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					fmt.Sprintf(
						"6 месяцев · %d ₽ (%d ₽/мес · −%d%%)",
						tariff.Price(tariffPeriods[2]),
						tariff.Price(tariffPeriods[2])/int64(tariffPeriods[2].Months),
						tariffPeriods[2].Discount,
					),
				).WithCallbackData("register_period_6"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					fmt.Sprintf(
						"12 месяцев · %d ₽ (%d ₽/мес · −%d%%)",
						tariff.Price(tariffPeriods[3]),
						tariff.Price(tariffPeriods[3])/int64(tariffPeriods[3].Months),
						tariffPeriods[3].Discount,
					),
				).WithCallbackData("register_period_12"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("❌ Отмена").
					WithCallbackData("register_cancel"),
			),
		)

		t.SendMsgToTgbot(
			chatID,
			fmt.Sprintf(
				"Вы выбрали:\n\n📊 %d ГБ\n📱 %d устройств\n💰 %d ₽/мес.\n\nВыберите срок подписки:",
				tariff.TotalGB,
				tariff.LimitHWID,
				tariff.MonthlyPrice,
			),
			inlineKeyboard,
		)

		return

	}

	t.SendMsgToTgbot(chatID, "Не удалось определить тариф. Попробуйте ещё раз.")
}

func (t *Tgbot) registrationPeriod(chatID int64, tgUserID int64, months int) {
	state, ok := registrationMgr.get(chatID)
	if !ok {
		t.SendMsgToTgbot(chatID, "Регистрация не найдена. Начните её заново.")
		return
	}

	if state.TgID != tgUserID {
		t.SendMsgToTgbot(chatID, "Регистрация принадлежит другому пользователю.")
		return
	}

	var period TariffPeriod
	found := false

	for _, candidate := range tariffPeriods {
		if candidate.Months == months {
			period = candidate
			found = true
			break
		}
	}

	if !found {
		t.SendMsgToTgbot(chatID, "Некорректный срок регистрации.")
		return
	}

	var tariff Tariff
	found = false

	for _, candidate := range tariffs {
		if candidate.ID == state.TariffID {
			tariff = candidate
			found = true
			break
		}
	}

	if !found {
		t.SendMsgToTgbot(chatID, "Выбранный тариф не найден. Начните регистрацию заново.")
		registrationMgr.clear(chatID)
		return
	}

	state.Months = months
	state.UpdatedAt = time.Now()
	registrationMgr.set(chatID, state)

	price := tariff.Price(period)
	monthlyPrice := price / int64(period.Months)

	inlineKeyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✅ Подтвердить").
				WithCallbackData("register_confirm"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("❌ Отмена").
				WithCallbackData("register_cancel"),
		),
	)

	t.SendMsgToTgbot(
		chatID,
		fmt.Sprintf(
			"📋 <b>Ваша подписка</b>\n\n"+
				"📊 %d ГБ\n"+
				"📱 %d устройств\n"+
				"📅 %d мес.\n"+
				"💰 <b>%d ₽</b> (%d ₽/мес.)\n\n"+
				"Подтвердить регистрацию?",
			tariff.TotalGB,
			tariff.LimitHWID,
			period.Months,
			price,
			monthlyPrice,
		),
		inlineKeyboard,
	)
}

func (t *Tgbot) confirmRegistration(chatID int64, tgUserID int64) {
	state, ok := registrationMgr.get(chatID)
	if !ok {
		t.SendMsgToTgbot(chatID, "Регистрация не найдена. Начните её заново.")
		return
	}

	if state.TgID != tgUserID {
		t.SendMsgToTgbot(chatID, "Регистрация принадлежит другому пользователю.")
		return
	}

	if state.TariffID == "" || state.Months <= 0 {
		t.SendMsgToTgbot(chatID, "Данные регистрации заполнены не полностью. Начните регистрацию заново.")
		registrationMgr.clear(chatID)
		return
	}

	var tariff Tariff
	found := false

	for _, candidate := range tariffs {
		if candidate.ID == state.TariffID {
			tariff = candidate
			found = true
			break
		}
	}

	if !found {
		t.SendMsgToTgbot(chatID, "Выбранный тариф не найден. Начните регистрацию заново.")
		registrationMgr.clear(chatID)
		return
	}

	var period TariffPeriod
	found = false

	for _, candidate := range tariffPeriods {
		if candidate.Months == state.Months {
			period = candidate
			found = true
			break
		}
	}

	if !found {
		t.SendMsgToTgbot(chatID, "Выбранный срок подписки не найден. Начните регистрацию заново.")
		registrationMgr.clear(chatID)
		return
	}

	// Генерируем идентификаторы непосредственно перед созданием клиента.
	clientEmail := t.randomLowerAndNum(8)
	clientSubID := t.randomLowerAndNum(16)

	// Срок подписки в миллисекундах, как используется 3x-ui.
	expiryTime := time.Now().AddDate(0, period.Months, 0).UnixMilli()

	client := model.Client{
		Email:      clientEmail,
		Enable:     true,
		LimitIP:    0,
		TotalGB:    tariff.TotalGB * 1024 * 1024 * 1024,
		ExpiryTime: expiryTime,
		SubID:      clientSubID,
		Comment:    state.Comment,
		Reset:      0,
		TgID:       state.TgID,
	}

	_, err := t.clientService.Create(
		&t.inboundService,
		&service.ClientCreatePayload{
			Client:     client,
			InboundIds: []int{tariff.InboundID},
			LimitHwid:  tariff.LimitHWID,
		},
	)
	if err != nil {
		t.SendMsgToTgbot(
			chatID,
			fmt.Sprintf("❌ Не удалось создать подписку: %v", err),
		)
		return
	}

	registrationMgr.clear(chatID)

	t.SendMsgToTgbot(
		chatID,
		fmt.Sprintf(
			"✅ <b>Подписка создана!</b>\n\n"+
				"📧 %s\n"+
				"📊 %d ГБ\n"+
				"📱 %d устройств\n"+
				"📅 %d мес.\n"+
				"💰 %d ₽\n\n"+
				"Сейчас подготовим ссылку на подключение.",
			html.EscapeString(clientEmail),
			tariff.TotalGB,
			tariff.LimitHWID,
			period.Months,
			tariff.Price(period),
		),
	)

	t.sendClientSubLinks(chatID, clientEmail)
	//t.showMainMenu(chatID, false)
}
