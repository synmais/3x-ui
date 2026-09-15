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

	clientSubID := t.randomLowerAndNum(16)

	record, err := t.clientService.GetRecordByEmail(nil, payment.ClientEmail)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		inboundIDs, err := t.clientService.GetInboundIdsForRecord(record.Id)
		if err != nil {
			return err
		}

		for _, inboundID := range inboundIDs {
			if inboundID == tariff.InboundID {
				now := time.Now()
				newExpiry := calculateSubscriptionExpiry(record.ExpiryTime, payment.Months, now)
				if current := tariffForRecord(record.TotalGB, record.LimitHwid); current != nil && current.ID != tariff.ID {
					newExpiry = now.AddDate(0, payment.Months, convertedTariffDays(record.ExpiryTime, *current, tariff, now)).UnixMilli()
				}
				needRestart, err := t.clientService.ResetClientExpiryTimeByEmail(&t.inboundService, payment.ClientEmail, newExpiry)
				if err != nil {
					return err
				}
				if needRestart {
					t.xrayService.SetToNeedRestart()
				}
				needRestart, err = t.clientService.ResetClientTrafficLimitByEmail(&t.inboundService, payment.ClientEmail, int(tariff.TotalGB))
				if err != nil {
					return err
				}
				if needRestart {
					t.xrayService.SetToNeedRestart()
				}
				if err := t.clientService.SetClientLimitHwidByEmail(payment.ClientEmail, tariff.LimitHWID); err != nil {
					return err
				}
				return nil
			}
		}

		if record.SubID == "" {
			return fmt.Errorf(
				"client %s has empty subId",
				payment.ClientEmail,
			)
		}

		clientSubID = record.SubID
	}

	client := model.Client{
		Email:      payment.ClientEmail,
		Enable:     true,
		LimitIP:    0,
		TotalGB:    tariff.TotalGB * 1024 * 1024 * 1024,
		ExpiryTime: time.Now().AddDate(0, payment.Months, 0).UnixMilli(),
		SubID:      clientSubID,
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

	_, renewal := t.clientService.GetRecordByEmail(nil, payment.ClientEmail)
	if err := t.CreateClientFromPayment(payment); err != nil {
		return err
	}

	billing := billingservice.BillingService{}
	if _, err := billing.CompletePayment(payment.ID); err != nil {
		return err
	}

	message := "✅ <b>Оплата получена!</b>\n\nПодписка создана. Сейчас подготовим ссылку на подключение."
	if renewal == nil {
		message = "✅ <b>Оплата получена!</b>\n\nПодписка продлена."
	}
	t.SendMsgToTgbot(payment.TgID, message)

	t.sendClientSubLinks(payment.TgID, payment.ClientEmail)

	return nil
}
