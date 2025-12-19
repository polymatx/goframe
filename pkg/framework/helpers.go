package framework

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	headerXForwardedFor   = "X-Forwarded-For"
	headerXForwardedProto = "X-Forwarded-Proto"
	headerXRealIP         = "X-Real-IP"
	headerCFConnectingIP  = "CF-Connecting-IP"
	headerContentType     = "Content-Type"
	jsonMIME              = "application/json;charset=UTF-8"

	// HTTP scheme
	HTTP string = "http"
	// HTTPS scheme
	HTTPS string = "https"
)

// ErrorResponse represents a simple error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// RealIP extracts the real IP address from the request
// Checks various headers in order: CF-Connecting-IP, X-Forwarded-For, X-Real-IP, RemoteAddr
func RealIP(r *http.Request) string {
	ra := r.RemoteAddr

	if ip := r.Header.Get(headerCFConnectingIP); ip != "" {
		return ip
	}

	if ip := r.Header.Get(headerXForwardedFor); ip != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	if ip := r.Header.Get(headerXRealIP); ip != "" {
		return ip
	}

	// Extract IP from RemoteAddr (may include port)
	ip, _, _ := net.SplitHostPort(ra)
	if ip != "" {
		return ip
	}

	return ra
}

// Scheme extracts the scheme (http/https) from the request
func Scheme(r *http.Request) string {
	if r.TLS != nil {
		return HTTPS
	}

	if proto := strings.ToLower(r.Header.Get(headerXForwardedProto)); proto == HTTPS {
		return HTTPS
	}

	return HTTP
}

// Redirect performs an HTTP redirect
func Redirect(w http.ResponseWriter, code int, targetURL *url.URL) error {
	if code < http.StatusMultipleChoices || code > http.StatusPermanentRedirect {
		return errors.New("invalid redirect code")
	}

	w.Header().Set("Location", targetURL.String())
	w.WriteHeader(code)
	return nil
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, code int, data interface{}) error {
	w.Header().Set(headerContentType, jsonMIME)
	w.WriteHeader(code)

	return json.NewEncoder(w).Encode(data)
}

// JSONErr writes a JSON error response
func JSONErr(w http.ResponseWriter, code int, err error) error {
	w.Header().Set(headerContentType, jsonMIME)
	w.WriteHeader(code)

	if err != nil {
		return json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
	}

	return json.NewEncoder(w).Encode(ErrorResponse{Error: "unknown error"})
}

// JSONMessage writes a JSON response with a simple message
func JSONMessage(w http.ResponseWriter, code int, message string) error {
	return JSON(w, code, map[string]string{"message": message})
}

// NoContent writes a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// DecodeJSON decodes JSON from the request body into the provided value
func DecodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
