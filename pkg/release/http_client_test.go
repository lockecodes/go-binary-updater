package release

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryableHTTPClient_Success(t *testing.T) {
	// Create a test server that responds successfully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	config.MaxRetries = 2
	client := NewRetryableHTTPClient(config)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRetryableHTTPClient_RetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	config.MaxRetries = 3
	config.InitialDelay = 10 * time.Millisecond
	client := NewRetryableHTTPClient(config)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryableHTTPClient_RateLimitHandling(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	config.MaxRetries = 2
	config.RateLimitDelay = 10 * time.Millisecond
	client := NewRetryableHTTPClient(config)

	start := time.Now()
	resp, err := client.Get(server.URL)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected success after rate limit, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should have waited at least 1 second due to Retry-After header
	if duration < 1*time.Second {
		t.Errorf("Expected to wait at least 1 second, waited %v", duration)
	}
}

func TestRetryableHTTPClient_CircuitBreaker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	config.MaxRetries = 0  // No retries to fail faster
	config.InitialDelay = 1 * time.Millisecond
	config.CircuitBreaker = true
	client := NewRetryableHTTPClient(config)

	// Make multiple failing requests to trigger circuit breaker
	for i := 0; i < 6; i++ {
		_, err := client.Get(server.URL)
		if err == nil {
			t.Errorf("Expected error on attempt %d", i+1)
		}
	}

	// Next request should fail immediately due to circuit breaker
	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("Expected circuit breaker error, got success")
	}

	if err != nil && !contains(err.Error(), "circuit breaker is open") {
		t.Errorf("Expected circuit breaker error, got: %v", err)
	}
}

func TestRetryableHTTPClient_GetWithHeaders(t *testing.T) {
	expectedHeaders := map[string]string{
		"Authorization": "Bearer token123",
		"Accept":        "application/json",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, expectedValue := range expectedHeaders {
			if r.Header.Get(key) != expectedValue {
				t.Errorf("Expected header %s: %s, got: %s", key, expectedValue, r.Header.Get(key))
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	client := NewRetryableHTTPClient(config)

	resp, err := client.GetWithHeaders(server.URL, expectedHeaders)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	defer resp.Body.Close()
}

func TestRetryableHTTPClient_MaxRetriesExceeded(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	config.MaxRetries = 2
	config.InitialDelay = 10 * time.Millisecond
	config.CircuitBreaker = false  // Disable circuit breaker for this test
	client := NewRetryableHTTPClient(config)

	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}

	expectedAttempts := config.MaxRetries + 1 // Initial attempt + retries
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
	}
}

func TestRetryableHTTPClient_ShouldRetry(t *testing.T) {
	config := DefaultHTTPClientConfig()
	client := NewRetryableHTTPClient(config)

	retryableCodes := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	for _, code := range retryableCodes {
		if !client.shouldRetry(code) {
			t.Errorf("Expected status code %d to be retryable", code)
		}
	}

	nonRetryableCodes := []int{
		http.StatusOK,
		http.StatusNotFound,
		http.StatusForbidden,
		http.StatusUnauthorized,
		http.StatusBadRequest,
	}

	for _, code := range nonRetryableCodes {
		if client.shouldRetry(code) {
			t.Errorf("Expected status code %d to not be retryable", code)
		}
	}
}

func TestDefaultHTTPClientConfig(t *testing.T) {
	config := DefaultHTTPClientConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay 1s, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay 30s, got %v", config.MaxDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", config.BackoffFactor)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", config.Timeout)
	}

	if !config.CircuitBreaker {
		t.Error("Expected CircuitBreaker to be enabled")
	}
}

func TestReadResponseBody(t *testing.T) {
	expectedBody := "test response body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	body, err := ReadResponseBody(resp)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, string(body))
	}
}

func BenchmarkRetryableHTTPClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig()
	client := NewRetryableHTTPClient(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}


