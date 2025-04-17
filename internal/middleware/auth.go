package middleware

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/model"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserId   int64
	Username string
	Role     model.UserRole
}

type JWTClaim struct {
	jwt.RegisteredClaims
	Username string `json:"name"`
}

func (c JWTClaim) GetUsername() (string, error) {
	return c.Username, nil
}

func AuthMiddleware(logger *logging.LoggerWrapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If the request is /health, we don't need to check the token
		if c.Request.URL.Path == "/healthcheck" {
			c.Next()
			return
		}

		authToken := c.GetHeader("Authorization")
		if authToken == "" {
			logger.Debugfln("authorization header is required")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		logger.Debugfln("authorization header: %s", authToken)

		parts := strings.Split(authToken, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debugfln("authorization header format must be Bearer {token}")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		// We assume that the token is valid (it has been verified by the API Gateway)
		logger.Debugfln("access token: %s", tokenStr)
		token, err := decodeTokenUnsafe(tokenStr)
		if err != nil {
			logger.Debugfln("decodeTokenUnsafe(): %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*JWTClaim)
		if !ok {
			logger.Debugfln("invalid token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		logger.Debugfln("claims: %#v", claims)

		userClaims, err := mapToUserClaims(*claims)
		if err != nil {
			logger.Debugfln("mapToUserClaims(): %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		logger.Debugfln("set user claims: %#v", userClaims)
		c.Set("user_claims", *userClaims)
		c.Next()
	}
}

func mapToUserClaims(claims JWTClaim) (*UserClaims, error) {
	userIdStr, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}

	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return nil, err
	}

	username, err := claims.GetUsername()
	if err != nil {
		return nil, err
	}

	aud, err := claims.GetAudience()
	if err != nil {
		return nil, err
	}
	if len(aud) == 0 {
		return nil, fmt.Errorf("aud claim is empty")
	}

	roleStr := aud[0]

	if !model.IsRole(roleStr) {
		return nil, fmt.Errorf("role claim is invalid")
	}

	role := model.UserRole(roleStr)

	return &UserClaims{
		UserId:   userId,
		Username: username,
		Role:     role,
	}, nil
}

func GetUserClaims(c *gin.Context) (*UserClaims, error) {
	userClaimsAny, ok := c.Get("user_claims")
	if !ok {
		return nil, fmt.Errorf("user claims not found")
	}
	userClaims, ok := userClaimsAny.(UserClaims)
	if !ok {
		return nil, fmt.Errorf("user claims has wrong type")
	}
	return &userClaims, nil
}

func decodeTokenUnsafe(tokenString string) (*jwt.Token, error) {
	// Decode the token here
	// This is a placeholder implementation
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, _, err := parser.ParseUnverified(tokenString, &JWTClaim{})

	if err != nil {
		return nil, err
	}

	return token, nil
}
