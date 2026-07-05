package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func gunzip(t *testing.T, r io.Reader) string {
	t.Helper()
	gz, err := gzip.NewReader(r)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to decompress body: %v", err)
	}
	return string(data)
}

func TestCompress(t *testing.T) {
	const payload = "hello hello hello hello hello compression"

	tests := []struct {
		name           string
		acceptEncoding string
		wantGzip       bool
	}{
		{
			name:           "gzip when client accepts gzip",
			acceptEncoding: "gzip",
			wantGzip:       true,
		},
		{
			name:           "gzip when client accepts multiple encodings",
			acceptEncoding: "deflate, gzip, br",
			wantGzip:       true,
		},
		{
			name:           "no gzip without Accept-Encoding",
			acceptEncoding: "",
			wantGzip:       false,
		},
		{
			name:           "no gzip for other encodings",
			acceptEncoding: "deflate",
			wantGzip:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Compress()(okHandler(payload))

			req := httptest.NewRequest(http.MethodGet, "/compress", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			if tt.wantGzip {
				if got := w.Header().Get("Content-Encoding"); got != "gzip" {
					t.Fatalf("expected Content-Encoding 'gzip', got %q", got)
				}
				if got := gunzip(t, w.Body); got != payload {
					t.Errorf("expected decompressed body %q, got %q", payload, got)
				}
			} else {
				if got := w.Header().Get("Content-Encoding"); got != "" {
					t.Fatalf("expected no Content-Encoding, got %q", got)
				}
				if got := w.Body.String(); got != payload {
					t.Errorf("expected body %q, got %q", payload, got)
				}
			}
		})
	}
}

func TestCompress_PreservesStatusCode(t *testing.T) {
	wrapped := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("teapot"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
	if got := gunzip(t, w.Body); got != "teapot" {
		t.Errorf("expected decompressed body 'teapot', got %q", got)
	}
}

func TestCompress_DeletesContentLength(t *testing.T) {
	wrapped := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handlers often set Content-Length for the uncompressed size; the
		// middleware must drop it because the compressed size differs.
		w.Header().Set("Content-Length", "6")
		_, _ = w.Write([]byte("sixsix"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/length", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Length"); got != "" {
		t.Errorf("expected Content-Length to be deleted, got %q", got)
	}
	if got := gunzip(t, w.Body); got != "sixsix" {
		t.Errorf("expected decompressed body 'sixsix', got %q", got)
	}
}

func TestCompress_LargeBodyIsActuallyCompressed(t *testing.T) {
	payload := strings.Repeat("goframe ", 1024)

	wrapped := Compress()(okHandler(payload))

	req := httptest.NewRequest(http.MethodGet, "/large", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Body.Len() >= len(payload) {
		t.Errorf("expected compressed size < %d, got %d", len(payload), w.Body.Len())
	}
	if got := gunzip(t, w.Body); got != payload {
		t.Error("decompressed body does not match original payload")
	}
}
