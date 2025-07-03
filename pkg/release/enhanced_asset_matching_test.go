package release

import (
	"fmt"
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

func TestFormatVersionForCDN(t *testing.T) {
	// Test version formatting for different CDN requirements
	testCases := []struct {
		version  string
		format   string
		expected string
	}{
		{"3.18.3", "with-v", "v3.18.3"},
		{"v3.18.3", "with-v", "v3.18.3"},
		{"v3.18.3", "without-v", "3.18.3"},
		{"3.18.3", "without-v", "3.18.3"},
		{"v3.18.3", "as-is", "v3.18.3"},
		{"3.18.3", "as-is", "3.18.3"},
		{"1.5.0", "unknown-format", "1.5.0"}, // Should default to as-is
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.version, tc.format), func(t *testing.T) {
			result := FormatVersionForCDN(tc.version, tc.format)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestHelmCDNVersionFormat(t *testing.T) {
	// Test that Helm CDN configuration uses correct version format
	config := GetHelmCDNConfig()

	if config.CDNVersionFormat != "with-v" {
		t.Errorf("Expected Helm CDN version format to be 'with-v', got '%s'", config.CDNVersionFormat)
	}

	// Test URL construction with architecture mapping
	downloader := NewCDNDownloaderWithArchMapping(config.CDNBaseURL, config.CDNPattern, config.CDNArchMapping)
	url := downloader.ConstructURLWithVersionFormat("3.18.3", "linux", "amd64", config.CDNVersionFormat)

	expected := "https://get.helm.sh/helm-v3.18.3-linux-amd64.tar.gz"
	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestHelmCDNArchitectureMapping(t *testing.T) {
	// Test that Helm CDN uses correct architecture format (amd64, not x86_64)
	config := GetHelmCDNConfig()
	downloader := NewCDNDownloaderWithArchMapping(config.CDNBaseURL, config.CDNPattern, config.CDNArchMapping)

	testCases := []struct {
		inputArch    string
		expectedArch string
	}{
		{"amd64", "amd64"},     // Should preserve amd64 for Helm
		{"x86_64", "amd64"},    // Should convert x86_64 to amd64 for Helm
		{"x64", "amd64"},       // Should convert x64 to amd64 for Helm
		{"arm64", "arm64"},     // Should preserve arm64
		{"aarch64", "arm64"},   // Should convert aarch64 to arm64
		{"arm", "arm"},         // Should preserve arm
		{"386", "386"},         // Should preserve 386
		{"i386", "386"},        // Should convert i386 to 386
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Helm_arch_%s", tc.inputArch), func(t *testing.T) {
			mappedArch := downloader.mapArchForCDN(tc.inputArch)
			if mappedArch != tc.expectedArch {
				t.Errorf("Helm CDN mapArchForCDN(%s) = %s, want %s", tc.inputArch, mappedArch, tc.expectedArch)
			}

			// Test URL construction with the mapped architecture
			url := downloader.ConstructURLWithVersionFormat("v3.18.3", "linux", tc.inputArch, "with-v")
			expectedURL := fmt.Sprintf("https://get.helm.sh/helm-v3.18.3-linux-%s.tar.gz", tc.expectedArch)
			if url != expectedURL {
				t.Errorf("Expected Helm URL %s, got %s", expectedURL, url)
			}
		})
	}
}

func TestKubectlCDNArchitectureMapping(t *testing.T) {
	// Test that kubectl CDN uses correct architecture format (amd64, not x86_64)
	config := GetKubectlCDNConfig()
	downloader := NewCDNDownloaderWithArchMapping(config.CDNBaseURL, config.CDNPattern, config.CDNArchMapping)

	testCases := []struct {
		inputArch    string
		expectedArch string
	}{
		{"amd64", "amd64"},     // Should preserve amd64 for kubectl
		{"x86_64", "amd64"},    // Should convert x86_64 to amd64 for kubectl
		{"x64", "amd64"},       // Should convert x64 to amd64 for kubectl
		{"arm64", "arm64"},     // Should preserve arm64
		{"aarch64", "arm64"},   // Should convert aarch64 to arm64
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("kubectl_arch_%s", tc.inputArch), func(t *testing.T) {
			mappedArch := downloader.mapArchForCDN(tc.inputArch)
			if mappedArch != tc.expectedArch {
				t.Errorf("kubectl CDN mapArchForCDN(%s) = %s, want %s", tc.inputArch, mappedArch, tc.expectedArch)
			}

			// Test URL construction with the mapped architecture
			url := downloader.ConstructURLWithVersionFormat("v1.28.0", "linux", tc.inputArch, "as-is")
			expectedURL := fmt.Sprintf("https://dl.k8s.io/release/v1.28.0/bin/linux/%s/kubectl", tc.expectedArch)
			if url != expectedURL {
				t.Errorf("Expected kubectl URL %s, got %s", expectedURL, url)
			}
		})
	}
}

func TestOtherCDNArchitectureMapping(t *testing.T) {
	// Test that other CDNs use standard MapArch function
	downloader := NewCDNDownloader("https://releases.hashicorp.com/terraform/", "{version}/terraform_{version}_{os}_{arch}.zip")

	// For non-Helm/kubectl CDNs, should use standard MapArch which converts amd64 to x86_64
	mappedArch := downloader.mapArchForCDN("amd64")
	expectedArch := "x86_64" // Standard MapArch converts amd64 to x86_64

	if mappedArch != expectedArch {
		t.Errorf("Other CDN mapArchForCDN(amd64) = %s, want %s", mappedArch, expectedArch)
	}
}

func TestRealWorldHelmCDNURL(t *testing.T) {
	// Test the real-world scenario: runtime.GOARCH = "amd64" should generate correct Helm URL
	config := GetHelmCDNConfig()
	downloader := NewCDNDownloaderWithArchMapping(config.CDNBaseURL, config.CDNPattern, config.CDNArchMapping)

	// Simulate the real download process with runtime.GOARCH = "amd64"
	runtimeArch := "amd64"  // This is what runtime.GOARCH returns on amd64 systems

	// Test that the CDN downloader correctly maps amd64 to amd64 for Helm (not x86_64)
	mappedArch := downloader.mapArchForCDN(runtimeArch)
	if mappedArch != "amd64" {
		t.Errorf("Helm CDN should preserve amd64 architecture, got %s", mappedArch)
	}

	// Test the complete URL construction process
	url := downloader.ConstructURLWithVersionFormat("3.18.3", "linux", runtimeArch, config.CDNVersionFormat)
	expectedURL := "https://get.helm.sh/helm-v3.18.3-linux-amd64.tar.gz"

	if url != expectedURL {
		t.Errorf("Real-world Helm CDN URL construction failed")
		t.Errorf("Expected: %s", expectedURL)
		t.Errorf("Got:      %s", url)
		t.Errorf("This means the architecture mapping fix is not working correctly")
	}

	// Verify this matches the actual Helm CDN format
	t.Logf("✅ Helm CDN URL correctly generated: %s", url)
	t.Logf("✅ Architecture correctly mapped: %s -> %s", runtimeArch, mappedArch)
}

func TestConfigurableArchitectureMapping(t *testing.T) {
	// Test that custom architecture mapping can be configured for any CDN
	customArchMapping := map[string]string{
		"amd64":  "x86_64",  // Custom mapping: amd64 -> x86_64
		"arm64":  "aarch64", // Custom mapping: arm64 -> aarch64
		"386":    "i386",    // Custom mapping: 386 -> i386
	}

	downloader := NewCDNDownloaderWithArchMapping(
		"https://custom-cdn.example.com/",
		"binary-{version}-{os}-{arch}.tar.gz",
		customArchMapping,
	)

	testCases := []struct {
		inputArch    string
		expectedArch string
	}{
		{"amd64", "x86_64"},   // Should use custom mapping
		{"arm64", "aarch64"},  // Should use custom mapping
		{"386", "i386"},       // Should use custom mapping
		{"unknown", "unknown"}, // Should fall back to MapArch (which returns input for unknown)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("custom_arch_%s", tc.inputArch), func(t *testing.T) {
			mappedArch := downloader.mapArchForCDN(tc.inputArch)
			if mappedArch != tc.expectedArch {
				t.Errorf("Custom CDN mapArchForCDN(%s) = %s, want %s", tc.inputArch, mappedArch, tc.expectedArch)
			}
		})
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
				Strategy:         CDNStrategy,
				CDNBaseURL:       "https://example.com/",
				CDNPattern:       "binary-{version}-{os}-{arch}.tar.gz",
				CDNVersionFormat: "with-v",
				IsDirectBinary:   false,
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
		{
			name: "Invalid version format",
			config: AssetMatchingConfig{
				Strategy:         CDNStrategy,
				CDNBaseURL:       "https://example.com/",
				CDNPattern:       "binary-{version}",
				CDNVersionFormat: "invalid-format",
				IsDirectBinary:   true,
			},
			expectError: true,
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
