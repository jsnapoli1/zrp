package main

import (
	"net/http"

	"zrp/internal/handlers/parts"
)

// Type aliases for backward compatibility with tests and other root-level code.
type GitPLMConfig = parts.GitPLMConfig
type GitPLMURLResponse = parts.GitPLMURLResponse

func handleGetGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().GetGitPLMConfig(w, r)
}

func handleUpdateGitPLMConfig(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().UpdateGitPLMConfig(w, r)
}

func handleGetGitPLMURL(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().GetGitPLMURL(w, r, ipn)
}
