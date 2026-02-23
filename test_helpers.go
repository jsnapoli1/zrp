package main

import (
	"net/http/httptest"
	"testing"

	"zrp/internal/testutil"
)

// decodeEnvelope decodes an API response envelope and extracts the data.
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	testutil.DecodeEnvelope(t, w, v)
}
