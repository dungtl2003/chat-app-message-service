package api

import (
	"dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(appCtx *context.AppContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		report := map[string]string{}
		serverStatus := "UP"
		for _, s := range appCtx.Services {
			if s.Status() == services.READY {
				report[s.Name()] = "UP"
			} else {
				report[s.Name()] = "DOWN"
				serverStatus = "DOWN"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": serverStatus,
			"report": report,
		})

		c.Abort()
	}
}
