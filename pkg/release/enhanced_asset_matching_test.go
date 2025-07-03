package release

import (
	"testing"
)

func TestAssetFiltering_K0sExclusion(t *testing.T) {
	// Test k0s asset filtering to exclude airgap bundles
	assetNames := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-airgap-bundle-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64",
		"k0s-airgap-bundle-v1.33.2+k0s.0-arm64",
		"k0s-v1.33.2+k0s.0-amd64.exe",
	}

	config := GetK0sConfig()
	matcher := NewAssetMatcher(config)

	// Test filtering
	filtered := matcher.filterExcludedAssets(assetNames)
	
	// Should exclude airgap bundles
	expected := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64",
		"k0s-v1.33.2+k0s.0-amd64.exe",
	}

	if len(filtered) != len(expected) {
		t.Errorf("Expected %d filtered assets, got %d", len(expected), len(filtered))
	}

	for i, asset := range filtered {
		if asset != expected[i] {
			t.Errorf("Expected filtered asset %d to be %s, got %s", i, expected[i], asset)
		}
	}
}

func TestAssetFiltering_SignatureFiles(t *testing.T) {
	// Test filtering of signature and checksum files
	assetNames := []string{
		"helm-v3.12.0-linux-amd64.tar.gz",
		"helm-v3.12.0-linux-amd64.tar.gz.asc",
		"helm-v3.12.0-linux-amd64.tar.gz.sha256",
		"helm-v3.12.0-darwin-arm64.tar.gz",
		"helm-v3.12.0-darwin-arm64.tar.gz.asc",
	}

	config := DefaultAssetMatchingConfig()
	matcher := NewAssetMatcher(config)

	filtered := matcher.filterExcludedAssets(assetNames)
	
	// Should exclude signature and checksum files
	expected := []string{
		"helm-v3.12.0-linux-amd64.tar.gz",
		"helm-v3.12.0-darwin-arm64.tar.gz",
	}

	if len(filtered) != len(expected) {
		t.Errorf("Expected %d filtered assets, got %d", len(expected), len(filtered))
	}

	for i, asset := range filtered {
		if asset != expected[i] {
			t.Errorf("Expected filtered asset %d to be %s, got %s", i, expected[i], asset)
		}
	}
}

func TestPriorityPatterns_K0s(t *testing.T) {
	// Test priority patterns for k0s
	assetNames := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64",
		"some-other-binary-amd64",
	}

	config := GetK0sConfig()
	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux"

	bestMatch, err := matcher.FindBestMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected to find a match, got error: %v", err)
	}

	expected := "k0s-v1.33.2+k0s.0-amd64"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}
}

func TestCDNStrategy_Helm(t *testing.T) {
	// Test CDN strategy for Helm
	config := GetHelmCDNConfig()
	matcher := NewAssetMatcher(config)

	// CDN strategy should return a URL pattern
	cdnURL, err := matcher.findCDNMatch()
	if err != nil {
		t.Fatalf("Expected CDN match to succeed, got error: %v", err)
	}

	// Should contain the CDN base URL and pattern
	expectedBase := "https://get.helm.sh/"
	if !containsSubstring(cdnURL, expectedBase) {
		t.Errorf("Expected CDN URL to contain %s, got %s", expectedBase, cdnURL)
	}
}

func TestCDNStrategy_Kubectl(t *testing.T) {
	// Test CDN strategy for kubectl
	config := GetKubectlCDNConfig()
	matcher := NewAssetMatcher(config)

	cdnURL, err := matcher.findCDNMatch()
	if err != nil {
		t.Fatalf("Expected CDN match to succeed, got error: %v", err)
	}

	// Should contain the Google CDN base URL
	expectedBase := "https://dl.k8s.io/release/"
	if !containsSubstring(cdnURL, expectedBase) {
		t.Errorf("Expected CDN URL to contain %s, got %s", expectedBase, cdnURL)
	}
}

func TestHybridStrategy(t *testing.T) {
	// Test hybrid strategy that tries GitHub first, then CDN
	assetNames := []string{
		"terraform_1.5.0_linux_amd64.zip",
		"terraform_1.5.0_darwin_arm64.zip",
	}

	config := GetTerraformConfig()
	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux"

	// Should find GitHub asset first
	bestMatch, err := matcher.findHybridMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected hybrid match to succeed, got error: %v", err)
	}

	expected := "terraform_1.5.0_linux_amd64.zip"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}
}

