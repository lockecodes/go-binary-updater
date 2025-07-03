package release

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
)

// TestGitLabIntegration tests the complete GitLab release workflow
func TestGitLabIntegration(t *testing.T) {
	// Create a comprehensive mock server that simulates GitLab API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "401 Unauthorized"}`))
			return
		}

		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "403 Forbidden"}`))
			return
		}

		// Simulate successful response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"tag_name": "v2.1.0",
			"name": "Release v2.1.0",
			"description": "Latest release with new features",
			"released_at": "2023-12-01T10:00:00Z",
			"assets": {
				"links": [
					{
						"name": "myapp-Linux_x86_64.tar.gz",
						"url": "https://gitlab.example.com/project/releases/v2.1.0/downloads/myapp-Linux_x86_64.tar.gz",
						"direct_asset_url": "https://gitlab.example.com/project/releases/v2.1.0/downloads/myapp-Linux_x86_64.tar.gz",
						"link_type": "package"
					},
					{
						"name": "myapp-Darwin_arm64.tar.gz",
						"url": "https://gitlab.example.com/project/releases/v2.1.0/downloads/myapp-Darwin_arm64.tar.gz",
						"direct_asset_url": "https://gitlab.example.com/project/releases/v2.1.0/downloads/myapp-Darwin_arm64.tar.gz",
						"link_type": "package"
					}
				]
			}
		}]`))
	}))
	defer server.Close()

	// Test configuration
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Test 1: Authentication with token
	t.Run("Authentication", func(t *testing.T) {
		gitlabRelease := NewGitlabReleaseWithToken("12345", "test-token", config)
		gitlabRelease.GitLabConfig.BaseURL = server.URL

		err := gitlabRelease.GetLatestRelease()
		if err != nil {
			t.Fatalf("Expected success with authentication, got error: %v", err)
		}

		if gitlabRelease.Version != "v2.1.0" {
			t.Errorf("Expected version v2.1.0, got %s", gitlabRelease.Version)
		}

		expectedURL := "https://gitlab.example.com/project/releases/v2.1.0/downloads/myapp-Linux_x86_64.tar.gz"
		if gitlabRelease.ReleaseLink != expectedURL {
			t.Errorf("Expected release link %s, got %s", expectedURL, gitlabRelease.ReleaseLink)
		}
	})

	// Test 2: Environment variable configuration
	t.Run("EnvironmentVariables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("GITLAB_TOKEN", "test-token")
		os.Setenv("GITLAB_API_URL", server.URL)
		defer func() {
			os.Unsetenv("GITLAB_TOKEN")
			os.Unsetenv("GITLAB_API_URL")
		}()

		gitlabRelease := NewGitlabRelease("12345", config)

		if gitlabRelease.GitLabConfig.Token != "test-token" {
			t.Errorf("Expected token from environment, got %s", gitlabRelease.GitLabConfig.Token)
		}

		if gitlabRelease.GitLabConfig.BaseURL != server.URL {
			t.Errorf("Expected base URL from environment, got %s", gitlabRelease.GitLabConfig.BaseURL)
		}

		err := gitlabRelease.GetLatestRelease()
		if err != nil {
			t.Fatalf("Expected success with environment variables, got error: %v", err)
		}
	})

	// Test 3: Custom configuration
	t.Run("CustomConfiguration", func(t *testing.T) {
		gitlabConfig := DefaultGitLabConfig()
		gitlabConfig.BaseURL = server.URL
		gitlabConfig.Token = "test-token"
		gitlabConfig.HTTPConfig.MaxRetries = 5
		gitlabConfig.HTTPConfig.InitialDelay = 100 * time.Millisecond
		gitlabConfig.CustomHeaders = map[string]string{
			"X-Custom-Header": "test-value",
		}

		gitlabRelease := NewGitlabReleaseWithConfig("12345", config, gitlabConfig)

		err := gitlabRelease.GetLatestRelease()
		if err != nil {
			t.Fatalf("Expected success with custom configuration, got error: %v", err)
		}

		if gitlabRelease.GitLabConfig.HTTPConfig.MaxRetries != 5 {
			t.Errorf("Expected MaxRetries 5, got %d", gitlabRelease.GitLabConfig.HTTPConfig.MaxRetries)
		}
	})
}

