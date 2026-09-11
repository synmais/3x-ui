package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/billing"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/billing/yoomoney"
)

type paymentProcessor interface {
	ProcessYooMoneyPayment(payment *model.Payment) error
}

type YooMoneyController struct {
	settingService   service.SettingService
	paymentProcessor paymentProcessor
	getSecret        func() (string, error)
}

func NewYooMoneyController(
	g *gin.RouterGroup,
	settingService service.SettingService,
	paymentProcessor paymentProcessor,
) *YooMoneyController {
	a := &YooMoneyController{
		settingService:   settingService,
		paymentProcessor: paymentProcessor,
		getSecret:        settingService.GetYooMoneyNotificationSecret,
	}
	a.initRouter(g)
	return a
}

func (a *YooMoneyController) initRouter(g *gin.RouterGroup) {
	g.POST("/yoomoney/notification", a.notification)
}

func (a *YooMoneyController) notification(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	logger.Infof(
		"YooMoney notification received: type=%q operation_id=%q amount=%q withdraw_amount=%q currency=%q label=%q unaccepted=%q",
		c.Request.PostForm.Get("notification_type"),
		c.Request.PostForm.Get("operation_id"),
		c.Request.PostForm.Get("amount"),
		c.Request.PostForm.Get("withdraw_amount"),
		c.Request.PostForm.Get("currency"),
		c.Request.PostForm.Get("label"),
		c.Request.PostForm.Get("unaccepted"),
	)

	secret, err := a.getSecret()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	notification, err := yoomoney.ParseYooMoneyNotification(
		c.Request.PostForm,
		secret,
	)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	billingService := &billing.BillingService{}
	payment, err := billingService.ConfirmYooMoneyPayment(notification)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if payment.Status == model.PaymentPaid {
		c.Status(http.StatusOK)
		return
	}

	if err := a.paymentProcessor.ProcessYooMoneyPayment(payment); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}
