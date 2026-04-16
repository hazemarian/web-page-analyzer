package steps

import (
	"context"
	"strings"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

// LoginForm is Stage 3. Detects whether the page contains a login form.
// Detection checks for:
//   - <input type="password"> anywhere in the document
//   - <input autocomplete="current-password">
//   - <input name="pass"> or <input name="password">
type LoginForm struct{}

func (s *LoginForm) Name() string { return "login_form" }
func (s *LoginForm) Stage() int   { return 3 }

func (s *LoginForm) Run(ctx context.Context, state *pipeline.State) error {
	doc, err := html.Parse(strings.NewReader(state.GetRawHTML()))
	if err != nil {
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: "failed to parse HTML"})
		return nil
	}

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: hasLoginForm(doc)})
	return nil
}

func hasLoginForm(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "input" && isPasswordInput(n) {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hasLoginForm(c) {
			return true
		}
	}
	return false
}

// isPasswordInput returns true if the input element looks like a password field.
func isPasswordInput(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "type" && strings.EqualFold(attr.Val, "password") {
			return true
		}
		if attr.Key == "autocomplete" && strings.EqualFold(attr.Val, "current-password") {
			return true
		}
		if attr.Key == "name" && (strings.EqualFold(attr.Val, "pass") || strings.EqualFold(attr.Val, "password")) {
			return true
		}
	}
	return false
}
