package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/polymatx/goframe/pkg/array"
)

// httpClient is a configured HTTP client for external API calls
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Call helper for api calls
func Call(ctx context.Context, method, url string, headers map[string]string, timeout time.Duration, pl interface{}, cookies []*http.Cookie) ([]byte, http.Header, int, error) {
	var b io.Reader
	method = strings.ToUpper(method)

	if !array.StringInArray(method, "GET", "DELETE") && pl != nil {
		d, err := json.Marshal(pl)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to marshal payload: %w", err)
		}
		b = bytes.NewReader(d)
	}

	r, err := http.NewRequest(method, url, b)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		r.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}

	nCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := httpClient.Do(r.WithContext(nCtx))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, resp.Header, resp.StatusCode, nil
}
