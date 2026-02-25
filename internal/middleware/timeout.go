package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			// If handler is still running, we return a timeout.
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{"error": "request timeout"})
			} else {
				c.Abort()
			}
			return
		}
	}
}