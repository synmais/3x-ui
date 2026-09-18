package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func (t *Tgbot) startOwnRenewal(chatID int64, user telego.User) {
	traffics, err := t.inboundService.GetClientTrafficTgBot(user.ID)
	if err != nil || len(traffics) == 0 {
		t.showRegistrationPrompt(chatID)
		return
	}
	if len(traffics) == 1 {
		t.startRenewal(chatID, user, traffics[0].Email)
		return
	}
	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, telegoutil.InlineKeyboardButton(traffic.Email).WithCallbackData("client_renew "+traffic.Email))
	}
	t.SendMsgToTgbot(chatID, "Выберите подписку для продления:", telegoutil.InlineKeyboardGrid(telegoutil.InlineKeyboardCols(1, buttons...)))
}

func (t *Tgbot) startRenewal(chatID int64, user telego.User, email string) {
	if !t.ownsClient(user.ID, email) {
		t.sendCallbackError(chatID)
		return
	}
	t.startPurchase(chatID, user, synvpn.PurchaseRenew, email)
}
