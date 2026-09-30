package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"guardrail/internal/limiter"
)

// RateLimitMiddleware is the bouncer standing at the door
func RateLimitMiddleware(rl *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.Allow(clientIP) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please slow down.",
			})
			return
		}

		c.Next()
	}
}

func main() {
	router := gin.Default()

	// 2 cookies per second, max burst of 5 cookies
	rl := limiter.NewRateLimiter(2.0, 5.0)

	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "public and unrestricted"})
	})

	api := router.Group("/api")
	api.Use(RateLimitMiddleware(rl))
	{
		api.GET("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"data":    "Access granted to protected resource",
				"user_ip": c.ClientIP(),
			})
		})
	}

	log.Println("Guardrail running on :8082")
	if err := router.Run(":8082"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}