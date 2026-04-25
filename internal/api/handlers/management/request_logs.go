package management

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// requestLogEntry summarises a single per-request log file for the management UI.
type requestLogEntry struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	IsError  bool   `json:"is_error"`
	Method   string `json:"method,omitempty"`
	Model    string `json:"model,omitempty"`
	Effort   string `json:"effort,omitempty"`
	Provider string `json:"provider,omitempty"`
	Status   int    `json:"status,omitempty"`
}

// requestLogDefaultLimit caps the number of log files returned by default
// to keep the management UI responsive.
const requestLogDefaultLimit = 100

// requestLogMaxLimit is the maximum value the caller can request via ?limit=.
const requestLogMaxLimit = 500

// GetRequestLogList returns a summary list of per-request log files
// (both successful when request-log is enabled, and error-*.log files).
//
// Query params:
//   - limit: max number of entries (default 100, hard cap 500)
//
// Response:
//
//	{ "files": [ { "name", "id", "path", "size", "modified", "is_error",
//	              "method", "model", "effort", "provider", "status" }, ... ],
//	  "total": <total before limit>,
//	  "limit": <effective limit>,
//	  "request_log_enabled": <bool>,
//	  "log_dir": "<absolute logs directory>" }
func (h *Handler) GetRequestLogList(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler unavailable"})
		return
	}
	if h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "configuration unavailable"})
		return
	}

	dir := h.logDirectory()
	if strings.TrimSpace(dir) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "log directory not configured"})
		return
	}

	limit := requestLogDefaultLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, errParse := strconv.Atoi(raw); errParse == nil && parsed > 0 {
			if parsed > requestLogMaxLimit {
				parsed = requestLogMaxLimit
			}
			limit = parsed
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{
				"files":               []any{},
				"total":               0,
				"limit":               limit,
				"request_log_enabled": h.cfg.RequestLog,
				"log_dir":             dir,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list request logs: %v", err)})
		return
	}

	type rawFile struct {
		name    string
		modTime int64
		size    int64
		isError bool
	}

	files := make([]rawFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isRequestLogFile(name) {
			continue
		}
		info, errInfo := entry.Info()
		if errInfo != nil {
			continue
		}
		files = append(files, rawFile{
			name:    name,
			modTime: info.ModTime().Unix(),
			size:    info.Size(),
			isError: strings.HasPrefix(name, "error-"),
		})
	}

	// Newest first.
	sort.Slice(files, func(i, j int) bool { return files[i].modTime > files[j].modTime })
	total := len(files)

	if len(files) > limit {
		files = files[:limit]
	}

	out := make([]requestLogEntry, 0, len(files))
	for _, f := range files {
		entry := requestLogEntry{
			Name:     f.name,
			Path:     filepath.Join(dir, f.name),
			Size:     f.size,
			Modified: f.modTime,
			IsError:  f.isError,
			ID:       extractRequestIDFromName(f.name),
		}
		method, model, effort, provider, status := extractRequestLogHints(filepath.Join(dir, f.name))
		entry.Method = method
		entry.Model = model
		entry.Effort = effort
		entry.Provider = provider
		entry.Status = status
		out = append(out, entry)
	}

	c.JSON(http.StatusOK, gin.H{
		"files":               out,
		"total":               total,
		"limit":               limit,
		"request_log_enabled": h.cfg.RequestLog,
		"log_dir":             dir,
	})
}

// isRequestLogFile reports whether name looks like a per-request log file
// produced by internal/logging.FileRequestLogger.
func isRequestLogFile(name string) bool {
	if !strings.HasSuffix(name, ".log") {
		return false
	}
	if name == defaultLogFileName {
		return false
	}
	if _, ok := rotationOrder(name); ok {
		return false
	}
	return true
}

// extractRequestIDFromName pulls the trailing request ID from the filename.
// The logger names files as "<sanitized-path>-<timestamp>-<requestID>.log"
// so we take the last hyphen-separated segment before the extension.
func extractRequestIDFromName(name string) string {
	base := strings.TrimSuffix(name, ".log")
	if idx := strings.LastIndexByte(base, '-'); idx >= 0 && idx < len(base)-1 {
		return base[idx+1:]
	}
	return ""
}

