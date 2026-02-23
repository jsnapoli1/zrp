package parts

import (
	"net/http"
	"strings"

	"zrp/internal/response"
)

// GitPLMConfig holds the gitplm-ui integration settings.
type GitPLMConfig struct {
	BaseURL string `json:"base_url"`
}

// GitPLMURLResponse is the response for the gitplm URL endpoint.
type GitPLMURLResponse struct {
	URL        string `json:"url"`
	Configured bool   `json:"configured"`
}

// GetGitPLMConfig handles GET /api/gitplm/config.
func (h *Handler) GetGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	var baseURL string
	err := h.DB.QueryRow("SELECT value FROM app_settings WHERE key = 'gitplm_base_url'").Scan(&baseURL)
	if err != nil {
		response.JSON(w, GitPLMConfig{BaseURL: ""})
		return
	}
	response.JSON(w, GitPLMConfig{BaseURL: baseURL})
}

// UpdateGitPLMConfig handles PUT /api/gitplm/config.
func (h *Handler) UpdateGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	var cfg GitPLMConfig
	if err := response.DecodeBody(r, &cfg); err != nil {
		response.Err(w, "invalid request body", 400)
		return
	}

	// Trim trailing slash
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	_, err := h.DB.Exec(`INSERT INTO app_settings (key, value) VALUES ('gitplm_base_url', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, cfg.BaseURL)
	if err != nil {
		response.Err(w, "failed to save setting", 500)
		return
	}
	response.JSON(w, cfg)
}

// GetGitPLMURL handles GET /api/parts/:ipn/gitplm-url.
func (h *Handler) GetGitPLMURL(w http.ResponseWriter, r *http.Request, ipn string) {
	var baseURL string
	err := h.DB.QueryRow("SELECT value FROM app_settings WHERE key = 'gitplm_base_url'").Scan(&baseURL)
	if err != nil || baseURL == "" {
		response.JSON(w, GitPLMURLResponse{URL: "", Configured: false})
		return
	}
	url := baseURL + "/parts/" + ipn
	response.JSON(w, GitPLMURLResponse{URL: url, Configured: true})
}
