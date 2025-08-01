package middleware

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	INTERNAL_AUDIENCE = "internal-service"
)

type InternalJWTClaim struct {
	jwt.RegisteredClaims
}

func AuthMiddleware(logger *logging.LoggerWrapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := c.GetHeader("Authorization")
		if authToken == "" {
			logger.Debug("authorization header is required")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authToken, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debug("authorization header format must be Bearer {token}")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// We assume that the token is valid (it has been verified by the API Gateway)
		tokenStr := parts[1]
		tokClaim, err := parseToken(tokenStr)
		if err != nil {
			debugMSg := fmt.Sprintf("failed to parse token: %v", err)
			logger.Debug(debugMSg)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("token", tokClaim)
		c.Next()
	}
}

func GetTokenClaim(c *gin.Context) (*InternalJWTClaim, error) {
	token, exists := c.Get("token")
	if !exists {
		return nil, fmt.Errorf("token not found in context")
	}

	tokClaim, ok := token.(*InternalJWTClaim)
	if !ok {
		return nil, fmt.Errorf("invalid token type in context")
	}

	return tokClaim, nil
}

func parseToken(tokenString string) (*InternalJWTClaim, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, _, err := parser.ParseUnverified(tokenString, &InternalJWTClaim{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*InternalJWTClaim)
	if !ok {
		return nil, fmt.Errorf("invalid claims (%#v)", token.Claims)
	}

	aud, err := claims.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("failed to get audience: %w", err)
	}
	if len(aud) == 0 {
		return nil, fmt.Errorf("audience is required")
	}

	if aud[0] != INTERNAL_AUDIENCE {
		return nil, fmt.Errorf("invalid audience: %s", aud[0])
	}

	return claims, nil
}

// format: user:id
func GetUserId(c InternalJWTClaim) (int64, error) {
	subject, err := c.GetSubject()
	if err != nil {
		return 0, fmt.Errorf("failed to get subject: %w", err)
	}

	parts := strings.Split(subject, ":")
	if len(parts) != 2 || parts[0] != "user" {
		return 0, fmt.Errorf("invalid subject format: %s", subject)
	}

	userId, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %v", err)
	}

	return userId, nil
}
