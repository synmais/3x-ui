package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func (f *Flow) StartOwnRenewal(chatID int64, user telego.User) {
	traffics, err := f.InboundService.GetClientTrafficTgBot(user.ID)
	if err != nil || len(traffics) == 0 {
		f.ShowRegistrationPrompt(chatID)
		return
	}
	if len(traffics) == 1 {
		f.StartRenewal(chatID, user, traffics[0].Email)
		return
	}
	buttons := make([]telego.InlineKeyboardButton, 0, len(traffics))
	for _, traffic := range traffics {
		buttons = append(buttons, telegoutil.InlineKeyboardButton(traffic.Email).WithCallbackData("client_renew "+traffic.Email))
	}
	f.SendMessage(chatID, "Выберите подписку для продления:", telegoutil.InlineKeyboardGrid(telegoutil.InlineKeyboardCols(1, buttons...)))
}

func (f *Flow) StartRenewal(chatID int64, user telego.User, email string) {
	traffics, err := f.InboundService.GetClientTrafficTgBot(user.ID)
	if err != nil {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	owned := false
	for _, traffic := range traffics {
		if traffic.Email == email {
			owned = true
			break
		}
	}
	if !owned {
		f.SendMessage(chatID, f.Translate("tgbot.answers.errorOperation"))
		return
	}
	f.StartPurchase(chatID, user, synvpn.PurchaseRenew, email)
}
