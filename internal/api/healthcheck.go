package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthCheckResponseBody struct {
	Status string            `json:"status"`
	Report map[string]string `json:"report"`
}

func HealthCheck(apiParams *HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		report := map[string]string{}
		serverStatus := "UP"
		// 2025-12-07: new bug: if 2 services depend on each other and they both
		// check each other's status here, it can cause a deadlock. To fix this,
		// we will not check the status of individual services for now.
		/*
			for _, s := range apiParams.Services {
				if s.Status() == services.ServiceReady {
					report[s.Name()] = "UP"
				} else {
					apiParams.Logger.Errorfln("Service %s is DOWN", s.Name())
					report[s.Name()] = "DOWN"
					// we don't set the overall server status to DOWN to avoid k8s
					// restarting the pod
					// serverStatus = "DOWN"

				}
			}
		*/

		responseBody := HealthCheckResponseBody{
			Status: serverStatus,
			Report: report,
		}

		apiParams.Logger.Debugfln("Response body: %v", responseBody)
		c.JSON(http.StatusOK, responseBody)
		c.Abort()
	}
}