// extractRequestLogHints streams through a request log and best-effort parses
// the HTTP method, downstream URL path, model, reasoning/thinking effort,
// provider hint, and final response status. Empty fields are returned when a
// field is not detectable.
//
// The log format is the one produced by `internal/logging.FileRequestLogger`:
//
//	=== REQUEST INFO ===
//	URL: /v1/responses
//	Method: POST
//	Timestamp: ...
//	=== HEADERS ===
//	...
//	=== REQUEST BODY ===
//	{...downstream JSON...}
//	=== API REQUEST 1 ===
//	...
//	Body: {...upstream JSON...}
//	=== API RESPONSE 1 ===
//	Status: 200
//	...
//	data: {...upstream response event JSON...}
//	...
//	=== RESPONSE ===
//	Status: 200
func extractRequestLogHints(path string) (method, model, effort, provider string, status int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	urlPath := ""
	reader := bufio.NewReaderSize(f, 64*1024)
	for {
		line, errRead := reader.ReadString('\n')
		if line != "" {
			scanRequestLogLine(line, &method, &urlPath, &model, &effort, &provider, &status)
			if method != "" && provider != "" && model != "" && effort != "" && status != 0 {
				break
			}
		}
		if errRead != nil {
			if errRead != io.EOF {
				return
			}
			break
		}
	}

	if provider == "" {
		provider = guessProviderFromPath(urlPath)
	}
	if effort == "" && model != "" {
		// Fall back to the model-name suffix (e.g. "gpt-5.3-codex:high").
		if idx := strings.LastIndexByte(model, ':'); idx >= 0 && idx < len(model)-1 {
			suffix := strings.ToLower(model[idx+1:])
			switch suffix {
			case "none", "low", "medium", "high", "xhigh", "minimal":
				effort = suffix
			}
		}
	}

	return
}

func scanRequestLogLine(line string, method, urlPath, model, effort, provider *string, status *int) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	switch {
	case *method == "" && strings.HasPrefix(trimmed, "Method: "):
		*method = strings.TrimSpace(strings.TrimPrefix(trimmed, "Method:"))
	case *urlPath == "" && strings.HasPrefix(trimmed, "URL: "):
		*urlPath = strings.TrimSpace(strings.TrimPrefix(trimmed, "URL:"))
		if *provider == "" {
			*provider = guessProviderFromPath(*urlPath)
		}
	case *status == 0 && strings.HasPrefix(trimmed, "Status: "):
		*status = parseLeadingStatusCode(strings.TrimSpace(strings.TrimPrefix(trimmed, "Status:")))
	case *status == 0 && strings.HasPrefix(trimmed, "HTTP Status: "):
		*status = parseLeadingStatusCode(strings.TrimSpace(strings.TrimPrefix(trimmed, "HTTP Status:")))
	}

	if *model != "" && *effort != "" && *provider != "" {
		return
	}
	if candidate := extractJSONCandidate(trimmed); candidate != "" {
		applyJSONHints(candidate, model, effort, provider)
	}
}

func parseLeadingStatusCode(text string) int {
	if text == "" {
		return 0
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0
	}
	status, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return status
}

func extractJSONCandidate(line string) string {
	switch {
	case strings.HasPrefix(line, "Body:"):
		line = strings.TrimSpace(strings.TrimPrefix(line, "Body:"))
	case strings.HasPrefix(line, "data:"):
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	default:
		openIdx := strings.IndexByte(line, '{')
		if openIdx < 0 {
			return ""
		}
		line = line[openIdx:]
	}
	if line == "" || line == "[DONE]" || !strings.HasPrefix(line, "{") {
		return ""
	}
	return line
}

func applyJSONHints(body string, model, effort, provider *string) {
	if *model == "" {
		for _, path := range []string{"model", "response.model"} {
			if v := gjson.Get(body, path); v.Exists() && v.String() != "" {
				*model = v.String()
				break
			}
		}
	}
	if *effort == "" {
		if e := extractEffort(body); e != "" {
			*effort = e
		}
	}
	if *provider == "" {
		if p := guessProviderFromBody(body); p != "" {
			*provider = p
		}
	}
}

