package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/middleware"
	"github.com/regpilot/regpilot/internal/services"
	"github.com/regpilot/regpilot/internal/utils"
	"gorm.io/gorm"
)

type UserHandler struct {
	users *services.UserService
}

func NewUserHandler(users *services.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) List(c *gin.Context) {
	list, err := h.users.List()
	if err != nil {
		utils.JSONInternal(c, "Failed to list users")
		return
	}
	utils.JSONOK(c, list)
}

func (h *UserHandler) Create(c *gin.Context) {
	var in services.CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid user payload")
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	user, err := h.users.Create(middleware.GetUserID(c), middleware.GetUsername(c), in)
	if err != nil {
		utils.JSONBadRequest(c, "USER_ERROR", err.Error())
		return
	}
	utils.JSONCreated(c, user)
}

func (h *UserHandler) SetDisabled(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid user id")
		return
	}
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid payload")
		return
	}
	user, err := h.users.SetDisabled(middleware.GetUserID(c), middleware.GetUsername(c), uint(id), body.Disabled)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONNotFound(c, "User not found")
			return
		}
		utils.JSONBadRequest(c, "USER_ERROR", err.Error())
		return
	}
	utils.JSONOK(c, user)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid user id")
		return
	}
	var body struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Password is required")
		return
	}
	if err := h.users.ResetPassword(middleware.GetUserID(c), middleware.GetUsername(c), uint(id), body.Password); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONNotFound(c, "User not found")
			return
		}
		utils.JSONBadRequest(c, "USER_ERROR", err.Error())
		return
	}
	utils.JSONOK(c, gin.H{"ok": true})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid user id")
		return
	}
	if err := h.users.Delete(middleware.GetUserID(c), middleware.GetUsername(c), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONNotFound(c, "User not found")
			return
		}
		utils.JSONBadRequest(c, "USER_ERROR", err.Error())
		return
	}
	utils.JSONOK(c, gin.H{"ok": true})
}
