package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"strings"

	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// handleSynVPNClientCallback keeps SynVPN client flows out of the upstream
// callback switch. The router remains responsible for authentication and
// passes only callbacks allowed for this client.
func (t *Tgbot) handleSynVPNClientCallback(query *telego.CallbackQuery, actor chatUser, view *clientViewRenderer) bool {
	if view == nil {
		return false
	}
	chatID := query.Message.GetChat().ID
	data := query.Data

	switch data {
	case "client_view_back":
		if !view.backToPrevious() {
			t.sendCallbackAnswerTgBot(query.ID, "Это первый экран")
		} else {
			t.sendCallbackAnswerTgBot(query.ID, "")
		}
		return true
	case "client_view_home":
		view.cancel()
		synvpntgbot.ClearPurchase(chatID)
		userStateMgr.clear(actor)
		t.sendResponse(chatID, t.I18nBot("tgbot.commands.pleaseChoose"), false, false)
		t.sendCallbackAnswerTgBot(query.ID, "")
		return true
	case "client_view_cancel":
		view.cancel()
		synvpntgbot.ClearPurchase(chatID)
		userStateMgr.clear(actor)
		if t.levelOf(query.From.ID) == levelStranger {
			t.startRegistration(chatID, query.From, t.clientViewForActor(actor))
			t.sendCallbackAnswerTgBot(query.ID, "Выберите тариф")
		} else {
			t.sendCallbackAnswerTgBot(query.ID, "Отменено")
		}
		return true
	case "client_payment_fee_info":
		t.sendCallbackAnswerTgBot(query.ID, "")
		view.send(chatID, synvpntgbot.PaymentFeeInfoText)
		return true
	}

	if strings.HasPrefix(data, "client_instruction") {
		page, ok := synvpntgbot.InstructionPageForCallback(data)
		if !ok {
			t.sendCallbackAnswerTgBot(query.ID, "Инструкция не найдена")
			return true
		}
		t.sendCallbackAnswerTgBot(query.ID, "")
		view.sendInstruction(chatID, page)
		return true
	}

	switch data {
	case "register_user":
		traffics, err := t.inboundService.GetClientTrafficTgBot(query.From.ID)
		if err != nil {
			view.send(chatID, t.I18nBot("tgbot.answers.errorOperation"))
		} else if len(traffics) > 0 {
			t.getClientUsage(chatID, query.From.ID, traffics[0].Email, view)
		} else {
			t.startRegistration(chatID, query.From, view)
		}
		t.sendCallbackAnswerTgBot(query.ID, "")
		return true
	case "register_cancel", "subscription_cancel":
		synvpntgbot.ClearPurchase(chatID)
		view.cancel()
		t.sendCallbackAnswerTgBot(query.ID, "Отменено")
		return true
	case "register_confirm", "subscription_confirm":
		t.sendCallbackAnswerTgBot(query.ID, t.I18nBot("tgbot.answers.successfulOperation"))
		t.confirmPurchase(chatID, query.From.ID, view)
		return true
	case "client_traffic":
		traffics, err := t.inboundService.GetClientTrafficTgBot(query.From.ID)
		answer := ""
		if err != nil {
			view.send(chatID, t.I18nBot("tgbot.answers.errorOperation"))
		} else if len(traffics) == 0 {
			t.showRegistrationPrompt(chatID, view)
		} else if len(traffics) == 1 {
			answer = t.I18nBot("tgbot.buttons.clientUsage")
			t.getClientUsage(chatID, query.From.ID, traffics[0].Email, view)
		} else {
			answer = t.I18nBot("tgbot.buttons.clientUsage")
			buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
			for _, traffic := range traffics {
				buttons = append(buttons, tu.InlineKeyboardButton(traffic.Email).WithCallbackData(t.encodeQuery("client_traffic "+traffic.Email)))
			}
			cols := 1
			if len(buttons) >= 6 {
				cols = 2
			}
			keyboard := tu.InlineKeyboardGrid(tu.InlineKeyboardCols(cols, buttons...))
			t.sendSynVPNClientViewWithMarkup(chatID, t.I18nBot("tgbot.commands.pleaseChoose"), view, keyboard)
		}
		t.sendCallbackAnswerTgBot(query.ID, answer)
		return true
	case "client_commands":
		view.send(chatID, t.I18nBot("tgbot.commands.helpClientCommands"))
		t.sendCallbackAnswerTgBot(query.ID, "")
		return true
	case "client_devices":
		t.showOwnDevices(chatID, query.From.ID, view)
	case "client_renew":
		t.startOwnRenewal(chatID, query.From, view)
	case "client_add_subscription":
		t.startNewPurchase(chatID, query.From, view)
	case "client_support":
		t.askClientForSupport(actor, view)
	case "client_sub_links", "client_individual_links", "client_qr_links":
		t.showClientLinkChoices(chatID, query.From.ID, data, view)
	default:
		if t.handleTariffCallback(chatID, query.From.ID, data, view) || t.handlePeriodCallback(chatID, query.From.ID, data, view) {
			t.sendCallbackAnswerTgBot(query.ID, "")
			return true
		}
		if email, ok := strings.CutPrefix(data, "client_traffic "); ok {
			t.getClientUsage(chatID, query.From.ID, email, view)
			t.sendCallbackAnswerTgBot(query.ID, "")
			return true
		}
		if t.handleDeviceCallback(chatID, query.From.ID, data, view) {
			t.sendCallbackAnswerTgBot(query.ID, "")
			return true
		}
		action, email, ok := synvpntgbot.SplitClientLinkCallback(data)
		if !ok || !t.clientOwnedByTgUser(query.From.ID, email) {
			t.sendCallbackAnswerTgBot(query.ID, t.I18nBot("tgbot.answers.errorOperation"))
			return true
		}
		switch action {
		case "client_sub_links":
			t.sendClientSubLinks(chatID, email, view)
		case "client_individual_links":
			t.sendClientIndividualLinks(chatID, email, view)
		case "client_qr_links":
			t.sendClientQRLinks(chatID, email, view)
		case "client_renew":
			t.startRenewal(chatID, query.From, email, view)
		}
		t.sendCallbackAnswerTgBot(query.ID, "")
		return true
	}
	t.sendCallbackAnswerTgBot(query.ID, "")
	return true
}