// extractEffort inspects a JSON request body and returns a normalised
// reasoning/thinking effort label (low/medium/high/xhigh) when one of the
// recognised provider-specific shapes is present.
func extractEffort(body string) string {
	if body == "" {
		return ""
	}

	// OpenAI Responses API: { "reasoning": { "effort": "high" } }
	for _, path := range []string{"reasoning.effort", "response.reasoning.effort"} {
		if v := gjson.Get(body, path); v.Exists() && v.String() != "" {
			return strings.ToLower(v.String())
		}
	}
	// OpenAI Chat Completions (GPT-5+): { "reasoning_effort": "high" }
	for _, path := range []string{"reasoning_effort", "response.reasoning_effort"} {
		if v := gjson.Get(body, path); v.Exists() && v.String() != "" {
			return strings.ToLower(v.String())
		}
	}
	// Codex internal/canonical key.
	for _, path := range []string{"thinking_effort", "response.thinking_effort"} {
		if v := gjson.Get(body, path); v.Exists() && v.String() != "" {
			return strings.ToLower(v.String())
		}
	}
	// Claude Messages API: { "thinking": { "type": "enabled", "budget_tokens": 8192 } }.
	for _, path := range []string{"thinking.budget_tokens", "response.thinking.budget_tokens"} {
		if v := gjson.Get(body, path); v.Exists() && v.Int() > 0 {
			return budgetToEffort(v.Int())
		}
	}
	for _, path := range []string{"thinking.type", "response.thinking.type"} {
		if v := gjson.Get(body, path); v.Exists() && v.String() != "" {
			// Surface the type when budget tokens are absent.
			return strings.ToLower(v.String())
		}
	}
	// Gemini: { "generationConfig": { "thinkingConfig": { "thinkingBudget": 4096 } } }
	for _, path := range []string{
		"generationConfig.thinkingConfig.thinkingBudget",
		"generation_config.thinking_config.thinking_budget",
		"response.generationConfig.thinkingConfig.thinkingBudget",
		"response.generation_config.thinking_config.thinking_budget",
	} {
		if v := gjson.Get(body, path); v.Exists() && v.Int() > 0 {
			return budgetToEffort(v.Int())
		}
	}

	return ""
}

// budgetToEffort buckets a numeric thinking-budget value into a coarse effort
// label so the UI can render it consistently next to providers that emit a
// pre-set effort string.
func budgetToEffort(budget int64) string {
	switch {
	case budget <= 0:
		return "none"
	case budget <= 1024:
		return "low"
	case budget <= 4096:
		return "medium"
	case budget <= 16384:
		return "high"
	default:
		return "xhigh"
	}
}

// guessProviderFromPath infers a provider key from the downstream URL path
// recorded at the top of the log file (e.g. `/v1/responses`).
func guessProviderFromPath(rawPath string) string {
	low := strings.ToLower(strings.TrimSpace(rawPath))
	if low == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(low, "/v1/responses"):
		return "openai"
	case strings.HasPrefix(low, "/v1/chat/completions"),
		strings.HasPrefix(low, "/v1/completions"),
		strings.HasPrefix(low, "/v1/embeddings"):
		return "openai"
	case strings.HasPrefix(low, "/v1/messages"):
		return "claude"
	case strings.Contains(low, "/v1beta/models"),
		strings.Contains(low, ":generatecontent"),
		strings.Contains(low, ":streamgeneratecontent"):
		return "gemini"
	case strings.Contains(low, "/codex"):
		return "codex"
	case strings.HasPrefix(low, "/amp/") || strings.Contains(low, "/ampcode"):
		return "amp"
	}
	return ""
}

// guessProviderFromURL infers a provider key from a fully-qualified upstream
// URL. Best-effort; returns "" when no match.
func guessProviderFromURL(rawURL string) string {
	low := strings.ToLower(rawURL)
	switch {
	case strings.Contains(low, "openai.com") || strings.Contains(low, "/responses"):
		return "openai"
	case strings.Contains(low, "anthropic.com") || strings.Contains(low, "/v1/messages"):
		return "claude"
	case strings.Contains(low, "googleapis.com/v1beta/models"), strings.Contains(low, "generativelanguage.googleapis.com"):
		return "gemini"
	case strings.Contains(low, "aiplatform.googleapis.com"):
		return "vertex"
	case strings.Contains(low, "moonshot") || strings.Contains(low, "kimi"):
		return "kimi"
	case strings.Contains(low, "antigravity"):
		return "antigravity"
	case strings.Contains(low, "/codex/"):
		return "codex"
	}
	return ""
}

// guessProviderFromBody is a fallback that infers provider from JSON shape.
func guessProviderFromBody(body string) string {
	if gjson.Get(body, "messages.0.role").Exists() {
		// Both Claude and OpenAI use messages[]; differentiate by Claude-only fields.
		if gjson.Get(body, "system").Type == gjson.JSON || gjson.Get(body, "max_tokens").Exists() && gjson.Get(body, "thinking").Exists() {
			return "claude"
		}
		return "openai"
	}
	if gjson.Get(body, "contents.0.parts").Exists() {
		return "gemini"
	}
	return ""
}
