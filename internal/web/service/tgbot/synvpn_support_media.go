package tgbot

import (
	"context"
	"slices"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func copySynVPNSupportMessages(fromChatID, toChatID int64, messageIDs []int) error {
	if len(messageIDs) == 0 {
		return nil
	}
	ids := slices.Clone(messageIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	from := tu.ID(fromChatID)
	to := tu.ID(toChatID)
	return callTelegramAPI(func(ctx context.Context) error {
		var err error
		if len(ids) == 1 {
			_, err = bot.CopyMessage(ctx, &telego.CopyMessageParams{
				ChatID: to, FromChatID: from, MessageID: ids[0],
			})
		} else {
			_, err = bot.CopyMessages(ctx, &telego.CopyMessagesParams{
				ChatID: to, FromChatID: from, MessageIDs: ids,
			})
		}
		return err
	})
}
