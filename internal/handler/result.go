package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"webpage-analyzer/internal/steps"
	"webpage-analyzer/internal/store"
	"webpage-analyzer/internal/urlutil"
)

// ResultViewModel is the data passed to the result.html template.
type ResultViewModel struct {
	URL           string
	Polling       bool
	OverallStatus string
	OverallError  string

	// Step statuses
	StepStatuses map[string]string
	StepErrors   map[string]string

	// Parsed step data
	HTMLVersion  string
	Title        string
	Headings     map[string]int
	HasLoginForm bool
	TotalLinks   int
	LinkCounts   steps.LinkCounts
}

type ResultHandler struct {
	store store.Store
}

func NewResultHandler(s store.Store) *ResultHandler {
	return &ResultHandler{store: s}
}

func (h *ResultHandler) Handle(c *gin.Context) {
	rawURL := c.Query("url")
	normalized := urlutil.Normalize(rawURL)
	jobKey := urlutil.JobKey(normalized)

	data, err := h.store.GetAll(c.Request.Context(), jobKey)
	if err != nil || len(data) == 0 {
		c.HTML(http.StatusOK, "result.html", ResultViewModel{
			URL:           normalized,
			OverallStatus: "not_found",
		})
		return
	}

	c.HTML(http.StatusOK, "result.html", buildViewModel(normalized, data))
}

// buildViewModel parses the Redis hash into a structured view model.
func buildViewModel(url string, data map[string]string) ResultViewModel {
	status := data["overall_status"]
	vm := ResultViewModel{
		URL:           url,
		Polling:       status == "pending" || status == "processing",
		OverallStatus: status,
		OverallError:  data["overall_error"],
		StepStatuses:  make(map[string]string),
		StepErrors:    make(map[string]string),
	}

	stepNames := []string{
		"url_validation", "fetch_html", "html_version",
		"title", "headings", "login_form", "links", "link_checker",
	}
	for _, name := range stepNames {
		vm.StepStatuses[name] = data["step:"+name+":status"]
		vm.StepErrors[name] = data["step:"+name+":error"]
	}

	// Parse step data
	vm.HTMLVersion = jsonString(data["step:html_version:data"])
	vm.Title = jsonString(data["step:title:data"])
	vm.Headings = jsonHeadings(data["step:headings:data"])
	vm.HasLoginForm = jsonBool(data["step:login_form:data"])
	vm.TotalLinks = jsonInt(data["step:links:data"])
	vm.LinkCounts = jsonLinkCounts(data["step:link_checker:data"])

	return vm
}

func parseStepKey(key string) (name, field string) {
	const prefix = "step:"
	if len(key) <= len(prefix) {
		return "", ""
	}
	rest := key[len(prefix):]
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == ':' {
			return rest[:i], rest[i+1:]
		}
	}
	return "", ""
}

func jsonString(raw string) string {
	if raw == "" {
		return ""
	}
	var s string
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return raw
	}
	return s
}

func jsonBool(raw string) bool {
	b, _ := strconv.ParseBool(raw)
	return b
}

func jsonInt(raw string) int {
	var n int
	_ = json.Unmarshal([]byte(raw), &n)
	return n
}

func jsonHeadings(raw string) map[string]int {
	if raw == "" {
		return nil
	}
	var m map[string]int
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func jsonLinkCounts(raw string) steps.LinkCounts {
	if raw == "" {
		return steps.LinkCounts{}
	}
	var lc steps.LinkCounts
	_ = json.Unmarshal([]byte(raw), &lc)
	return lc
}
