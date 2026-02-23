package response

import (
	"encoding/json"
	"net/http"

	"zrp/internal/models"
)

// JSON writes a successful API response with the given data.
func JSON(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(models.APIResponse{Data: data})
}

// JSONMeta writes a successful API response with pagination metadata.
func JSONMeta(w http.ResponseWriter, data interface{}, total, page, limit int) {
	json.NewEncoder(w).Encode(models.APIResponse{
		Data: data,
		Meta: &models.Meta{Total: total, Page: page, Limit: limit},
	})
}

// Err writes a JSON error response with the given message and HTTP status code.
func Err(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// DecodeBody decodes a JSON request body into the given value.
func DecodeBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
