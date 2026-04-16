package server

import (
	"context"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"webpage-analyzer/config"
	"webpage-analyzer/internal/handler"
	"webpage-analyzer/internal/middleware"
	"webpage-analyzer/internal/steps"
	"webpage-analyzer/internal/store"
	"webpage-analyzer/web"
)

func New(cfg *config.Config) *http.Server {
	redisStore := store.NewRedisStore(cfg.RedisAddr, cfg.CacheTTL)

	// Verify Redis connectivity at startup
	if err := redisStore.Ping(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Redis not reachable at startup — will retry on first request")
	}

	allSteps := steps.All(cfg)

	analyzeHandler := handler.NewAnalyzeHandler(redisStore, allSteps)
	resultHandler := handler.NewResultHandler(redisStore)

	rateLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(zerolog())
	router.Use(rateLimiter.Middleware())

	// Load embedded templates
	tmpl := template.Must(template.New("").ParseFS(web.Templates, "templates/*.html"))
	router.SetHTMLTemplate(tmpl)

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	router.POST("/analyze", analyzeHandler.Handle)
	router.GET("/result", resultHandler.Handle)

	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
}

// zerolog returns a Gin middleware that logs requests using zerolog.
func zerolog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Str("ip", c.ClientIP()).
			Msg("request")
	}
}
