package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// parseDoc extracts a Document from APIResponse-wrapped JSON
func parseDoc(t *testing.T, body []byte) Document {
	t.Helper()
	var wrap struct {
		Data Document `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse doc: %v", err)
	}
	return wrap.Data
}

func parseDocVersions(t *testing.T, body []byte) []DocumentVersion {
	t.Helper()
	var wrap struct {
		Data []DocumentVersion `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse versions: %v", err)
	}
	return wrap.Data
}

func parseDocVersion(t *testing.T, body []byte) DocumentVersion {
	t.Helper()
	var wrap struct {
		Data DocumentVersion `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse version: %v", err)
	}
	return wrap.Data
}

func TestDocVersionCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a document
	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs", `{"title":"Test Doc","category":"spec","content":"Version A content","revision":"A"}`, cookie)
	handleCreateDoc(w, req)
	if w.Code != 200 {
		t.Fatalf("create doc: %d %s", w.Code, w.Body.String())
	}
	doc := parseDoc(t, w.Body.Bytes())

	// Update document (should auto-snapshot)
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/docs/"+doc.ID, `{"title":"Test Doc","category":"spec","content":"Version A updated","revision":"A"}`, cookie)
	handleUpdateDoc(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("update doc: %d %s", w.Code, w.Body.String())
	}

	// List versions
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions", "", cookie)
	handleListDocVersions(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("list versions: %d %s", w.Code, w.Body.String())
	}
	versions := parseDocVersions(t, w.Body.Bytes())
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
	if versions[0].Revision != "A" {
		t.Errorf("expected revision A, got %s", versions[0].Revision)
	}
	if versions[0].Content != "Version A content" {
		t.Errorf("expected original content, got %s", versions[0].Content)
	}
}

func TestDocRelease(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs", `{"title":"Release Test","category":"spec","content":"Draft content","revision":"A","status":"draft"}`, cookie)
	handleCreateDoc(w, req)
	doc := parseDoc(t, w.Body.Bytes())

	// Release
	w = httptest.NewRecorder()
	req = authedRequest("POST", "/api/v1/docs/"+doc.ID+"/release", "", cookie)
	handleReleaseDoc(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("release: %d %s", w.Code, w.Body.String())
	}

	released := parseDoc(t, w.Body.Bytes())
	if released.Status != "released" {
		t.Errorf("expected status released, got %s", released.Status)
	}

	// Verify version was created
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions", "", cookie)
	handleListDocVersions(w, req, doc.ID)
	versions := parseDocVersions(t, w.Body.Bytes())
	if len(versions) != 1 {
		t.Fatalf("expected 1 version after release, got %d", len(versions))
	}
}

func TestDocRevert(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create doc with content A
	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs", `{"title":"Revert Test","category":"spec","content":"Original content","revision":"A"}`, cookie)
	handleCreateDoc(w, req)
	doc := parseDoc(t, w.Body.Bytes())

	// Update to content B (snapshots A)
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/docs/"+doc.ID, `{"title":"Revert Test","category":"spec","content":"Updated content","revision":"B"}`, cookie)
	handleUpdateDoc(w, req, doc.ID)

	// Revert to A
	w = httptest.NewRecorder()
	req = authedRequest("POST", "/api/v1/docs/"+doc.ID+"/revert/A", "", cookie)
	handleRevertDoc(w, req, doc.ID, "A")
	if w.Code != 200 {
		t.Fatalf("revert: %d %s", w.Code, w.Body.String())
	}

	reverted := parseDoc(t, w.Body.Bytes())
	if reverted.Content != "Original content" {
		t.Errorf("expected original content after revert, got %s", reverted.Content)
	}
	if reverted.Revision != "A" {
		t.Errorf("expected revision A after revert, got %s", reverted.Revision)
	}
}

func TestDocDiff(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs", `{"title":"Diff Test","category":"spec","content":"Line 1\nLine 2\nLine 3","revision":"A"}`, cookie)
	handleCreateDoc(w, req)
	doc := parseDoc(t, w.Body.Bytes())

	// Update (snapshots revision A)
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/docs/"+doc.ID, `{"title":"Diff Test","category":"spec","content":"Line 1\nLine 2 modified\nLine 3\nLine 4","revision":"B"}`, cookie)
	handleUpdateDoc(w, req, doc.ID)

	// Snapshot revision B too
	snapshotDocumentVersion(doc.ID, "Snapshot B", "admin", nil)

	// Get diff from A to B
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/docs/"+doc.ID+"/diff?from=A&to=B", "", cookie)
	handleDocDiff(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("diff: %d %s", w.Code, w.Body.String())
	}

	var wrap struct {
		Data struct {
			From  string     `json:"from"`
			To    string     `json:"to"`
			Lines []DiffLine `json:"lines"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &wrap)
	if wrap.Data.From != "A" || wrap.Data.To != "B" {
		t.Errorf("unexpected from/to: %s/%s", wrap.Data.From, wrap.Data.To)
	}
	if len(wrap.Data.Lines) == 0 {
		t.Error("expected diff lines")
	}
}

