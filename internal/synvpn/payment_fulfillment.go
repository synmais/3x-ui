package synvpn

import (
	"strings"
	"github.com/google/uuid"
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
			if containsInt(tariff.InboundIDs, inboundID) {
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
		Comment:         payment.Comment,
		Reset:           0,
		Password:        strings.ReplaceAll(uuid.NewString(), "-", ""),
		Auth:            strings.ReplaceAll(uuid.NewString(), "-", ""),
		TgID:            payment.TgID,
		TrafficReset:    "monthly",
		TrafficResetDay: DefaultTrafficResetDay,
	}

	needRestart, err := s.ClientService.Create(
		&s.InboundService,
		&service.ClientCreatePayload{
			Client:     client,
			InboundIds: tariff.InboundIDs,
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


func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want { return true }
	}
	return false
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 { continue }
		if _, ok := seen[value]; ok { continue }
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
