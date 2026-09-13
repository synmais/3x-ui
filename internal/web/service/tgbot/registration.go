package tgbot

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	billingservice "github.com/mhsanaei/3x-ui/v3/internal/web/service/billing"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type purchaseKind string

const (
	purchaseCreate purchaseKind = "create"
	purchaseRenew  purchaseKind = "renew"
)

type registrationState struct {
	TgID, UpdatedAt                int64
	Comment, ClientEmail, TariffID string
	Kind                           purchaseKind
	Months                         int
}

type registrationStore struct {
	mu    sync.Mutex
	items map[int64]registrationState
}

const registrationTTL = time.Hour

var registrationMgr = &registrationStore{items: make(map[int64]registrationState)}

func (s *registrationStore) set(chatID int64, state registrationState) {
	s.mu.Lock()
	s.items[chatID] = state
	s.mu.Unlock()
}
func (s *registrationStore) get(chatID int64) (registrationState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.items[chatID]
	if !ok || time.Since(time.UnixMilli(state.UpdatedAt)) > registrationTTL {
		delete(s.items, chatID)
		return registrationState{}, false
	}
	return state, true
}
func (s *registrationStore) clear(chatID int64) { s.mu.Lock(); delete(s.items, chatID); s.mu.Unlock() }

// startRegistration remains the first-time entry point. Creation and renewal
// share the tariff, period, summary, and payment flow below.
func (t *Tgbot) startRegistration(chatID int64, user telego.User) {
	t.startPurchase(chatID, user, purchaseCreate, "")
}

func (t *Tgbot) startPurchase(chatID int64, user telego.User, kind purchaseKind, email string) {
	registrationMgr.set(chatID, registrationState{TgID: user.ID, Comment: telegramUserComment(user), ClientEmail: email, Kind: kind, UpdatedAt: time.Now().UnixMilli()})
	buttons := make([]telego.InlineKeyboardButton, 0, len(tariffs)+1)
	for _, tariff := range tariffs {
		buttons = append(buttons, tu.InlineKeyboardButton(tariffLabel(tariff)).WithCallbackData("subscription_tariff_"+tariff.ID))
	}
	buttons = append(buttons, tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("subscription_cancel"))
	prompt := "Выберите тариф:"
	if kind == purchaseRenew {
		prompt = "Выберите тариф для продления:"
	}
	if name := html.EscapeString(user.FirstName); name != "" {
		prompt = fmt.Sprintf("👇 <i>%s</i>, %s", name, prompt)
	}
	t.SendMsgToTgbot(chatID, prompt, tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...)))
}

func telegramUserComment(user telego.User) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
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

func (t *Tgbot) purchaseTariff(chatID, tgUserID int64, tariffID string) {
	state, ok := t.purchaseState(chatID, tgUserID)
	if !ok {
		return
	}
	tariff := findTariff(tariffID)
	if tariff == nil {
		t.SendMsgToTgbot(chatID, "Не удалось определить тариф. Попробуйте ещё раз.")
		return
	}
	state.TariffID, state.UpdatedAt = tariff.ID, time.Now().UnixMilli()
	registrationMgr.set(chatID, state)
	buttons := make([]telego.InlineKeyboardButton, 0, len(tariffPeriods)+1)
	for _, period := range tariffPeriods {
		buttons = append(buttons, tu.InlineKeyboardButton(periodLabel(*tariff, period)).WithCallbackData(fmt.Sprintf("subscription_period_%d", period.Months)))
	}
	buttons = append(buttons, tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("subscription_cancel"))
	t.SendMsgToTgbot(chatID, fmt.Sprintf("Вы выбрали:\n\n%s\n\nВыберите срок подписки:", tariffSummary(*tariff)), tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...)))
}

