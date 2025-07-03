package release

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

// HTTPClientConfig holds configuration for the HTTP client with retry logic
type HTTPClientConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Exponential backoff multiplier
	Timeout         time.Duration // Request timeout
	RateLimitDelay  time.Duration // Additional delay for rate limiting
	CircuitBreaker  bool          // Enable circuit breaker pattern
}

// DefaultHTTPClientConfig returns a sensible default configuration
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		Timeout:         30 * time.Second,
		RateLimitDelay:  1 * time.Second,
		CircuitBreaker:  true,
	}
}

// RetryableHTTPClient provides HTTP client with retry logic and rate limiting
type RetryableHTTPClient struct {
	client         *http.Client
	config         HTTPClientConfig
	failureCount   int
	lastFailure    time.Time
	circuitOpen    bool
	circuitTimeout time.Duration
}

// NewRetryableHTTPClient creates a new HTTP client with retry capabilities
func NewRetryableHTTPClient(config HTTPClientConfig) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config:         config,
		circuitTimeout: 60 * time.Second, // Circuit breaker timeout
	}
}

// Do executes an HTTP request with retry logic and rate limiting
func (c *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Check circuit breaker
	if c.config.CircuitBreaker && c.isCircuitOpen() {
		return nil, fmt.Errorf("circuit breaker is open, too many recent failures")
	}

	var lastErr error
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Add context with timeout for each attempt
		ctx, cancel := context.WithTimeout(req.Context(), c.config.Timeout)
		reqWithContext := req.WithContext(ctx)
		
		resp, err := c.client.Do(reqWithContext)
		cancel()
		
		if err == nil {
			// Check for rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				c.handleRateLimit(resp, attempt)
				resp.Body.Close()
				c.recordFailure()
				if attempt < c.config.MaxRetries {
					continue
				}
				return nil, fmt.Errorf("rate limited after %d attempts", c.config.MaxRetries+1)
			}

			// Check for server errors that should be retried
			if c.shouldRetry(resp.StatusCode) {
				resp.Body.Close()
				c.recordFailure()
				if attempt < c.config.MaxRetries {
					c.waitBeforeRetry(attempt)
					continue
				}
				return nil, fmt.Errorf("server error %d after %d attempts", resp.StatusCode, c.config.MaxRetries+1)
			}

			// Success - reset failure count and circuit breaker
			c.resetCircuitBreaker()
			return resp, nil
		}
		
		lastErr = err
		c.recordFailure()
		
		// Don't wait after the last attempt
		if attempt < c.config.MaxRetries {
			c.waitBeforeRetry(attempt)
		}
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// shouldRetry determines if a request should be retried based on status code
func (c *RetryableHTTPClient) shouldRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,     // 429
		 http.StatusInternalServerError,  // 500
		 http.StatusBadGateway,          // 502
		 http.StatusServiceUnavailable,  // 503
		 http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// handleRateLimit handles rate limiting responses
func (c *RetryableHTTPClient) handleRateLimit(resp *http.Response, attempt int) {
	// Check for Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			delay := time.Duration(seconds) * time.Second
			// Cap the delay to prevent excessive waiting
			if delay > c.config.MaxDelay {
				delay = c.config.MaxDelay
			}
			time.Sleep(delay)
			return
		}
	}
	
	// Fallback to configured rate limit delay with exponential backoff
	delay := c.config.RateLimitDelay * time.Duration(math.Pow(c.config.BackoffFactor, float64(attempt)))
	if delay > c.config.MaxDelay {
		delay = c.config.MaxDelay
	}
	time.Sleep(delay)
}

// waitBeforeRetry implements exponential backoff
func (c *RetryableHTTPClient) waitBeforeRetry(attempt int) {
	delay := time.Duration(float64(c.config.InitialDelay) * math.Pow(c.config.BackoffFactor, float64(attempt)))
	if delay > c.config.MaxDelay {
		delay = c.config.MaxDelay
	}
	time.Sleep(delay)
}

// recordFailure records a failure for circuit breaker logic
func (c *RetryableHTTPClient) recordFailure() {
	c.failureCount++
	c.lastFailure = time.Now()
	
	// Open circuit breaker after 5 consecutive failures
	if c.config.CircuitBreaker && c.failureCount >= 5 {
		c.circuitOpen = true
	}
}

// resetCircuitBreaker resets the circuit breaker state
func (c *RetryableHTTPClient) resetCircuitBreaker() {
	c.failureCount = 0
	c.circuitOpen = false
}

// isCircuitOpen checks if the circuit breaker is open
func (c *RetryableHTTPClient) isCircuitOpen() bool {
	if !c.circuitOpen {
		return false
	}
	
	// Check if circuit breaker timeout has passed
	if time.Since(c.lastFailure) > c.circuitTimeout {
		c.circuitOpen = false
		c.failureCount = 0
		return false
	}
	
	return true
}

// Get is a convenience method for GET requests
func (c *RetryableHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// GetWithHeaders is a convenience method for GET requests with custom headers
func (c *RetryableHTTPClient) GetWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	return c.Do(req)
}

// ReadResponseBody safely reads and closes the response body
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
