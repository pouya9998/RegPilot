package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSONData(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{"data": data})
}

func JSONOK(c *gin.Context, data interface{}) {
	JSONData(c, http.StatusOK, data)
}

func JSONCreated(c *gin.Context, data interface{}) {
	JSONData(c, http.StatusCreated, data)
}

func JSONError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": APIError{Code: code, Message: message}})
}

func JSONBadRequest(c *gin.Context, code, message string) {
	JSONError(c, http.StatusBadRequest, code, message)
}

func JSONUnauthorized(c *gin.Context, message string) {
	JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func JSONForbidden(c *gin.Context, message string) {
	JSONError(c, http.StatusForbidden, "FORBIDDEN", message)
}

func JSONNotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, "NOT_FOUND", message)
}

func JSONInternal(c *gin.Context, message string) {
	JSONError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