func (t *Tgbot) purchasePeriod(chatID, tgUserID int64, months int) {
	state, ok := t.purchaseState(chatID, tgUserID)
	if !ok {
		return
	}
	tariff, period := findTariff(state.TariffID), findPeriod(months)
	if tariff == nil || period == nil {
		t.SendMsgToTgbot(chatID, "Некорректный срок подписки.")
		return
	}
	state.Months, state.UpdatedAt = months, time.Now().UnixMilli()
	registrationMgr.set(chatID, state)
	price, action := tariff.Price(*period), "Подтвердить регистрацию?"
	if state.Kind == purchaseRenew {
		action = "Подтвердить продление?"
	}
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(tu.InlineKeyboardButton("✅ Подтвердить").WithCallbackData("subscription_confirm")), tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("subscription_cancel")))
	t.SendMsgToTgbot(chatID, fmt.Sprintf("📋 <b>Ваша подписка</b>\n\n%s\n📅 %d мес.\n💰 <b>%d ₽</b> (%d ₽/мес.)\n\n%s", tariffSummary(*tariff), period.Months, price, price/int64(period.Months), action), keyboard)
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64) {
	state, ok := t.purchaseState(chatID, tgUserID)
	if !ok || state.Months <= 0 {
		t.SendMsgToTgbot(chatID, "Данные подписки заполнены не полностью. Начните заново.")
		registrationMgr.clear(chatID)
		return
	}
	tariff, period := findTariff(state.TariffID), findPeriod(state.Months)
	if tariff == nil || period == nil {
		t.SendMsgToTgbot(chatID, "Выбранный тариф или срок не найден.")
		return
	}
	wallet, err := t.settingService.GetYooMoneyWallet()
	if err != nil || wallet == "" {
		t.SendMsgToTgbot(chatID, "❌ Оплата сейчас недоступна. Попробуйте позже.")
		return
	}
	clientEmail := state.ClientEmail
	if clientEmail == "" {
		clientEmail = t.randomLowerAndNum(8)
	}
	price := tariff.Price(*period)
	billing := billingservice.BillingService{}
	payment, err := billing.CreatePayment(state.TgID, clientEmail, state.Comment, tariff.ID, period.Months, price*100, "yoomoney", time.Now().Add(30*time.Minute))
	if err != nil {
		t.SendMsgToTgbot(chatID, fmt.Sprintf("❌ Не удалось создать платёж: %v", err))
		return
	}
	paymentURL, err := billingservice.YooMoneyPaymentURL(wallet, payment, "")
	if err != nil {
		t.SendMsgToTgbot(chatID, fmt.Sprintf("❌ Не удалось сформировать ссылку на оплату: %v", err))
		return
	}
	registrationMgr.clear(chatID)
	action, after := "регистрации", "подписка будет создана автоматически."
	if state.Kind == purchaseRenew {
		action, after = "продления", "подписка будет продлена автоматически."
	}
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(tu.InlineKeyboardButton(fmt.Sprintf("💳 Оплатить %d ₽", price)).WithURL(paymentURL)))
	t.SendMsgToTgbot(chatID, fmt.Sprintf("💳 <b>Оплата %s</b>\n\n%s\n📅 %d мес.\n💰 <b>%d ₽</b>\n\nПосле оплаты %s", action, tariffSummary(*tariff), period.Months, price, after), keyboard)
}

func (t *Tgbot) purchaseState(chatID, tgUserID int64) (registrationState, bool) {
	state, ok := registrationMgr.get(chatID)
	if !ok {
		t.SendMsgToTgbot(chatID, "Операция не найдена. Начните заново.")
		return registrationState{}, false
	}
	if state.TgID != tgUserID {
		t.SendMsgToTgbot(chatID, "Операция принадлежит другому пользователю.")
		return registrationState{}, false
	}
	return state, true
}
func findTariff(id string) *Tariff {
	for i := range tariffs {
		if tariffs[i].ID == id {
			return &tariffs[i]
		}
	}
	return nil
}
func findPeriod(months int) *TariffPeriod {
	for i := range tariffPeriods {
		if tariffPeriods[i].Months == months {
			return &tariffPeriods[i]
		}
	}
	return nil
}
func tariffLabel(tariff Tariff) string {
	return fmt.Sprintf("%d ГБ · %d %s · %d ₽/мес", tariff.TotalGB, tariff.LimitHWID, russianDeviceWord(tariff.LimitHWID), tariff.MonthlyPrice)
}
func tariffSummary(tariff Tariff) string {
	return fmt.Sprintf("📊 %d ГБ\n📱 %d %s\n💰 %d ₽/мес.", tariff.TotalGB, tariff.LimitHWID, russianDeviceWord(tariff.LimitHWID), tariff.MonthlyPrice)
}
func periodLabel(tariff Tariff, period TariffPeriod) string {
	price := tariff.Price(period)
	if period.Months == 1 {
		return fmt.Sprintf("1 месяц · %d ₽", price)
	}
	return fmt.Sprintf("%d месяца · %d ₽ (%d ₽/мес · −%d%%)", period.Months, price, price/int64(period.Months), period.Discount)
}
func russianDeviceWord(count int) string {
	last := count % 10
	if last == 1 {
		return "устройство"
	}
	if last >= 2 && last <= 4 {
		return "устройства"
	}
	return "устройств"
}
