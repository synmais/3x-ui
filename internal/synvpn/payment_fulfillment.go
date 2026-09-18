package synvpn

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type PaymentFulfillmentService struct {
	ClientService  service.ClientService
	InboundService service.InboundService
	XrayService    service.XrayService
}

func (s *PaymentFulfillmentService) CreateClientFromPayment(payment *model.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}
	if payment.ClientEmail == "" {
		return fmt.Errorf("payment %s has empty client email", payment.ID)
	}

	tariff := FindTariff(payment.TariffID)
	if tariff == nil {
		return fmt.Errorf("tariff not found: %s", payment.TariffID)
	}

	clientSubID, err := randomLowerAndNum(16)
	if err != nil {
		return err
	}

	record, err := s.ClientService.GetRecordByEmail(nil, payment.ClientEmail)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		inboundIDs, err := s.ClientService.GetInboundIdsForRecord(record.Id)
		if err != nil {
			return err
		}

		for _, inboundID := range inboundIDs {
			if inboundID == tariff.InboundID {
				now := time.Now()
				newExpiry := CalculateSubscriptionExpiry(record.ExpiryTime, payment.Months, now)

				if current := FindTariffForRecord(record.TotalGB, record.LimitHwid); current != nil && current.ID != tariff.ID {
					newExpiry = now.AddDate(
						0,
						payment.Months,
						ConvertedTariffDays(record.ExpiryTime, *current, *tariff, now),
					).UnixMilli()
				}

				needRestart, err := s.ClientService.ResetClientExpiryTimeByEmail(
					&s.InboundService,
					payment.ClientEmail,
					newExpiry,
				)
				if err != nil {
					return err
				}
				if needRestart {
					s.XrayService.SetToNeedRestart()
				}

				needRestart, err = s.ClientService.ResetClientTrafficLimitByEmail(
					&s.InboundService,
					payment.ClientEmail,
					int(tariff.TotalGB),
				)
				if err != nil {
					return err
				}
				if needRestart {
					s.XrayService.SetToNeedRestart()
				}

				if err := s.ClientService.SetClientLimitHwidByEmail(
					payment.ClientEmail,
					tariff.LimitHWID,
				); err != nil {
					return err
				}

				return nil
			}
		}

		if record.SubID == "" {
			return fmt.Errorf("client %s has empty subId", payment.ClientEmail)
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

	needRestart, err := s.ClientService.Create(
		&s.InboundService,
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
		s.XrayService.SetToNeedRestart()
	}

	return nil
}