func TestHybridStrategy_FallbackToCDN(t *testing.T) {
	// Test hybrid strategy fallback to CDN when no GitHub assets match
	assetNames := []string{
		"some-other-file.txt",
		"unrelated-binary.exe",
	}

	config := GetTerraformConfig()
	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux"

	// Should fall back to CDN
	bestMatch, err := matcher.findHybridMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected hybrid match to fall back to CDN, got error: %v", err)
	}

	// Should contain CDN base URL
	expectedBase := "https://releases.hashicorp.com/terraform/"
	if !containsSubstring(bestMatch, expectedBase) {
		t.Errorf("Expected CDN fallback URL to contain %s, got %s", expectedBase, bestMatch)
	}
}

func TestValidateCDNConfig(t *testing.T) {
	// Test CDN configuration validation
	testCases := []struct {
		name        string
		config      AssetMatchingConfig
		expectError bool
	}{
		{
			name: "Valid CDN config",
			config: AssetMatchingConfig{
				Strategy:    CDNStrategy,
				CDNBaseURL:  "https://example.com/",
				CDNPattern:  "binary-{version}-{os}-{arch}.tar.gz",
				IsDirectBinary: false,
			},
			expectError: false,
		},
		{
			name: "Missing CDN base URL",
			config: AssetMatchingConfig{
				Strategy:   CDNStrategy,
				CDNPattern: "binary-{version}-{os}-{arch}.tar.gz",
			},
			expectError: true,
		},
		{
			name: "Missing CDN pattern",
			config: AssetMatchingConfig{
				Strategy:   CDNStrategy,
				CDNBaseURL: "https://example.com/",
			},
			expectError: true,
		},
		{
			name: "Missing version placeholder",
			config: AssetMatchingConfig{
				Strategy:    CDNStrategy,
				CDNBaseURL:  "https://example.com/",
				CDNPattern:  "binary-{os}-{arch}.tar.gz",
				IsDirectBinary: false,
			},
			expectError: true,
		},
		{
			name: "Direct binary without OS/arch placeholders (valid)",
			config: AssetMatchingConfig{
				Strategy:       CDNStrategy,
				CDNBaseURL:     "https://example.com/",
				CDNPattern:     "binary-{version}",
				IsDirectBinary: true,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCDNConfig(tc.config)
			if tc.expectError && err == nil {
				t.Error("Expected validation error, got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestGetPresetConfig(t *testing.T) {
	// Test preset configurations for common binaries
	testCases := []struct {
		binaryName string
		expectError bool
	}{
		{"helm", false},
		{"kubectl", false},
		{"k0s", false},
		{"terraform", false},
		{"docker", false},
		{"unknown-binary", true},
	}

	for _, tc := range testCases {
		t.Run(tc.binaryName, func(t *testing.T) {
			config, err := GetPresetConfig(tc.binaryName)
			if tc.expectError && err == nil {
				t.Error("Expected error for unknown binary, got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tc.binaryName, err)
			}
			if !tc.expectError {
				// Validate that the config has reasonable defaults
				if config.ProjectName == "" {
					t.Errorf("Expected ProjectName to be set for %s", tc.binaryName)
				}
			}
		})
	}
}

func TestDockerConfig_ExclusionPatterns(t *testing.T) {
	// Test Docker configuration excludes unwanted packages
	assetNames := []string{
		"docker-20.10.17-linux-amd64.tgz",
		"docker-desktop-4.12.0-linux-amd64.deb",
		"docker-rootless-extras-20.10.17-linux-amd64.tgz",
		"docker-20.10.17-darwin-arm64.tgz",
	}

	config := GetDockerConfig()
	matcher := NewAssetMatcher(config)

	filtered := matcher.filterExcludedAssets(assetNames)
	
	// Should exclude desktop and rootless packages
	expected := []string{
		"docker-20.10.17-linux-amd64.tgz",
		"docker-20.10.17-darwin-arm64.tgz",
	}

	if len(filtered) != len(expected) {
		t.Errorf("Expected %d filtered assets, got %d", len(expected), len(filtered))
	}

	for i, asset := range filtered {
		if asset != expected[i] {
			t.Errorf("Expected filtered asset %d to be %s, got %s", i, expected[i], asset)
		}
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
