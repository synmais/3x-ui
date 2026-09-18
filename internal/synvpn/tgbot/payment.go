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
		"Clash Mi, INCY, Happ, Shadowrocket.\n\n" +
		"🔄 Для Clash Mi, INCY и Happ правила маршрутизации применяются автоматически. " +
		"В Shadowrocket правила маршрутизации необходимо настроить вручную.\n\n" +
		"⚙️ <b>Clash Mi</b> — при добавлении подписки не забудьте включить переключатель <b>X-HWID</b>.\n\n" +
		"⚙️ <b>INCY и Happ</b> — может потребоваться включить режим <b>MUX (мультиплексирование)</b> " +
		"в настройках приложения.\n\n" +
		"🔗 <b>Ниже — ссылка на вашу подписку.</b>"

	f.SendMessage(payment.TgID, message)
	f.SendSubscriptionLinks(payment.TgID, payment.ClientEmail)

	return nil
}
