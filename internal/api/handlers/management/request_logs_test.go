package management

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractRequestLogHints_OpenAIResponses verifies that the parser pulls
// model, effort, provider, method, and final status out of a log file shaped
// like the real `internal/logging.FileRequestLogger` output for a Codex /
// OpenAI Responses request.
func TestExtractRequestLogHints_OpenAIResponses(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "v1-responses-2026-04-25T151734-1d423323.log")

	body := strings.Repeat(" ", 0) + `{"model":"gpt-5.5","reasoning":{"effort":"high"},"input":[{"role":"user","content":"hi"}]}`
	// Simulate a long upstream API REQUEST/RESPONSE body.
	pad := strings.Repeat("x", 300*1024)
	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"Version: test",
		"URL: /v1/responses",
		"Method: POST",
		"Timestamp: 2026-04-25T15:17:30+07:00",
		"",
		"=== HEADERS ===",
		"Content-Type: application/json",
		"",
		"=== REQUEST BODY ===",
		body,
		"",
		"=== API REQUEST 1 ===",
		"Timestamp: 2026-04-25T15:17:31+07:00",
		`{"model":"gpt-5.5","input":[]}`,
		pad,
		"",
		"=== API RESPONSE 1 ===",
		"Timestamp: 2026-04-25T15:17:32+07:00",
		"Status: 200",
		"",
		"=== RESPONSE ===",
		"Status: 200",
		"",
	}, "\n")

	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	method, model, effort, provider, status := extractRequestLogHints(logPath)

	if method != "POST" {
		t.Errorf("method: want POST, got %q", method)
	}
	if model != "gpt-5.5" {
		t.Errorf("model: want gpt-5.5, got %q", model)
	}
	if effort != "high" {
		t.Errorf("effort: want high, got %q", effort)
	}
	if provider != "openai" {
		t.Errorf("provider: want openai, got %q", provider)
	}
	if status != 200 {
		t.Errorf("status: want 200, got %d", status)
	}
}

// TestExtractRequestLogHints_CodexResponseEffort covers the observed Codex
// shape where the downstream request contains only model/input, while the
// actual effort is present later in an API response event.
func TestExtractRequestLogHints_CodexResponseEffort(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "v1-responses-2026-04-25T153006-7c2ca239.log")

	largeInput := strings.Repeat("context ", 60*1024)
	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"Version: dev",
		"URL: /v1/responses",
		"Method: POST",
		"Downstream Transport: http",
		"Upstream Transport: http",
		"Timestamp: 2026-04-25T15:29:58+07:00",
		"",
		"=== HEADERS ===",
		"Content-Type: application/json",
		"",
		"=== REQUEST BODY ===",
		`{"model":"gpt-5.5","input":[{"role":"user","content":[{"type":"input_text","text":"` + largeInput + `"}]}]}`,
		"",
		"=== API REQUEST 1 ===",
		"Timestamp: 2026-04-25T15:29:58+07:00",
		"Upstream URL: https://chatgpt.com/backend-api/codex/responses",
		"HTTP Method: POST",
		"Body: " + `{"model":"gpt-5.5","input":[{"role":"user","content":[{"type":"input_text","text":"` + largeInput + `"}]}],"stream":true}`,
		"",
		"=== API RESPONSE 1 ===",
		"Timestamp: 2026-04-25T15:30:00+07:00",
		"Status: 200",
		"Headers:",
		"",
		`data: {"type":"response.created","response":{"id":"resp_123","model":"gpt-5.5","reasoning":{"effort":"medium","summary":null}}}`,
		"",
		"=== RESPONSE ===",
		"Status: 200",
	}, "\n")

	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	method, model, effort, provider, status := extractRequestLogHints(logPath)
	if method != "POST" {
		t.Errorf("method: want POST, got %q", method)
	}
	if model != "gpt-5.5" {
		t.Errorf("model: want gpt-5.5, got %q", model)
	}
	if effort != "medium" {
		t.Errorf("effort: want medium, got %q", effort)
	}
	if provider != "openai" {
		t.Errorf("provider: want openai, got %q", provider)
	}
	if status != 200 {
		t.Errorf("status: want 200, got %d", status)
	}
}

