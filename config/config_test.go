package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	t.Setenv("PORT", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("CACHE_TTL_MINUTES", "")
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("LINK_CHECK_CONCURRENCY", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "localhost:6379")
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, 10*time.Minute)
	}
	if cfg.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS = %d, want %d", cfg.RateLimitRPS, 10)
	}
	if cfg.LinkCheckConcurrency != 20 {
		t.Errorf("LinkCheckConcurrency = %d, want %d", cfg.LinkCheckConcurrency, 20)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("REDIS_ADDR", "redis:6380")
	t.Setenv("CACHE_TTL_MINUTES", "30")
	t.Setenv("RATE_LIMIT_RPS", "5")
	t.Setenv("LINK_CHECK_CONCURRENCY", "50")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.RedisAddr != "redis:6380" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "redis:6380")
	}
	if cfg.CacheTTL != 30*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, 30*time.Minute)
	}
	if cfg.RateLimitRPS != 5 {
		t.Errorf("RateLimitRPS = %d, want %d", cfg.RateLimitRPS, 5)
	}
	if cfg.LinkCheckConcurrency != 50 {
		t.Errorf("LinkCheckConcurrency = %d, want %d", cfg.LinkCheckConcurrency, 50)
	}
}

func TestLoadInvalidInt(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "not_a_number")
	t.Setenv("PORT", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("CACHE_TTL_MINUTES", "")
	t.Setenv("LINK_CHECK_CONCURRENCY", "")

	cfg := Load()

	// Should fall back to default when env var is not a valid int
	if cfg.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS = %d, want default 10 for invalid env", cfg.RateLimitRPS)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	if got := getEnv("TEST_KEY", "fallback"); got != "value" {
		t.Errorf("getEnv = %q, want %q", got, "value")
	}

	t.Setenv("TEST_KEY", "")
	if got := getEnv("TEST_KEY", "fallback"); got != "fallback" {
		t.Errorf("getEnv = %q, want fallback %q", got, "fallback")
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if got := getEnvInt("TEST_INT", 0); got != 42 {
		t.Errorf("getEnvInt = %d, want %d", got, 42)
	}

	t.Setenv("TEST_INT", "abc")
	if got := getEnvInt("TEST_INT", 99); got != 99 {
		t.Errorf("getEnvInt = %d, want fallback %d", got, 99)
	}

	t.Setenv("TEST_INT", "")
	if got := getEnvInt("TEST_INT", 7); got != 7 {
		t.Errorf("getEnvInt = %d, want fallback %d", got, 7)
	}
}
