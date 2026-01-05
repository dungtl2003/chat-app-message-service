package middleware

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/types"
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
			logger.Error("authorization header is required")
			resp := types.Response[any]{}
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusUnauthorized,
				Message: "Authorization header is required",
				Errors: []types.ErrorItem{{
					Message: "Authorization header is required",
				}},
			}
			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		parts := strings.Split(authToken, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Error("authorization header format must be Bearer {token}")
			resp := types.Response[any]{}
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusUnauthorized,
				Message: "Authorization header format must be Bearer {token}",
				Errors: []types.ErrorItem{{
					Message: "Authorization header format must be Bearer {token}",
				}},
			}
			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		// We assume that the token is valid (it has been verified by the API Gateway)
		tokenStr := parts[1]
		tokClaim, err := parseToken(tokenStr)
		if err != nil {
			errorMSg := fmt.Sprintf("failed to parse token: %v", err)
			logger.Error(errorMSg)
			resp := types.Response[any]{}
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusUnauthorized,
				Message: errorMSg,
				Errors: []types.ErrorItem{{
					Message: errorMSg,
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		c.Set("token_claim", tokClaim)
		c.Set("token", tokenStr)
		c.Next()
	}
}

func GetToken(c *gin.Context) (string, error) {
	token, exists := c.Get("token")
	if !exists {
		return "", fmt.Errorf("token not found in context")
	}

	tokenStr, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("invalid token type in context")
	}

	return tokenStr, nil
}

func GetTokenClaim(c *gin.Context) (*InternalJWTClaim, error) {
	token, exists := c.Get("token_claim")
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
