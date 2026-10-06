package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/utils"
)

const (
	ContextUserID   = "userID"
	ContextUsername = "username"
	ContextRole     = "role"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.JSONUnauthorized(c, "Missing or invalid authorization header")
			c.Abort()
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ParseToken(token, jwtSecret)
		if err != nil {
			utils.JSONUnauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		roleStr, _ := role.(string)
		if !allowed[roleStr] {
			utils.JSONForbidden(c, "Insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	v, _ := c.Get(ContextUserID)
	id, _ := v.(uint)
	return id
}

func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUsername)
	s, _ := v.(string)
	return s
}

func GetRole(c *gin.Context) string {
	v, _ := c.Get(ContextRole)
	s, _ := v.(string)
	return s
}

func IsAdmin(c *gin.Context) bool {
	return GetRole(c) == models.RoleAdmin
}
