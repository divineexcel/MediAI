package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/response"
)

const UserContextKey = "user_claims"

func Auth(jwtManager *pkgjwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Unauthorized(c, "authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(UserContextKey, claims)
		c.Next()
	}
}
