package middleware

import (
	"github.com/gin-gonic/gin"
	pkgjwt "github.com/medisave/app/pkg/jwt"
)

// ClaimsFromContext extracts the JWT claims injected by Auth middleware.
// Always call after Auth middleware — panics if used on an unprotected route.
func ClaimsFromContext(c *gin.Context) *pkgjwt.Claims {
	val, _ := c.Get(UserContextKey)
	claims, _ := val.(*pkgjwt.Claims)
	return claims
}
