package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/middleware"
	"github.com/regpilot/regpilot/internal/registry"
	"github.com/regpilot/regpilot/internal/services"
	"github.com/regpilot/regpilot/internal/utils"
	"gorm.io/gorm"
)

type RegistryHandler struct {
	regs *services.RegistryService
}

func NewRegistryHandler(regs *services.RegistryService) *RegistryHandler {
	return &RegistryHandler{regs: regs}
}

func (h *RegistryHandler) List(c *gin.Context) {
	list, err := h.regs.List()
	if err != nil {
		utils.JSONInternal(c, "Failed to list registries")
		return
	}
	utils.JSONOK(c, list)
}

func (h *RegistryHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	reg, err := h.regs.Get(id)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, reg)
}

func (h *RegistryHandler) Create(c *gin.Context) {
	var in services.CreateRegistryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid registry payload")
		return
	}
	reg, err := h.regs.Create(middleware.GetUserID(c), middleware.GetUsername(c), in)
	if err != nil {
		utils.JSONBadRequest(c, "REGISTRY_ERROR", err.Error())
		return
	}
	utils.JSONCreated(c, reg)
}

func (h *RegistryHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in services.UpdateRegistryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid registry payload")
		return
	}
	reg, err := h.regs.Update(middleware.GetUserID(c), middleware.GetUsername(c), id, in)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, reg)
}

func (h *RegistryHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.regs.Delete(middleware.GetUserID(c), middleware.GetUsername(c), id); err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, gin.H{"ok": true})
}

func (h *RegistryHandler) Test(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.regs.Test(middleware.GetUserID(c), middleware.GetUsername(c), id); err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, gin.H{"ok": true, "message": "Connection successful"})
}

func (h *RegistryHandler) Health(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	hres, err := h.regs.Health(id)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, hres)
}

func (h *RegistryHandler) Storage(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, err := h.regs.Get(id); err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, h.regs.StorageInfo())
}

func (h *RegistryHandler) ListRepositories(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	search := c.Query("search")
	repos, err := h.regs.ListRepositories(id, search)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, repos)
}

func (h *RegistryHandler) ListTags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	repo := c.Query("repository")
	if repo == "" {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "repository query parameter is required")
		return
	}
	tags, err := h.regs.ListTags(id, repo)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, tags)
}

func (h *RegistryHandler) GetManifest(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	repo := c.Query("repository")
	ref := c.Query("reference")
	if repo == "" || ref == "" {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "repository and reference query parameters are required")
		return
	}
	m, err := h.regs.GetManifest(id, repo, ref)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, m)
}

func (h *RegistryHandler) DeleteImage(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Repository string `json:"repository" binding:"required"`
		Tag        string `json:"tag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "repository and tag are required")
		return
	}
	if err := h.regs.DeleteImage(middleware.GetUserID(c), middleware.GetUsername(c), id, body.Repository, body.Tag); err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, gin.H{"ok": true})
}

func (h *RegistryHandler) GC(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		DryRun bool `json:"dry_run"`
	}
	_ = c.ShouldBindJSON(&body)
	result, err := h.regs.RunGC(middleware.GetUserID(c), middleware.GetUsername(c), id, body.DryRun)
	if err != nil && result == nil {
		utils.JSONInternal(c, "Garbage collection failed")
		return
	}
	utils.JSONOK(c, result)
}

func (h *RegistryHandler) PreviewCleanup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Repository      string `json:"repository" binding:"required"`
		KeepLatest      int    `json:"keep_latest"`
		DeleteOlderDays int    `json:"delete_older_days"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "repository is required")
		return
	}
	preview, err := h.regs.PreviewCleanup(
		middleware.GetUserID(c), middleware.GetUsername(c),
		id, body.Repository, body.KeepLatest, body.DeleteOlderDays,
	)
	if err != nil {
		writeRegistryErr(c, err)
		return
	}
	utils.JSONOK(c, preview)
}

func (h *RegistryHandler) Dashboard(c *gin.Context) {
	stats, err := h.regs.Dashboard()
	if err != nil {
		utils.JSONInternal(c, "Failed to load dashboard")
		return
	}
	utils.JSONOK(c, stats)
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONBadRequest(c, "VALIDATION_ERROR", "Invalid id")
		return 0, false
	}
	return uint(id), true
}

func writeRegistryErr(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.JSONNotFound(c, "Registry not found")
		return
	}
	if errors.Is(err, registry.ErrUnauthorized) {
		utils.JSONError(c, http.StatusUnauthorized, "REGISTRY_UNAUTHORIZED", "Registry authentication failed")
		return
	}
	if errors.Is(err, registry.ErrUnreachable) {
		utils.JSONError(c, http.StatusBadGateway, "REGISTRY_UNREACHABLE", "Unable to connect to registry")
		return
	}
	if errors.Is(err, registry.ErrNotFound) {
		utils.JSONNotFound(c, "Resource not found in registry")
		return
	}
	utils.JSONBadRequest(c, "REGISTRY_ERROR", err.Error())
}
