package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, v any) {
	c.JSON(http.StatusOK, v)
}

func Created(c *gin.Context, v any) {
	c.JSON(http.StatusCreated, v)
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func BadRequest(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, msg)
}

func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, msg)
}

func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, msg)
}

func Internal(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, msg)
}