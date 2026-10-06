package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/services"
	"github.com/regpilot/regpilot/internal/utils"
)

type AuditHandler struct {
	audit *services.AuditService
}

func NewAuditHandler(audit *services.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) List(c *gin.Context) {
	f := repositories.AuditFilter{
		Username: c.Query("user"),
		Action:   c.Query("action"),
		Limit:    queryInt(c, "limit", 50),
		Offset:   queryInt(c, "offset", 0),
	}
	if rid := c.Query("registry_id"); rid != "" {
		if v, err := strconv.ParseUint(rid, 10, 64); err == nil {
			id := uint(v)
			f.RegistryID = &id
		}
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.To = &t
		}
	}
	logs, total, err := h.audit.List(f)
	if err != nil {
		utils.JSONInternal(c, "Failed to list audit logs")
		return
	}
	utils.JSONOK(c, gin.H{"items": logs, "total": total})
}

func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