// TestExtractRequestLogHints_ClaudeMessages covers the Claude-style payload
// using thinking.budget_tokens (mapped to a coarse effort bucket).
func TestExtractRequestLogHints_ClaudeMessages(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "v1-messages-2026-04-25T101010-abc123.log")

	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"URL: /v1/messages",
		"Method: POST",
		"Timestamp: 2026-04-25T10:10:10+07:00",
		"",
		"=== HEADERS ===",
		"x-api-key: ***",
		"",
		"=== REQUEST BODY ===",
		`{"model":"claude-3-5-sonnet-20241022","thinking":{"type":"enabled","budget_tokens":8192},"messages":[{"role":"user","content":"hi"}]}`,
		"",
		"=== API REQUEST 1 ===",
		"Timestamp: 2026-04-25T10:10:11+07:00",
		`{"model":"claude-3-5-sonnet-20241022"}`,
		"",
		"=== API RESPONSE 1 ===",
		"Timestamp: 2026-04-25T10:10:12+07:00",
		"Status: 200",
		"",
		"=== RESPONSE ===",
		"Status: 200",
	}, "\n")

	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, model, effort, provider, status := extractRequestLogHints(logPath)
	if model != "claude-3-5-sonnet-20241022" {
		t.Errorf("model: want claude-3-5-sonnet-20241022, got %q", model)
	}
	// 8192 budget tokens fall into the `high` bucket per budgetToEffort.
	if effort != "high" {
		t.Errorf("effort: want high, got %q", effort)
	}
	if provider != "claude" {
		t.Errorf("provider: want claude, got %q", provider)
	}
	if status != 200 {
		t.Errorf("status: want 200, got %d", status)
	}
}

// TestExtractRequestLogHints_GeminiThinkingBudget covers the Gemini-style
// nested generationConfig.thinkingConfig.thinkingBudget shape.
func TestExtractRequestLogHints_GeminiThinkingBudget(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "v1beta-models-gemini-pro-generateContent-2026-04-25T090000-xyz.log")

	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"URL: /v1beta/models/gemini-2.5-pro:generateContent",
		"Method: POST",
		"",
		"=== HEADERS ===",
		"",
		"=== REQUEST BODY ===",
		`{"model":"gemini-2.5-pro","contents":[{"parts":[{"text":"hi"}]}],"generationConfig":{"thinkingConfig":{"thinkingBudget":2048}}}`,
		"",
		"=== RESPONSE ===",
		"Status: 200",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, model, effort, provider, _ := extractRequestLogHints(logPath)
	if model != "gemini-2.5-pro" {
		t.Errorf("model: want gemini-2.5-pro, got %q", model)
	}
	// 2048 falls in the `medium` bucket.
	if effort != "medium" {
		t.Errorf("effort: want medium, got %q", effort)
	}
	if provider != "gemini" {
		t.Errorf("provider: want gemini, got %q", provider)
	}
}

// TestExtractRequestLogHints_ErrorLog covers a forced error log (no
// `=== REQUEST BODY ===` body but still parsable header).
func TestExtractRequestLogHints_ErrorLog(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "error-v1-responses-2026-04-25T080000-err.log")

	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"URL: /v1/responses",
		"Method: POST",
		"",
		"=== HEADERS ===",
		"",
		"=== REQUEST BODY ===",
		`{"model":"gpt-5","reasoning_effort":"low"}`,
		"",
		"=== API ERROR RESPONSE ===",
		"HTTP Status: 503",
		"upstream temporarily unavailable",
		"",
		"=== RESPONSE ===",
		"Status: 503",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, model, effort, provider, status := extractRequestLogHints(logPath)
	if model != "gpt-5" {
		t.Errorf("model: want gpt-5, got %q", model)
	}
	if effort != "low" {
		t.Errorf("effort: want low, got %q", effort)
	}
	if provider != "openai" {
		t.Errorf("provider: want openai, got %q", provider)
	}
	if status != 503 {
		t.Errorf("status: want 503, got %d", status)
	}
}

// TestExtractRequestLogHints_ModelSuffixFallback covers the case where the
// body uses a `model:effort` suffix (canonical across the Codex/OpenAI
// integration) and no explicit reasoning/thinking field.
func TestExtractRequestLogHints_ModelSuffixFallback(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "v1-responses-suffix-1.log")

	content := strings.Join([]string{
		"=== REQUEST INFO ===",
		"URL: /v1/responses",
		"Method: POST",
		"",
		"=== HEADERS ===",
		"",
		"=== REQUEST BODY ===",
		`{"model":"gpt-5.3-codex:xhigh","input":[]}`,
		"",
		"=== RESPONSE ===",
		"Status: 200",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, _, effort, _, _ := extractRequestLogHints(logPath)
	if effort != "xhigh" {
		t.Errorf("effort: want xhigh, got %q", effort)
	}
}

// TestExtractRequestIDFromName ensures the trailing-id heuristic survives
// realistic filenames the writer can produce.
func TestExtractRequestIDFromName(t *testing.T) {
	cases := map[string]string{
		"v1-responses-2026-04-25T151734-1d423323.log":         "1d423323",
		"error-v1-messages-2026-04-25T080000-abc-def.log":     "def",
		"v1beta-models-gemini-pro-generateContent-99.log":     "99",
		"root-2026-04-25T000000-trailing-id-with-hyphens.log": "hyphens",
		"no-extension": "extension",
	}
	for name, want := range cases {
		if got := extractRequestIDFromName(name); got != want {
			t.Errorf("extractRequestIDFromName(%q) = %q; want %q", name, got, want)
		}
	}
}
