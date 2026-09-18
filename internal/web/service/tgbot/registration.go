package tgbot

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
)

var purchaseMgr = synvpn.NewPurchaseStore()

func (t *Tgbot) startRegistration(chatID int64, user telego.User) {
	t.startPurchase(chatID, user, synvpn.PurchaseCreate, "")
}

func (t *Tgbot) startPurchase(
	chatID int64,
	user telego.User,
	kind synvpn.PurchaseKind,
	email string,
) {
	purchaseMgr.Set(chatID, synvpn.PurchaseState{
		TgID:        user.ID,
		Comment:     telegramUserComment(user),
		ClientEmail: email,
		Kind:        kind,
		UpdatedAt:   time.Now().UnixMilli(),
	})

	catalog := synvpn.Tariffs()
	buttons := make([]telego.InlineKeyboardButton, 0, len(catalog)+1)
	for _, tariff := range catalog {
		buttons = append(buttons, tu.InlineKeyboardButton(tariffLabel(tariff)).WithCallbackData("subscription_tariff_"+tariff.ID))
	}
	buttons = append(buttons, tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("subscription_cancel"))

	prompt := "Выберите тариф:"
	if kind == synvpn.PurchaseRenew {
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
	tariff := synvpn.FindTariff(tariffID)
	if tariff == nil {
		t.SendMsgToTgbot(chatID, "Не удалось определить тариф. Попробуйте ещё раз.")
		return
	}
	state.TariffID, state.UpdatedAt = tariff.ID, time.Now().UnixMilli()
	purchaseMgr.Set(chatID, state)
	periods := synvpn.TariffPeriods()
	buttons := make([]telego.InlineKeyboardButton, 0, len(periods)+1)
	for _, period := range periods {
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
	tariff, period := synvpn.FindTariff(state.TariffID), synvpn.FindPeriod(months)
	if tariff == nil || period == nil {
		t.SendMsgToTgbot(chatID, "Некорректный срок подписки.")
		return
	}
	state.Months, state.UpdatedAt = months, time.Now().UnixMilli()
	state.CarryoverDays = 0
	warning := ""
	if state.Kind == synvpn.PurchaseRenew {
		if record, err := t.clientService.GetRecordByEmail(nil, state.ClientEmail); err == nil {
			if current := synvpn.FindTariffForRecord(record.TotalGB, record.LimitHwid); current != nil && current.ID != tariff.ID {
				state.CarryoverDays = synvpn.ConvertedTariffDays(record.ExpiryTime, *current, *tariff, time.Now())
				warning = fmt.Sprintf("\n\n⚠️ Выбранный тариф не соответствует текущему. Остаток пересчитан: <b>%d %s</b> нового тарифа добавлено к выбранному сроку.", state.CarryoverDays, russianDayWord(state.CarryoverDays))
			}
		}
	}
	purchaseMgr.Set(chatID, state)
	price, action := synvpn.CalculatePrice(*tariff, *period), "Подтвердить регистрацию?"
	if state.Kind == synvpn.PurchaseRenew {
		action = "Подтвердить продление?"
	}
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(tu.InlineKeyboardButton("✅ Подтвердить").WithCallbackData("subscription_confirm")), tu.InlineKeyboardRow(tu.InlineKeyboardButton(t.I18nBot("tgbot.buttons.cancel")).WithCallbackData("subscription_cancel")))
	totalTerm := fmt.Sprintf("%d мес.", period.Months)
	if state.CarryoverDays > 0 {
		totalTerm += fmt.Sprintf(" + %d %s", state.CarryoverDays, russianDayWord(state.CarryoverDays))
	}
	t.SendMsgToTgbot(chatID, fmt.Sprintf("📋 <b>Ваша подписка</b>\n\n%s\n📅 %s\n💰 <b>%d ₽</b> (%d ₽/мес.)%s\n\n%s", tariffSummary(*tariff), totalTerm, price, price/int64(period.Months), warning, action), keyboard)
}

func (t *Tgbot) confirmPurchase(chatID, tgUserID int64) {
	state, ok := t.purchaseState(chatID, tgUserID)
	if !ok || state.Months <= 0 {
		t.SendMsgToTgbot(chatID, "Данные подписки заполнены не полностью. Начните заново.")
		purchaseMgr.Clear(chatID)
		return
	}
	tariff, period := synvpn.FindTariff(state.TariffID), synvpn.FindPeriod(state.Months)
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

	price := synvpn.CalculatePrice(*tariff, *period)
	billing := synvpn.BillingService{}
	paymentURL, err := billing.CreateYooMoneyPayment(
		state.TgID,
		clientEmail,
		state.Comment,
		tariff.ID,
		period.Months,
		price*100,
		time.Now().Add(30*time.Minute),
		wallet,
	)
	if err != nil {
		t.SendMsgToTgbot(chatID, fmt.Sprintf("❌ Не удалось создать платёж: %v", err))
		return
	}
	purchaseMgr.Clear(chatID)
	action, after := "регистрации", "подписка будет создана автоматически."
	if state.Kind == synvpn.PurchaseRenew {
		action, after = "продления", "подписка будет продлена автоматически."
	}

	totalTerm := fmt.Sprintf("%d мес.", period.Months)
	if state.CarryoverDays > 0 {
		totalTerm += fmt.Sprintf(" + %d %s", state.CarryoverDays, russianDayWord(state.CarryoverDays))
	}

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(fmt.Sprintf("💳 Оплатить %d ₽", price)).WithURL(paymentURL),
		),
	)

	t.SendMsgToTgbot(
		chatID,
		fmt.Sprintf(
			"💳 <b>Оплата %s</b>\n\n%s\n📅 %s\n💰 <b>%d ₽</b>\n\nПосле оплаты %s\n\n⚠️ <b>Важно о комиссии</b>\n\nДанная операция может трактоваться банком как перевод по номеру карты.\n\nНапример, Альфа-Банк может взимать комиссию 1,95%% + 49 ₽, тогда как Сбербанк, Т-Банк и МТС Деньги в рамках ежемесячного лимита комиссию не взимают.\n\nУточните размер комиссии за переводы по номеру карты в вашем банке перед оплатой.",
			action,
			tariffSummary(*tariff),
			totalTerm,
			price,
			after,
		),
		keyboard,
	)
}