func TestDocDiffMissingParams(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	w := httptest.NewRecorder()
	req := authedRequest("GET", "/api/v1/docs/DOC-001/diff", "", cookie)
	handleDocDiff(w, req, "DOC-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetDocVersionByRevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs", `{"title":"Version Get Test","category":"spec","content":"Content A","revision":"A"}`, cookie)
	handleCreateDoc(w, req)
	doc := parseDoc(t, w.Body.Bytes())

	// Update to create version snapshot
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/docs/"+doc.ID, `{"title":"Version Get Test","category":"spec","content":"Content B","revision":"B"}`, cookie)
	handleUpdateDoc(w, req, doc.ID)

	// Get version A
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions/A", "", cookie)
	handleGetDocVersion(w, req, doc.ID, "A")
	if w.Code != 200 {
		t.Fatalf("get version: %d %s", w.Code, w.Body.String())
	}
	v := parseDocVersion(t, w.Body.Bytes())
	if v.Content != "Content A" {
		t.Errorf("expected Content A, got %s", v.Content)
	}

	// Get nonexistent version
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions/Z", "", cookie)
	handleGetDocVersion(w, req, doc.ID, "Z")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNextRevision(t *testing.T) {
	cases := []struct{ in, out string }{
		{"", "A"},
		{"A", "B"},
		{"B", "C"},
		{"Z", "AA"},
		{"AA", "AB"},
		{"AZ", "BA"},
	}
	for _, c := range cases {
		got := nextRevision(c.in)
		if got != c.out {
			t.Errorf("nextRevision(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestGitDocsSettings(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// PUT settings
	w := httptest.NewRecorder()
	req := authedRequest("PUT", "/api/v1/settings/git-docs", `{"repo_url":"https://github.com/test/repo","branch":"main","token":"secret123"}`, cookie)
	handlePutGitDocsSettings(w, req)
	if w.Code != 200 {
		t.Fatalf("put settings: %d %s", w.Code, w.Body.String())
	}

	// GET settings (token should be masked)
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/settings/git-docs", "", cookie)
	handleGetGitDocsSettings(w, req)
	if w.Code != 200 {
		t.Fatalf("get settings: %d %s", w.Code, w.Body.String())
	}
	var wrap struct {
		Data GitDocsConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &wrap)
	cfg := wrap.Data
	if cfg.RepoURL != "https://github.com/test/repo" {
		t.Errorf("expected repo URL, got %s", cfg.RepoURL)
	}
	if cfg.Token != "***" {
		t.Errorf("expected masked token, got %s", cfg.Token)
	}
}

func TestPushDocToGitNoConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/docs/DOC-2026-001/push", "", cookie)
	handlePushDocToGit(w, req, "DOC-2026-001")
	if w.Code != 400 {
		t.Errorf("expected 400 when git not configured, got %d", w.Code)
	}
}

func TestComputeDiff(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "x", "c", "d"}
	diff := computeDiff(from, to)

	types := make([]string, len(diff))
	for i, d := range diff {
		types[i] = d.Type
	}
	expected := []string{"same", "removed", "added", "same", "added"}
	if len(types) != len(expected) {
		t.Fatalf("diff length %d, expected %d: %v", len(types), len(expected), diff)
	}
	for i, e := range expected {
		if types[i] != e {
			t.Errorf("diff[%d].Type = %s, expected %s", i, types[i], e)
		}
	}
}
