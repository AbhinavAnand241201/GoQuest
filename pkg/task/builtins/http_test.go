package builtins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPTask(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test headers
		if r.Header.Get("Test-Header") != "test-value" {
			t.Errorf("Expected header Test-Header=test-value, got %s", r.Header.Get("Test-Header"))
		}

		// Test method
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Return test response
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Test successful request
	t.Run("Successful request", func(t *testing.T) {
		task := Get(server.URL).
			WithHeader("Test-Header", "test-value").
			WithTimeout(5 * time.Second)

		result, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != "test response" {
			t.Errorf("Expected result 'test response', got %v", result)
		}
	})

	// Test timeout
	t.Run("Timeout", func(t *testing.T) {
		// Create a slow server
		slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte("slow response"))
		}))
		defer slowServer.Close()

		task := Get(slowServer.URL).
			WithTimeout(50 * time.Millisecond)

		_, err := task.Run(context.Background())
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})

	// Test error response
	t.Run("Error response", func(t *testing.T) {
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error response"))
		}))
		defer errorServer.Close()

		task := Get(errorServer.URL)

		_, err := task.Run(context.Background())
		if err == nil {
			t.Error("Expected error for 500 status code, got nil")
		}
	})
}

func TestHTTPMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Method))
	}))
	defer server.Close()

	tests := []struct {
		name     string
		task     *HTTPTask
		expected string
	}{
		{
			name:     "GET",
			task:     Get(server.URL),
			expected: "GET",
		},
		{
			name:     "POST",
			task:     Post(server.URL),
			expected: "POST",
		},
		{
			name:     "PUT",
			task:     Put(server.URL),
			expected: "PUT",
		},
		{
			name:     "DELETE",
			task:     DeleteHTTP(server.URL),
			expected: "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.task.Run(context.Background())
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected method %s, got %v", tt.expected, result)
			}
		})
	}
}
