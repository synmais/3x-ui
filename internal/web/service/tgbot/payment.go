package tgbot

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	billingservice "github.com/mhsanaei/3x-ui/v3/internal/web/service/billing"
)

func (t *Tgbot) CreateClientFromPayment(payment *model.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}

	if payment.ClientEmail == "" {
		return fmt.Errorf("payment %s has empty client email", payment.ID)
	}

	var tariff Tariff
	found := false

	for _, candidate := range tariffs {
		if candidate.ID == payment.TariffID {
			tariff = candidate
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("tariff not found: %s", payment.TariffID)
	}

	// YooMoney может прислать повторное уведомление после того,
	// как клиент уже был создан, но Payment ещё не успел перейти в Paid.
	if _, err := t.clientService.GetRecordByEmail(nil, payment.ClientEmail); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		return nil
	}

	client := model.Client{
		Email:      payment.ClientEmail,
		Enable:     true,
		LimitIP:    0,
		TotalGB:    tariff.TotalGB * 1024 * 1024 * 1024,
		ExpiryTime: time.Now().AddDate(0, payment.Months, 0).UnixMilli(),
		SubID:      t.randomLowerAndNum(16),
		Comment:    payment.Comment,
		Reset:      0,
		TgID:       payment.TgID,
	}

	needRestart, err := t.clientService.Create(
		&t.inboundService,
		&service.ClientCreatePayload{
			Client:     client,
			InboundIds: []int{tariff.InboundID},
			LimitHwid:  tariff.LimitHWID,
		},
	)
	if err != nil {
		return err
	}

	if needRestart {
		t.xrayService.SetToNeedRestart()
	}

	return nil
}

func (t *Tgbot) ProcessYooMoneyPayment(payment *model.Payment) error {
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

	if err := t.CreateClientFromPayment(payment); err != nil {
		return err
	}

	billing := billingservice.BillingService{}
	if _, err := billing.CompletePayment(payment.ID); err != nil {
		return err
	}

	t.SendMsgToTgbot(
		payment.TgID,
		"✅ <b>Оплата получена!</b>\n\n"+
			"Подписка создана. Сейчас подготовим ссылку на подключение.",
	)

	t.sendClientSubLinks(payment.TgID, payment.ClientEmail)

	return nil
}
