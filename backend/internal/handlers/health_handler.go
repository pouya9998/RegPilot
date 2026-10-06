package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/utils"
)

func Health(c *gin.Context) {
	utils.JSONOK(c, gin.H{"status": "ok"})
}
