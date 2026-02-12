package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return fmt.Errorf("cannot write JSON response: %w", err)
	}

	return nil
}

func WriteError(w http.ResponseWriter, code int, msg string) error {
	return WriteJSON(w, code, map[string]any{"error": msg})
}
