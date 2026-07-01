package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/domain/entity"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/response"
)

func RequireRole(roles ...entity.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get(UserContextKey)
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		userClaims, ok := claims.(*pkgjwt.Claims)
		if !ok {
			response.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}

		for _, role := range roles {
			if userClaims.Role == role {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "you do not have permission to access this resource")
		c.Abort()
	}
}

func RequirePatient() gin.HandlerFunc {
	return RequireRole(entity.RolePatient)
}

func RequireDoctor() gin.HandlerFunc {
	return RequireRole(entity.RoleDoctor)
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole(entity.RoleAdmin)
}

func RequirePatientOrDoctor() gin.HandlerFunc {
	return RequireRole(entity.RolePatient, entity.RoleDoctor)
}
