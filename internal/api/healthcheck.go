package api

import (
	"dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/services"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthCheckResponseBody struct {
	Status string            `json:"status"`
	Report map[string]string `json:"report"`
}

func (h HealthCheckResponseBody) String() string {
	b, err := json.Marshal(h)
	if err != nil {
		return ""
	}
	return string(b)
}

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

		responseBody := HealthCheckResponseBody{
			Status: serverStatus,
			Report: report,
		}

		appCtx.Logger.Debugfln("Response body: %s", responseBody)
		c.JSON(http.StatusOK, responseBody)
		c.Abort()
	}
}
