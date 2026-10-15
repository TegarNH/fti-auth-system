package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// ReadJSON decodes request body with a 64KB limit to prevent resource exhaustion.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := int64(64 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must only contain a single JSON object")
	}

	return nil
}

// GetClientIP extracts the remote client IP without port number.
func GetClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}
	return strings.TrimSpace(remoteIP)
}
