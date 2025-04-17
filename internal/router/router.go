package router

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/middleware"
	"fmt"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

type Method string

const (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	PATCH  Method = "PATCH"
	DELETE Method = "DELETE"
)

type Handler struct {
	Method Method
	Path   string
	H      gin.HandlerFunc
}

func isMethodValid(m Method) bool {
	switch m {
	case GET, POST, PUT, DELETE, PATCH:
		return true
	default:
		return false
	}
}

func New(logger *logging.LoggerWrapper, publicHandlers []Handler, privateHandlers []Handler) (*gin.Engine, error) {
	r := gin.New()
	r.Use(sloggin.New(logger.Logger))
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	public := r.Group("")
	private := r.Group("")
	private.Use(middleware.AuthMiddleware(logger))
	err := setHandler(public, publicHandlers...)
	if err != nil {
		return nil, err
	}
	err = setHandler(private, privateHandlers...)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func setHandler(r *gin.RouterGroup, handlers ...Handler) error {
	for _, h := range handlers {
		if !isMethodValid(h.Method) {
			return fmt.Errorf("invalid method: %s", h.Method)
		}

		r.Handle(string(h.Method), h.Path, h.H)
	}

	return nil
}
