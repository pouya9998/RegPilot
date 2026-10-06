package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/middleware"
	"github.com/regpilot/regpilot/internal/services"
	"github.com/regpilot/regpilot/internal/utils"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Username and password are required")
		return
	}
	result, err := h.auth.Login(req.Username, req.Password)
	if err != nil {
		utils.JSONUnauthorized(c, "Invalid credentials")
		return
	}
	utils.JSONOK(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.auth.Logout(middleware.GetUserID(c), middleware.GetUsername(c))
	utils.JSONOK(c, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	utils.JSONOK(c, gin.H{
		"id":       middleware.GetUserID(c),
		"username": middleware.GetUsername(c),
		"role":     middleware.GetRole(c),
	})
}
