package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRateLimitRouter(rps int) *gin.Engine {
	limiter := NewIPRateLimiter(rps)
	router := gin.New()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return router
}

func TestRateLimiterAllows(t *testing.T) {
	router := setupRateLimitRouter(10) // 10 rps, burst=20

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimiterRejects(t *testing.T) {
	router := setupRateLimitRouter(1) // 1 rps, burst=2

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	// Exhaust burst
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterPerIP(t *testing.T) {
	router := setupRateLimitRouter(1) // 1 rps, burst=2

	reqA, _ := http.NewRequest("GET", "/test", nil)
	reqA.RemoteAddr = "1.2.3.4:1234"

	reqB, _ := http.NewRequest("GET", "/test", nil)
	reqB.RemoteAddr = "5.6.7.8:1234"

	// Exhaust burst for IP A
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, reqA)
	}

	// IP B should still be allowed
	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqB)

	if w.Code != http.StatusOK {
		t.Errorf("IP B should not be limited, got status %d", w.Code)
	}
}

func TestGetLimiter(t *testing.T) {
	rl := NewIPRateLimiter(10)

	l1 := rl.getLimiter("1.2.3.4")
	l2 := rl.getLimiter("1.2.3.4")

	if l1 != l2 {
		t.Error("same IP should return same limiter")
	}

	l3 := rl.getLimiter("5.6.7.8")
	if l1 == l3 {
		t.Error("different IP should return different limiter")
	}
}