func (t *Tgbot) purchaseState(chatID, tgUserID int64) (synvpn.PurchaseState, bool) {
	state, ok := purchaseMgr.Get(chatID)
	if !ok {
		t.SendMsgToTgbot(chatID, "Операция не найдена. Начните заново.")
		return synvpn.PurchaseState{}, false
	}
	if state.TgID != tgUserID {
		t.SendMsgToTgbot(chatID, "Операция принадлежит другому пользователю.")
		return synvpn.PurchaseState{}, false
	}
	return state, true
}
func tariffLabel(tariff synvpn.Tariff) string {
	return fmt.Sprintf("%d ГБ · %d %s · %d ₽/мес", tariff.TotalGB, tariff.LimitHWID, russianDeviceWord(tariff.LimitHWID), tariff.MonthlyPrice)
}
func tariffSummary(tariff synvpn.Tariff) string {
	return fmt.Sprintf("📊 %d ГБ\n📱 %d %s\n💰 %d ₽/мес.", tariff.TotalGB, tariff.LimitHWID, russianDeviceWord(tariff.LimitHWID), tariff.MonthlyPrice)
}
func periodLabel(tariff synvpn.Tariff, period synvpn.TariffPeriod) string {
	price := synvpn.CalculatePrice(tariff, period)
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

func russianDayWord(count int) string {
	mod100 := count % 100
	if mod100 >= 11 && mod100 <= 14 {
		return "дней"
	}

	switch count % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}
