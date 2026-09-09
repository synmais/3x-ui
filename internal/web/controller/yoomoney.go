package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/billing"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/billing/yoomoney"
)

type YooMoneyController struct {
	settingService service.SettingService
	getSecret      func() (string, error)
}

func NewYooMoneyController(
	g *gin.RouterGroup,
	settingService service.SettingService,
) *YooMoneyController {
	a := &YooMoneyController{
		settingService: settingService,
		getSecret:      settingService.GetYooMoneyNotificationSecret,
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
	if _, err := billingService.ConfirmYooMoneyPayment(notification); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.Status(http.StatusOK)
}
