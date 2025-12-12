// Package copilot provides model fetching capabilities for GitHub Copilot.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	log "github.com/sirupsen/logrus"
)

// CopilotModelResponse represents the /models endpoint response
type CopilotModelResponse struct {
	Models []CopilotModelInfo `json:"models"`
	Data   []CopilotModelInfo `json:"data"` // Alternative format (OpenAI-style)
}

// CopilotModelInfo represents a model from Copilot API
type CopilotModelInfo struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name,omitempty"`
	DisplayName         string   `json:"display_name,omitempty"`
	ModelFamily         string   `json:"model_family,omitempty"`
	Vendor              string   `json:"vendor,omitempty"`
	Version             string   `json:"version,omitempty"`
	Object              string   `json:"object,omitempty"`
	Created             int64    `json:"created,omitempty"`
	OwnedBy             string   `json:"owned_by,omitempty"`
	ContextWindow       int      `json:"context_window,omitempty"`
	MaxOutputTokens     int      `json:"max_output_tokens,omitempty"`
	InputModalities     []string `json:"input_modalities,omitempty"`
	OutputModalities    []string `json:"output_modalities,omitempty"`
	IsPreview           bool     `json:"is_preview,omitempty"`
	SupportedParameters []string `json:"supported_parameters,omitempty"`
}

// FetchModels retrieves available models from GitHub Copilot API
func (c *CopilotAuth) FetchModels(ctx context.Context, apiToken *CopilotAPIToken) ([]*registry.ModelInfo, error) {
	if apiToken == nil || apiToken.Token == "" {
		return nil, fmt.Errorf("copilot: api token required for fetching models")
	}

	modelsURL := copilotAPIEndpoint + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("copilot: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", copilotUserAgent)
	req.Header.Set("Editor-Version", copilotEditorVersion)
	req.Header.Set("Editor-Plugin-Version", copilotPluginVersion)
	req.Header.Set("Openai-Intent", copilotOpenAIIntent)
	req.Header.Set("Copilot-Integration-Id", copilotIntegrationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("copilot: failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("copilot: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Debugf("copilot: models endpoint returned %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("copilot: models endpoint returned status %d", resp.StatusCode)
	}

	// Try to parse response
	var modelsResp CopilotModelResponse
	if err := json.Unmarshal(bodyBytes, &modelsResp); err != nil {
		// Try parsing as array directly
		var models []CopilotModelInfo
		if err2 := json.Unmarshal(bodyBytes, &models); err2 != nil {
			log.Debugf("copilot: failed to parse models response: %v", err)
			return nil, fmt.Errorf("copilot: failed to parse models: %w", err)
		}
		modelsResp.Models = models
	}

	// Use Data if Models is empty (OpenAI-style response)
	copilotModels := modelsResp.Models
	if len(copilotModels) == 0 && len(modelsResp.Data) > 0 {
		copilotModels = modelsResp.Data
	}

	log.Infof("copilot: fetched %d models from API", len(copilotModels))

	// Convert to ModelInfo
	result := make([]*registry.ModelInfo, 0, len(copilotModels))
	now := time.Now().Unix()

	for _, m := range copilotModels {
		if m.ID == "" {
			continue
		}

		modelInfo := &registry.ModelInfo{
			ID:      m.ID,
			Object:  "model",
			Created: now,
			OwnedBy: "copilot",
			Type:    "copilot",
		}

		// Set display name
		if m.DisplayName != "" {
			modelInfo.DisplayName = m.DisplayName
		} else if m.Name != "" {
			modelInfo.DisplayName = m.Name
		} else {
			modelInfo.DisplayName = m.ID
		}

		// Set context length
		if m.ContextWindow > 0 {
			modelInfo.ContextLength = m.ContextWindow
		} else {
			modelInfo.ContextLength = 128000 // Default
		}

		// Set max output tokens
		if m.MaxOutputTokens > 0 {
			modelInfo.MaxCompletionTokens = m.MaxOutputTokens
		} else {
			modelInfo.MaxCompletionTokens = 16384 // Default
		}

		// Build description
		vendor := m.Vendor
		if vendor == "" {
			vendor = "OpenAI"
		}
		modelInfo.Description = fmt.Sprintf("%s %s via Copilot (auto-discovered)", vendor, m.ID)
		if m.IsPreview {
			modelInfo.Description += " [Preview]"
		}

		result = append(result, modelInfo)
	}

	return result, nil
}

// FetchAndRegisterModels fetches models and registers them to the global registry
func (c *CopilotAuth) FetchAndRegisterModels(ctx context.Context, apiToken *CopilotAPIToken, clientID string) error {
	models, err := c.FetchModels(ctx, apiToken)
	if err != nil {
		return err
	}

	if len(models) == 0 {
		log.Debugf("copilot: no models fetched for client %s", clientID)
		return nil
	}

	globalRegistry := registry.GetGlobalRegistry()

	// Register client with all discovered models
	// RegisterClient expects (clientID, provider, []*ModelInfo)
	globalRegistry.RegisterClient(clientID, "github-copilot", models)

	log.Infof("copilot: registered %d auto-discovered models for client %s", len(models), clientID)
	return nil
}
