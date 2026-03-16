package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                 string
	RedisAddr            string
	CacheTTL             time.Duration
	RateLimitRPS         int
	LinkCheckConcurrency int
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		CacheTTL:             time.Duration(getEnvInt("CACHE_TTL_MINUTES", 10)) * time.Minute,
		RateLimitRPS:         getEnvInt("RATE_LIMIT_RPS", 10),
		LinkCheckConcurrency: getEnvInt("LINK_CHECK_CONCURRENCY", 20),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