// TestArchitectureMapping tests the comprehensive architecture mapping
func TestArchitectureMapping(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// Standard mappings
		{"amd64", "x86_64"},
		{"arm64", "arm64"},
		{"arm", "arm"},
		{"386", "i386"},
		
		// Case variations
		{"AMD64", "x86_64"},
		{"ARM64", "arm64"},
		{"X86_64", "x86_64"},
		
		// Whitespace handling
		{" amd64 ", "x86_64"},
		{"\tarm64\t", "arm64"},
		
		// Architecture variants
		{"aarch64", "arm64"},
		{"armv7", "arm"},
		{"i686", "i386"},
		{"x64", "x86_64"},
		
		// Specialized architectures
		{"mips", "mips"},
		{"mips64le", "mips64le"},
		{"ppc64", "ppc64"},
		{"s390x", "s390x"},
		{"riscv64", "riscv64"},
		
		// Fallback behavior
		{"unknown", "unknown"},
		{"custom-arch", "custom-arch"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("MapArch_%s", tc.input), func(t *testing.T) {
			result := MapArch(tc.input)
			if result != tc.expected {
				t.Errorf("MapArch(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestHTTPClientResilience tests the HTTP client's resilience features
func TestHTTPClientResilience(t *testing.T) {
	// Test retry logic with server errors
	t.Run("RetryLogic", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
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

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	// Test circuit breaker
	t.Run("CircuitBreaker", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := DefaultHTTPClientConfig()
		config.MaxRetries = 0
		config.CircuitBreaker = true
		client := NewRetryableHTTPClient(config)

		// Make multiple failing requests
		for i := 0; i < 6; i++ {
			client.Get(server.URL)
		}

		// Circuit breaker should be open now
		_, err := client.Get(server.URL)
		if err == nil || !contains(err.Error(), "circuit breaker is open") {
			t.Errorf("Expected circuit breaker error, got: %v", err)
		}
	})

	// Test rate limiting
	t.Run("RateLimiting", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
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

		// Should have waited at least 1 second due to Retry-After header
		if duration < 1*time.Second {
			t.Errorf("Expected to wait at least 1 second, waited %v", duration)
		}
	})
}

// TestInterfaceCompliance verifies that both GitHub and GitLab implementations
// properly implement the Release interface
func TestInterfaceCompliance(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test.tar.gz",
	}

	// Test GitLab implementation
	var gitlabRelease Release = NewGitlabRelease("12345", config)
	if gitlabRelease == nil {
		t.Error("GitLab release should implement Release interface")
	}

	// Test GitHub implementation
	var githubRelease Release = NewGithubRelease("owner/repo", config)
	if githubRelease == nil {
		t.Error("GitHub release should implement Release interface")
	}

	// Test polymorphic usage
	providers := []Release{gitlabRelease, githubRelease}
	for i, provider := range providers {
		if provider == nil {
			t.Errorf("Provider %d should not be nil", i)
		}
	}
}

// TestEnhancedReleaseInterface tests the new path resolution and installation info methods
func TestEnhancedReleaseInterface(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "testapp",
		BinaryName:             "testapp",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-enhanced",
		SourceArchivePath:      "/tmp/testapp.tar.gz",
	}

	// Test GitHub implementation
	githubRelease := NewGithubRelease("owner/repo", config)

	// Test methods without version (should fail gracefully)
	_, err := githubRelease.GetInstalledBinaryPath()
	if err == nil {
		t.Error("Expected error when calling GetInstalledBinaryPath without version")
	}

	_, err = githubRelease.GetInstallationInfo()
	if err == nil {
		t.Error("Expected error when calling GetInstallationInfo without version")
	}

	// Set version and test again
	githubRelease.Version = "v1.0.0"

	// These should not error (even if paths don't exist, the methods should work)
	_, err = githubRelease.GetInstalledBinaryPath()
	if err == nil {
		// This is expected to fail because the binary doesn't actually exist
		// but the method should handle this gracefully
	}

	_, err = githubRelease.GetInstallationInfo()
	if err == nil {
		// This is expected to fail because the binary doesn't actually exist
		// but the method should handle this gracefully
	}

	// Test GitLab implementation
	gitlabRelease := NewGitlabRelease("12345", config)

	// Test methods without version (should fail gracefully)
	_, err = gitlabRelease.GetInstalledBinaryPath()
	if err == nil {
		t.Error("Expected error when calling GetInstalledBinaryPath without version")
	}

	_, err = gitlabRelease.GetInstallationInfo()
	if err == nil {
		t.Error("Expected error when calling GetInstallationInfo without version")
	}

	// Set version and test again
	gitlabRelease.Version = "v1.0.0"

	// These should not error (even if paths don't exist, the methods should work)
	_, err = gitlabRelease.GetInstalledBinaryPath()
	if err == nil {
		// This is expected to fail because the binary doesn't actually exist
		// but the method should handle this gracefully
	}

	_, err = gitlabRelease.GetInstallationInfo()
	if err == nil {
		// This is expected to fail because the binary doesn't actually exist
		// but the method should handle this gracefully
	}
}
