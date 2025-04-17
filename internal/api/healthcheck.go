package api

import (
	"dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(appCtx *context.AppContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := appCtx.Logger
		if appCtx.IdGeneratorService.GetStatus() != services.RUNNING {
			logger.Debugfln("%s is not running", appCtx.IdGeneratorService.GetName())
			c.JSON(http.StatusOK, "DOWN")
			c.Abort()
			return
		}

		logger.Debugfln("all services are running")
		c.JSON(http.StatusOK, "UP")
		c.Abort()
	}
}
