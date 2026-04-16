package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"webpage-analyzer/internal/pipeline"
	"webpage-analyzer/internal/store"
	"webpage-analyzer/internal/urlutil"
)

type AnalyzeHandler struct {
	store     store.Store
	steps     []pipeline.Step
	stepNames []string
}

func NewAnalyzeHandler(s store.Store, steps []pipeline.Step) *AnalyzeHandler {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name()
	}
	return &AnalyzeHandler{store: s, steps: steps, stepNames: names}
}

func (h *AnalyzeHandler) Handle(c *gin.Context) {
	rawURL := c.PostForm("url")
	if rawURL == "" {
		c.HTML(http.StatusBadRequest, "result.html", gin.H{
			"Error": "URL is required",
		})
		return
	}

	normalized := urlutil.Normalize(rawURL)
	jobKey := urlutil.JobKey(normalized)
	ctx := c.Request.Context()

	// If a job is already running for this URL, return the polling view
	existing, _ := h.store.GetAll(ctx, jobKey)
	if s := existing["overall_status"]; s == "pending" || s == "processing" {
		c.HTML(http.StatusOK, "result.html", buildViewModel(normalized, existing))
		return
	}

	// Initialise job state in Redis
	if err := h.store.InitJob(ctx, jobKey, h.stepNames); err != nil {
		log.Error().Err(err).Msg("failed to init job in Redis")
		c.HTML(http.StatusInternalServerError, "result.html", gin.H{
			"Error": "service unavailable, please try again",
		})
		return
	}

	// Run the pipeline in the background
	go func() {
		bgCtx := context.Background()
		logger := log.With().Str("job", jobKey).Str("url", normalized).Logger()
		logger.Info().Msg("analysis started")

		state := pipeline.NewState(normalized)

		callback := func(ctx context.Context, stepName string, result pipeline.StepResult) {
			dataStr := ""
			if result.Data != nil {
				if b, err := json.Marshal(result.Data); err == nil {
					dataStr = string(b)
				}
			}
			if err := h.store.SetStep(ctx, jobKey, stepName, result.Status, dataStr, result.Error); err != nil {
				logger.Error().Err(err).Str("step", stepName).Msg("failed to update step in Redis")
			}
		}

		p := pipeline.New(h.steps...).WithCallback(callback)

		if err := p.Run(bgCtx, state); err != nil {
			logger.Error().Err(err).Msg("pipeline failed")
			_ = h.store.SetOverallStatus(bgCtx, jobKey, "failed", err.Error())
			return
		}

		_ = h.store.SetOverallStatus(bgCtx, jobKey, "done", "")
		logger.Info().Msg("analysis complete")
	}()

	c.HTML(http.StatusAccepted, "result.html", buildViewModel(normalized, map[string]string{
		"overall_status": "pending",
	}))
}
