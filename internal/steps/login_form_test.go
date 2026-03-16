package steps

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

func TestHasLoginForm(t *testing.T) {
	tests := []struct {
		name    string
		htmlStr string
		want    bool
	}{
		{
			"with password input",
			`<html><body><form><input type="password"></form></body></html>`,
			true,
		},
		{
			"without password input",
			`<html><body><form><input type="text"></form></body></html>`,
			false,
		},
		{
			"case insensitive type",
			`<html><body><form><input type="Password"></form></body></html>`,
			true,
		},
		{
			"password outside form",
			`<html><body><input type="password"></body></html>`,
			true,
		},
		{
			"no form at all",
			`<html><body><p>Hello</p></body></html>`,
			false,
		},
		{
			"autocomplete current-password",
			`<html><body><form><input autocomplete="current-password"></form></body></html>`,
			true,
		},
		{
			"input name pass",
			`<html><body><form><input name="pass"></form></body></html>`,
			true,
		},
		{
			"input name password",
			`<html><body><input name="password"></body></html>`,
			true,
		},
		{
			"JS-only page no inputs",
			`<html><head><script>renderLogin()</script></head><body><div id="root"></div></body></html>`,
			false,
		},
		{
			"nested form elements",
			`<html><body><form><div><input type="text"><input type="password"></div></form></body></html>`,
			true,
		},
		{
			"multiple forms one with password",
			`<html><body><form><input type="text"></form><form><input type="password"></form></body></html>`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.htmlStr))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			got := hasLoginForm(doc)
			if got != tt.want {
				t.Errorf("hasLoginForm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoginFormRun(t *testing.T) {
	state := pipeline.NewState("example.com")
	state.SetRawHTML(`<html><body><form><input type="password"></form></body></html>`)

	step := &LoginForm{}
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("login_form")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}
	if result.Data != true {
		t.Errorf("Data = %v, want true", result.Data)
	}
}
