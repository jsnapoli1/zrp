package main

import (
	"net/http"

	"zrp/internal/handlers/engineering"
)

// Type alias for backward compatibility.
type DiffLine = engineering.DiffLine

// nextRevision delegates to the engineering package.
func nextRevision(rev string) string {
	return engineering.NextRevision(rev)
}

// computeDiff delegates to the engineering package.
func computeDiff(from, to []string) []engineering.DiffLine {
	return engineering.ComputeDiff(from, to)
}

// snapshotDocumentVersion delegates to the engineering handler.
func snapshotDocumentVersion(docID, changeSummary, createdBy string, ecoID *string) error {
	return getEngineeringHandler().SnapshotDocumentVersion(docID, changeSummary, createdBy, ecoID)
}

// bumpDocRevisionsForECO delegates to the engineering handler.
func bumpDocRevisionsForECO(ecoID string, username string) error {
	return getEngineeringHandler().BumpDocRevisionsForECO(ecoID, username)
}

func handleListDocVersions(w http.ResponseWriter, r *http.Request, docID string) {
	getEngineeringHandler().ListDocVersions(w, r, docID)
}

func handleGetDocVersion(w http.ResponseWriter, r *http.Request, docID, revision string) {
	getEngineeringHandler().GetDocVersion(w, r, docID, revision)
}

func handleDocDiff(w http.ResponseWriter, r *http.Request, docID string) {
	getEngineeringHandler().DocDiff(w, r, docID)
}

func handleReleaseDoc(w http.ResponseWriter, r *http.Request, docID string) {
	getEngineeringHandler().ReleaseDoc(w, r, docID)
}

func handleRevertDoc(w http.ResponseWriter, r *http.Request, docID, revision string) {
	getEngineeringHandler().RevertDoc(w, r, docID, revision)
}
