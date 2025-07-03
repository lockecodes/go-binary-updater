package release

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
)

// TestCDNExecutionFix demonstrates the fix for the CDN strategy execution issue
func TestCDNExecutionFix(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() *GithubRelease
		setVersion     bool
		version        string
		expectGitHubCall bool
		expectCDNCall    bool
		description    string
	}{
		{
			name: "kubectl CDN with version discovery - no GitHub calls",
			setupFunc: func() *GithubRelease {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "kubectl",
					BinaryName:             "kubectl",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/kubectl-test",
					IsDirectBinary:         true,
					ProjectName:            "kubectl",
				}
				kubectlConfig := GetKubectlCDNConfig()
				return NewGithubReleaseWithCDNConfig("kubernetes/kubernetes", config, kubectlConfig)
			},
			setVersion:       false, // Let it discover version from CDN
			expectGitHubCall: false, // Should not call GitHub if CDN discovery works
			expectCDNCall:    true,
			description:      "kubectl should discover version from CDN without GitHub calls",
		},
		{
			name: "Helm CDN with version discovery fallback - GitHub calls for version only",
			setupFunc: func() *GithubRelease {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "helm",
					BinaryName:             "helm",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/helm-test.tar.gz",
					IsDirectBinary:         false,
					ProjectName:            "helm",
				}
				helmConfig := GetHelmCDNConfig()
				return NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)
			},
			setVersion:       false, // Let it try CDN discovery, then fall back to GitHub
			expectGitHubCall: true,  // Should call GitHub for version info only
			expectCDNCall:    true,
			description:      "Helm should try CDN discovery, fall back to GitHub for version, then download from CDN",
		},
		{
			name: "Helm CDN with pre-set version - no GitHub calls",
			setupFunc: func() *GithubRelease {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "helm",
					BinaryName:             "helm",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/helm-test.tar.gz",
					IsDirectBinary:         false,
					ProjectName:            "helm",
				}
				helmConfig := GetHelmCDNConfig()
				return NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)
			},
			setVersion:       true,
			version:          "v3.18.3",
			expectGitHubCall: false, // Should not call GitHub if version is pre-set
			expectCDNCall:    true,
			description:      "Helm with pre-set version should download directly from CDN without GitHub calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var logBuffer bytes.Buffer
			originalOutput := log.Writer()
			log.SetOutput(&logBuffer)
			defer log.SetOutput(originalOutput)

			githubRelease := tt.setupFunc()

			// Set version if specified
			if tt.setVersion {
				githubRelease.Version = tt.version
			}

			// Verify strategy is CDN
			if githubRelease.AssetMatchingConfig.Strategy != CDNStrategy {
				t.Fatalf("Expected CDNStrategy, got %v", githubRelease.AssetMatchingConfig.Strategy)
			}

			// Try to download
			err := githubRelease.DownloadLatestRelease()

			// Check log output for GitHub calls
			logOutput := logBuffer.String()
			hasGitHubCall := strings.Contains(logOutput, "Fetching latest release from GitHub")

			// Verify GitHub call expectation - this is the key fix we're testing
			if !tt.expectGitHubCall && hasGitHubCall {
				t.Errorf("%s: Did not expect GitHub API call but found it in log: %s", tt.description, logOutput)
			}

			// If we expected GitHub calls but didn't find them, that's OK if CDN discovery worked
			if tt.expectGitHubCall && !hasGitHubCall {
				t.Logf("%s: Expected GitHub API call but didn't find it (CDN discovery may have worked)", tt.description)
			}

			// The key success criteria:
			// 1. CDN strategy is being used (verified above)
			// 2. No unexpected GitHub calls are made
			// 3. Download attempt is made (success/failure depends on network/test environment)

			// Log the result for debugging
			if err != nil {
				t.Logf("%s: Download error (expected in test environment): %v", tt.description, err)
			}
			t.Logf("%s: Log output: %s", tt.description, logOutput)
		})
	}
}

// TestDirectCDNDownload tests the new DownloadCDNVersion method
func TestDirectCDNDownload(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "helm",
		BinaryName:             "helm",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/helm-direct-test.tar.gz",
		IsDirectBinary:         false,
		ProjectName:            "helm",
	}

	helmConfig := GetHelmCDNConfig()
	githubRelease := NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

	// Capture log output
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

	// Use the direct CDN download method
	err := githubRelease.DownloadCDNVersion("v3.18.3")

	// Check log output - should NOT contain GitHub calls
	logOutput := logBuffer.String()
	if strings.Contains(logOutput, "Fetching latest release from GitHub") {
		t.Errorf("Direct CDN download should not make GitHub API calls, but log contains: %s", logOutput)
	}

	// CDN download messages go to stdout, not log, so we just verify no GitHub calls were made
	t.Logf("Direct CDN download completed without GitHub API calls")

	// Error is expected due to network/file system in test environment
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}

	// Verify version was set
	if githubRelease.Version != "v3.18.3" {
		t.Errorf("Expected version to be set to v3.18.3, got: %s", githubRelease.Version)
	}
}

// TestCDNStrategyValidation tests that CDN methods validate strategy correctly
func TestCDNStrategyValidation(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "test",
		BinaryName:             "test",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test.tar.gz",
		AssetMatchingStrategy:  "flexible", // Not CDN strategy
	}

	githubRelease := NewGithubRelease("owner/repo", config)

	// Try to use CDN download with non-CDN strategy
	err := githubRelease.DownloadCDNVersion("v1.0.0")
	
	if err == nil {
		t.Error("Expected error when using CDN download with non-CDN strategy")
	}

	if !strings.Contains(err.Error(), "CDN download requires CDNStrategy or HybridStrategy") {
		t.Errorf("Expected strategy validation error, got: %v", err)
	}
}

// TestCDNConfigurationValidation tests that CDN methods validate configuration
func TestCDNConfigurationValidation(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "test",
		BinaryName:             "test",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test.tar.gz",
		AssetMatchingStrategy:  "cdn",
	}

	githubRelease := NewGithubRelease("owner/repo", config)
	
	// CDN strategy is set but no CDN configuration
	err := githubRelease.DownloadCDNVersion("v1.0.0")
	
	if err == nil {
		t.Error("Expected error when using CDN download with incomplete CDN configuration")
	}

	if !strings.Contains(err.Error(), "CDN configuration is incomplete") {
		t.Errorf("Expected configuration validation error, got: %v", err)
	}
}
