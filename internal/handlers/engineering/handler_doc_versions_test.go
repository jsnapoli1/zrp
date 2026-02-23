package engineering_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/engineering"
	"zrp/internal/models"
	"zrp/internal/testutil"

	_ "modernc.org/sqlite"
)

// parseDocFromBody extracts a Document from APIResponse-wrapped JSON
func parseDocFromBody(t *testing.T, body []byte) models.Document {
	t.Helper()
	var wrap struct {
		Data models.Document `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse doc: %v", err)
	}
	return wrap.Data
}

func parseDocVersionsFromBody(t *testing.T, body []byte) []models.DocumentVersion {
	t.Helper()
	var wrap struct {
		Data []models.DocumentVersion `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse versions: %v", err)
	}
	return wrap.Data
}

func parseDocVersionFromBody(t *testing.T, body []byte) models.DocumentVersion {
	t.Helper()
	var wrap struct {
		Data models.DocumentVersion `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse version: %v", err)
	}
	return wrap.Data
}

func TestDocVersionCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	// Create a document
	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("POST", "/api/v1/docs", []byte(`{"title":"Test Doc","category":"spec","content":"Version A content","revision":"A"}`), cookie)
	h.CreateDoc(w, req)
	if w.Code != 200 {
		t.Fatalf("create doc: %d %s", w.Code, w.Body.String())
	}
	doc := parseDocFromBody(t, w.Body.Bytes())

	// Update document (should auto-snapshot)
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("PUT", "/api/v1/docs/"+doc.ID, []byte(`{"title":"Test Doc","category":"spec","content":"Version A updated","revision":"A"}`), cookie)
	h.UpdateDoc(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("update doc: %d %s", w.Code, w.Body.String())
	}

	// List versions
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions", nil, cookie)
	h.ListDocVersions(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("list versions: %d %s", w.Code, w.Body.String())
	}
	versions := parseDocVersionsFromBody(t, w.Body.Bytes())
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
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("POST", "/api/v1/docs", []byte(`{"title":"Release Test","category":"spec","content":"Draft content","revision":"A","status":"draft"}`), cookie)
	h.CreateDoc(w, req)
	doc := parseDocFromBody(t, w.Body.Bytes())

	// Release
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("POST", "/api/v1/docs/"+doc.ID+"/release", nil, cookie)
	h.ReleaseDoc(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("release: %d %s", w.Code, w.Body.String())
	}

	released := parseDocFromBody(t, w.Body.Bytes())
	if released.Status != "released" {
		t.Errorf("expected status released, got %s", released.Status)
	}

	// Verify version was created
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions", nil, cookie)
	h.ListDocVersions(w, req, doc.ID)
	versions := parseDocVersionsFromBody(t, w.Body.Bytes())
	if len(versions) != 1 {
		t.Fatalf("expected 1 version after release, got %d", len(versions))
	}
}

func TestDocRevert(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	// Create doc with content A
	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("POST", "/api/v1/docs", []byte(`{"title":"Revert Test","category":"spec","content":"Original content","revision":"A"}`), cookie)
	h.CreateDoc(w, req)
	doc := parseDocFromBody(t, w.Body.Bytes())

	// Update to content B (snapshots A)
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("PUT", "/api/v1/docs/"+doc.ID, []byte(`{"title":"Revert Test","category":"spec","content":"Updated content","revision":"B"}`), cookie)
	h.UpdateDoc(w, req, doc.ID)

	// Revert to A
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("POST", "/api/v1/docs/"+doc.ID+"/revert/A", nil, cookie)
	h.RevertDoc(w, req, doc.ID, "A")
	if w.Code != 200 {
		t.Fatalf("revert: %d %s", w.Code, w.Body.String())
	}

	reverted := parseDocFromBody(t, w.Body.Bytes())
	if reverted.Content != "Original content" {
		t.Errorf("expected original content after revert, got %s", reverted.Content)
	}
	if reverted.Revision != "A" {
		t.Errorf("expected revision A after revert, got %s", reverted.Revision)
	}
}

func TestDocDiff(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("POST", "/api/v1/docs", []byte(`{"title":"Diff Test","category":"spec","content":"Line 1\nLine 2\nLine 3","revision":"A"}`), cookie)
	h.CreateDoc(w, req)
	doc := parseDocFromBody(t, w.Body.Bytes())

	// Update (snapshots revision A)
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("PUT", "/api/v1/docs/"+doc.ID, []byte(`{"title":"Diff Test","category":"spec","content":"Line 1\nLine 2 modified\nLine 3\nLine 4","revision":"B"}`), cookie)
	h.UpdateDoc(w, req, doc.ID)

	// Snapshot revision B too
	h.SnapshotDocumentVersion(doc.ID, "Snapshot B", "admin", nil)

	// Get diff from A to B
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("GET", "/api/v1/docs/"+doc.ID+"/diff?from=A&to=B", nil, cookie)
	h.DocDiff(w, req, doc.ID)
	if w.Code != 200 {
		t.Fatalf("diff: %d %s", w.Code, w.Body.String())
	}

	var wrap struct {
		Data struct {
			From  string                `json:"from"`
			To    string                `json:"to"`
			Lines []engineering.DiffLine `json:"lines"`
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
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("GET", "/api/v1/docs/DOC-001/diff", nil, cookie)
	h.DocDiff(w, req, "DOC-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetDocVersionByRevision(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	w := httptest.NewRecorder()
	req := testutil.AuthedRequest("POST", "/api/v1/docs", []byte(`{"title":"Version Get Test","category":"spec","content":"Content A","revision":"A"}`), cookie)
	h.CreateDoc(w, req)
	doc := parseDocFromBody(t, w.Body.Bytes())

	// Update to create version snapshot
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("PUT", "/api/v1/docs/"+doc.ID, []byte(`{"title":"Version Get Test","category":"spec","content":"Content B","revision":"B"}`), cookie)
	h.UpdateDoc(w, req, doc.ID)

	// Get version A
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions/A", nil, cookie)
	h.GetDocVersion(w, req, doc.ID, "A")
	if w.Code != 200 {
		t.Fatalf("get version: %d %s", w.Code, w.Body.String())
	}
	v := parseDocVersionFromBody(t, w.Body.Bytes())
	if v.Content != "Content A" {
		t.Errorf("expected Content A, got %s", v.Content)
	}

	// Get nonexistent version
	w = httptest.NewRecorder()
	req = testutil.AuthedRequest("GET", "/api/v1/docs/"+doc.ID+"/versions/Z", nil, cookie)
	h.GetDocVersion(w, req, doc.ID, "Z")
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
		got := engineering.NextRevision(c.in)
		if got != c.out {
			t.Errorf("NextRevision(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestComputeDiff(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "x", "c", "d"}
	diff := engineering.ComputeDiff(from, to)

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
