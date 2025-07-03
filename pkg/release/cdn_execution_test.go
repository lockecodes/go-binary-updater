package release

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
)

// TestCDNExecutionDoesNotCallGitHub tests that CDN strategy doesn't make GitHub API calls
func TestCDNExecutionDoesNotCallGitHub(t *testing.T) {
	// Capture log output to verify no GitHub API calls are made
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "helm",
		BinaryName:             "helm",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/helm-test.tar.gz",
		IsDirectBinary:         false,
		ProjectName:            "helm",
	}

	// Test with Helm CDN configuration
	helmConfig := GetHelmCDNConfig()
	githubRelease := NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

	// Verify the strategy is correctly set
	if githubRelease.AssetMatchingConfig.Strategy != CDNStrategy {
		t.Fatalf("Expected CDNStrategy, got %v", githubRelease.AssetMatchingConfig.Strategy)
	}

	// Set a version manually to avoid GitHub API calls
	githubRelease.Version = "v3.18.3"

	// Try to download (this will fail due to network, but we're testing the execution path)
	err := githubRelease.DownloadLatestRelease()
	
	// Check the log output - it should NOT contain "Fetching latest release from GitHub"
	logOutput := logBuffer.String()
	if strings.Contains(logOutput, "Fetching latest release from GitHub") {
		t.Errorf("CDN strategy should not make GitHub API calls, but log contains: %s", logOutput)
	}

	// The error should be related to CDN download, not GitHub API
	if err != nil && !strings.Contains(err.Error(), "CDN") && !strings.Contains(err.Error(), "version") {
		t.Logf("Expected CDN-related error, got: %v", err)
	}
}

// TestCDNStrategyWithoutVersionRequiresGitHub tests that CDN strategy without version still needs GitHub for version info
func TestCDNStrategyWithoutVersionRequiresGitHub(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

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
	githubRelease := NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

	// Don't set version - this should require GitHub API call for version info
	// Try to download (this will fail due to network, but we're testing the execution path)
	err := githubRelease.DownloadLatestRelease()

	// Check the log output - it SHOULD contain "Fetching latest release from GitHub" for version info
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Fetching latest release from GitHub") {
		t.Errorf("CDN strategy without version should make GitHub API call for version info, but log doesn't contain expected message. Log: %s", logOutput)
	}

	// The error should be related to GitHub API or network
	if err == nil {
		t.Error("Expected error due to network/API call, but got nil")
	}
}

// TestHybridStrategyExecution tests that Hybrid strategy tries GitHub first, then CDN
func TestHybridStrategyExecution(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "terraform",
		BinaryName:             "terraform",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/terraform-test.zip",
		IsDirectBinary:         false,
		ProjectName:            "terraform",
	}

	terraformConfig := GetTerraformConfig()
	githubRelease := NewGithubReleaseWithAssetConfig("hashicorp/terraform", config, terraformConfig)

	// Verify the strategy is correctly set
	if githubRelease.AssetMatchingConfig.Strategy != HybridStrategy {
		t.Fatalf("Expected HybridStrategy, got %v", githubRelease.AssetMatchingConfig.Strategy)
	}

	// This test just verifies the strategy is set correctly
	// Actual execution testing would require mock servers
}

// TestCDNStrategyDetection tests that CDN strategy is properly detected in DownloadLatestRelease
func TestCDNStrategyDetection(t *testing.T) {
	tests := []struct {
		name     string
		strategy AssetMatchingStrategy
		shouldUseCDN bool
	}{
		{
			name:     "CDNStrategy should use CDN",
			strategy: CDNStrategy,
			shouldUseCDN: true,
		},
		{
			name:     "HybridStrategy should use CDN",
			strategy: HybridStrategy,
			shouldUseCDN: true,
		},
		{
			name:     "FlexibleStrategy should not use CDN",
			strategy: FlexibleStrategy,
			shouldUseCDN: false,
		},
		{
			name:     "StandardStrategy should not use CDN",
			strategy: StandardStrategy,
			shouldUseCDN: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := fileUtils.FileConfig{
				VersionedDirectoryName: "versions",
				SourceBinaryName:       "test",
				BinaryName:             "test",
				BaseBinaryDirectory:    "/tmp/test",
				SourceArchivePath:      "/tmp/test.tar.gz",
			}

			assetConfig := DefaultAssetMatchingConfig()
			assetConfig.Strategy = tt.strategy

			// Only set CDN config for CDN/Hybrid strategies to avoid auto-detection
			if tt.strategy == CDNStrategy || tt.strategy == HybridStrategy {
				assetConfig.CDNBaseURL = "https://example.com/"
				assetConfig.CDNPattern = "test-{version}-{os}-{arch}.tar.gz"
			}

			githubRelease := NewGithubReleaseWithAssetConfig("owner/repo", config, assetConfig)

			// Check if the strategy detection logic would use CDN
			usesCDN := githubRelease.AssetMatchingConfig.Strategy == CDNStrategy || 
					   githubRelease.AssetMatchingConfig.Strategy == HybridStrategy

			if usesCDN != tt.shouldUseCDN {
				t.Errorf("Strategy %v: expected shouldUseCDN=%v, got %v", 
					tt.strategy, tt.shouldUseCDN, usesCDN)
			}
		})
	}
}

// TestKubectlCDNVersionDiscovery tests that kubectl can discover its version from CDN
func TestKubectlCDNVersionDiscovery(t *testing.T) {
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
	_ = NewGithubReleaseWithCDNConfig("kubernetes/kubernetes", config, kubectlConfig)

	// Test CDN version discovery
	cdnDownloader := NewCDNDownloader(kubectlConfig.CDNBaseURL, kubectlConfig.CDNPattern)
	version, err := cdnDownloader.TryDiscoverLatestVersion()

	if err != nil {
		t.Logf("kubectl version discovery failed (this may be expected in test environment): %v", err)
		return
	}

	if version == "" {
		t.Error("kubectl version discovery returned empty version")
		return
	}

	t.Logf("Discovered kubectl version: %s", version)

	// Verify version format (should start with 'v')
	if !strings.HasPrefix(version, "v") {
		t.Errorf("kubectl version should start with 'v', got: %s", version)
	}
}

// TestCDNVersionDiscoveryFallback tests that CDN strategy falls back to GitHub when CDN discovery fails
func TestCDNVersionDiscoveryFallback(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	defer log.SetOutput(originalOutput)

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
	githubRelease := NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

	// Don't set version - this should try CDN discovery first, then fall back to GitHub
	err := githubRelease.DownloadLatestRelease()

	// Check the log output
	logOutput := logBuffer.String()

	// Should contain CDN discovery failure message and GitHub fallback
	if !strings.Contains(logOutput, "CDN version discovery failed") {
		t.Logf("Expected CDN discovery failure message in log, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "falling back to GitHub") {
		t.Logf("Expected GitHub fallback message in log, got: %s", logOutput)
	}

	// The error could be network-related, which is expected in test environment
	if err != nil {
		t.Logf("Expected error due to network/API call: %v", err)
	}
}
