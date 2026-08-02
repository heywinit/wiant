package httpapi

import (
	"encoding/json"
	"net/http"
)

const maxJSONBodyBytes = 1 << 20

// Error is the common error envelope returned by the HTTP API.
type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// DecodeJSON decodes one bounded JSON object and rejects unknown fields.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Request body is invalid", nil)
		return false
	}
	return true
}

// WriteJSON serializes a response using the API's JSON content type.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError serializes the common API error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	WriteJSON(w, status, Error{Code: code, Message: message, Fields: fields})
}
