package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse представляет ошибку в формате API
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// writeJSON пишет JSON-ответ
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Логируем ошибку, но ответ уже не изменить
		return
	}
}

// writeError пишет ответ с ошибкой
func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// writeErrorWithDetails пишет ответ с ошибкой и дополнительными деталями
func writeErrorWithDetails(w http.ResponseWriter, status int, message, code string, details map[string]interface{}) {
	writeJSON(w, status, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// decodeJSON декодирует JSON из тела запроса
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