func (t *Tgbot) showClientLinkChoices(chatID, userID int64, action string, view *clientViewRenderer) {
	traffics, err := t.inboundService.GetClientTrafficTgBot(userID)
	if err != nil {
		message := t.I18nBot("tgbot.answers.errorOperation") + "\r\n" + err.Error()
		if action == "client_qr_links" {
			message = t.I18nBot("tgbot.answers.errorOccurred") + "\r\n" + err.Error()
		}
		view.send(chatID, message)
		return
	}
	if len(traffics) == 0 {
		t.showRegistrationPrompt(chatID, view)
		return
	}
	if action == "client_sub_links" && len(traffics) == 1 {
		t.sendClientSubLinks(chatID, traffics[0].Email, view)
		return
	}
	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, tu.InlineKeyboardButton(traffic.Email).WithCallbackData(t.encodeQuery(action+" "+traffic.Email)))
	}
	cols := 1
	if len(buttons) >= 6 {
		cols = 2
	}
	keyboard := tu.InlineKeyboardGrid(tu.InlineKeyboardCols(cols, buttons...))
	t.sendSynVPNClientViewWithMarkup(chatID, t.I18nBot("tgbot.commands.pleaseChoose"), view, keyboard)
}

// showRegistrationPrompt... *some comment*
func (t *Tgbot) showRegistrationPrompt(chatId int64, view ...*clientViewRenderer) {
	inlineKeyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🆕 Зарегистрироваться").
				WithCallbackData("register_user"),
		),
	)

	if len(view) > 0 && view[0] != nil {
		view[0].send(chatId, t.I18nBot("tgbot.answers.userNotRegistered"), inlineKeyboard)
	} else {
		t.SendMsgToTgbot(chatId, t.I18nBot("tgbot.answers.userNotRegistered"), inlineKeyboard)
	}
}

// clientOwnedByTgUser reports whether email belongs to a client bound to this
// Telegram account, the same list the self-service usage command reads.
func (t *Tgbot) clientOwnedByTgUser(tgUserID int64, email string) bool {
	traffics, err := t.inboundService.GetClientTrafficTgBot(tgUserID)
	if err != nil {
		return false
	}
	for _, traffic := range traffics {
		if traffic.Email == email {
			return true
		}
	}
	return false
}

// getClientUsage retrieves and sends client usage information to the chat.
func (t *Tgbot) getClientUsage(chatId int64, tgUserID int64, email string, view ...*clientViewRenderer) {
	send := t.SendMsgToTgbot
	if len(view) > 0 && view[0] != nil {
		send = view[0].send
	}
	traffics, err := t.inboundService.GetClientTrafficTgBot(tgUserID)
	if err != nil {
		logger.Warning(err)
		msg := t.I18nBot("tgbot.wentWrong")
		t.SendMsgToTgbot(chatId, msg)
		return
	}

	if len(traffics) == 0 {
		t.showRegistrationPrompt(chatId, view...)
		return
	}

	output := ""

	if email != "" {
		for _, traffic := range traffics {
			if traffic.Email == email {
				output = t.clientInfoMsg(traffic, true, true, true, true, true, true)
				send(chatId, output)
				return
			}
		}
		send(chatId, t.I18nBot("tgbot.noResult"))
		return
	}
	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, tu.InlineKeyboardButton(traffic.Email).WithCallbackData(t.encodeQuery("client_traffic "+traffic.Email)))
	}
	kb := tu.InlineKeyboardGrid(tu.InlineKeyboardCols(1, buttons...))
	send(chatId, t.I18nBot("tgbot.commands.pleaseChoose"), kb)
}
