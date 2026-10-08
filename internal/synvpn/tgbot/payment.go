package tgbot

import (
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/synvpn"
)

func (f *Flow) ProcessYooMoneyPayment(payment *model.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}
	if payment.Status == model.PaymentPaid {
		return nil
	}
	if payment.Status != model.PaymentProcessing {
		return fmt.Errorf(
			"payment %s has unexpected status: %s",
			payment.ID,
			payment.Status,
		)
	}

	if err := f.FulfillPayment(payment); err != nil {
		return err
	}

	billing := synvpn.BillingService{}
	if _, err := billing.CompletePayment(payment.ID); err != nil {
		return err
	}

	message := "🎉 <b>Оплата получена!</b>\n\n" +
		"📱 <b>Рекомендуемые приложения:</b>\n" +
		"<a href=\"https://happ.info/\">Happ</a>\n" +
		"(iOS / Android / Windows / Linux / macOS)\n\n" +
		"<a href=\"https://incy.app/\">INCY</a>\n" +
		"(iOS / Android / Windows / Linux / macOS)\n\n" +
		"<a href=\"https://clashmi.app/download\">Clash Mi</a>\n" +
		"(iOS / Android / ATV / Windows / Linux / macOS)\n\n" +
		"<a href=\"https://apps.apple.com/ru/app/shadowrocket/id932747118\">Shadowrocket</a>\n" +
		"(iOS / macOS / Apple TV).\n\n" +
		"🔄 Для Happ, INCY и Clash Mi правила маршрутизации применяются автоматически. " +
		"⚙️ Для Shadowrocket правила маршрутизации необходимо настроить вручную.\n\n" +
		"⚙️ Для Clash Mi при добавлении подписки не забудьте включить переключатель <b>X-HWID</b>.\n\n" +
		"🔗 <b>Ниже — ссылка на вашу подписку.</b>"

	f.SendMessageNoPreview(payment.TgID, message)
	f.SendSubscriptionLinks(payment.TgID, payment.ClientEmail)

	return nil
}
