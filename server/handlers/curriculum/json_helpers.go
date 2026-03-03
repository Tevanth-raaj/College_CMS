package curriculum

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes an error response as JSON instead of plain text.
// This prevents "Unexpected token" errors on the frontend when response.json() is called.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
