package builtins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPTask represents a task that performs an HTTP request
type HTTPTask struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Timeout time.Duration
}

// NewHTTPTask creates a new HTTP task with default timeout
func NewHTTPTask(method, url string) *HTTPTask {
	return &HTTPTask{
		Method:  method,
		URL:     url,
		Headers: make(map[string]string),
		Timeout: 30 * time.Second,
	}
}

// WithHeader adds a header to the HTTP request
func (h *HTTPTask) WithHeader(key, value string) *HTTPTask {
	h.Headers[key] = value
	return h
}

// WithBody sets the request body
func (h *HTTPTask) WithBody(body string) *HTTPTask {
	h.Body = body
	return h
}

// WithTimeout sets the request timeout
func (h *HTTPTask) WithTimeout(timeout time.Duration) *HTTPTask {
	h.Timeout = timeout
	return h
}

// Run executes the HTTP request and returns the response body or status code
func (h *HTTPTask) Run(ctx context.Context) (interface{}, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, h.Method, h.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: h.Timeout,
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Return error for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// Get creates a new HTTP GET task
func Get(url string) *HTTPTask {
	return NewHTTPTask(http.MethodGet, url)
}

// Post creates a new HTTP POST task
func Post(url string) *HTTPTask {
	return NewHTTPTask(http.MethodPost, url)
}

// Put creates a new HTTP PUT task
func Put(url string) *HTTPTask {
	return NewHTTPTask(http.MethodPut, url)
}

// DeleteHTTP creates a new HTTP DELETE task
func DeleteHTTP(url string) *HTTPTask {
	return NewHTTPTask(http.MethodDelete, url)
}
