package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter enforces a per-IP request rate limit using a token bucket.
type IPRateLimiter struct {
	limiters sync.Map
	rps      rate.Limit
	burst    int
}

func NewIPRateLimiter(rps int) *IPRateLimiter {
	return &IPRateLimiter{
		rps:   rate.Limit(rps),
		burst: rps * 2,
	}
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	if v, ok := l.limiters.Load(ip); ok {
		return v.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(l.rps, l.burst)
	l.limiters.Store(ip, limiter)
	return limiter
}

// Middleware returns a Gin handler that rejects requests exceeding the rate limit.
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.getLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please slow down",
			})
			return
		}
		c.Next()
	}
}
