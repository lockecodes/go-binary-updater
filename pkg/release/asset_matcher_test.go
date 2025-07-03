package release

import (
	"runtime"
	"testing"
)

func TestAssetMatcher_K0sPattern(t *testing.T) {
	// Test k0s-style naming: k0s-v1.33.2+k0s.0-amd64
	assetNames := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64", 
		"k0s-v1.33.2+k0s.0-amd64.exe",
		"k0s-v1.33.2+k0s.0-arm64.exe",
	}

	config := DefaultAssetMatchingConfig()
	config.ProjectName = "k0s"
	config.IsDirectBinary = true
	config.Strategy = FlexibleStrategy

	matcher := NewAssetMatcher(config)

	// Test for amd64 architecture
	originalArch := runtime.GOARCH
	originalOS := runtime.GOOS
	
	// Mock amd64 architecture
	matcher.arch = "amd64"
	matcher.os = "linux"
	
	bestMatch, err := matcher.FindBestMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected to find a match for k0s amd64, got error: %v", err)
	}

	expected := "k0s-v1.33.2+k0s.0-amd64"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}

	// Test for arm64 architecture
	matcher.arch = "arm64"
	matcher.os = "linux"
	
	bestMatch, err = matcher.FindBestMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected to find a match for k0s arm64, got error: %v", err)
	}

	expected = "k0s-v1.33.2+k0s.0-arm64"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}

	// Restore original values
	_ = originalArch
	_ = originalOS
}

func TestAssetMatcher_StandardPattern(t *testing.T) {
	// Test standard naming: myapp-Linux_x86_64.tar.gz
	assetNames := []string{
		"myapp-Linux_x86_64.tar.gz",
		"myapp-Darwin_arm64.tar.gz",
		"myapp-Windows_x86_64.zip",
	}

	config := DefaultAssetMatchingConfig()
	config.Strategy = StandardStrategy

	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux"

	bestMatch, err := matcher.FindBestMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected to find a match for standard pattern, got error: %v", err)
	}

	expected := "myapp-Linux_x86_64.tar.gz"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}
}

func TestAssetMatcher_FlexiblePattern(t *testing.T) {
	// Test various naming patterns
	testCases := []struct {
		name        string
		assetNames  []string
		arch        string
		os          string
		projectName string
		expected    string
	}{
		{
			name: "kubectl-style",
			assetNames: []string{
				"kubectl-linux-amd64",
				"kubectl-darwin-arm64",
				"kubectl-windows-amd64.exe",
			},
			arch:        "amd64",
			os:          "linux",
			projectName: "kubectl",
			expected:    "kubectl-linux-amd64",
		},
		{
			name: "helm-style",
			assetNames: []string{
				"helm-v3.12.0-linux-amd64.tar.gz",
				"helm-v3.12.0-darwin-arm64.tar.gz",
				"helm-v3.12.0-windows-amd64.zip",
			},
			arch:        "amd64",
			os:          "linux",
			projectName: "helm",
			expected:    "helm-v3.12.0-linux-amd64.tar.gz",
		},
		{
			name: "terraform-style",
			assetNames: []string{
				"terraform_1.5.0_linux_amd64.zip",
				"terraform_1.5.0_darwin_arm64.zip",
				"terraform_1.5.0_windows_amd64.zip",
			},
			arch:        "amd64",
			os:          "linux",
			projectName: "terraform",
			expected:    "terraform_1.5.0_linux_amd64.zip",
		},
		{
			name: "docker-style",
			assetNames: []string{
				"docker-20.10.17-linux-amd64.tgz",
				"docker-20.10.17-darwin-arm64.tgz",
				"docker-20.10.17-windows-amd64.zip",
			},
			arch:        "amd64",
			os:          "linux",
			projectName: "docker",
			expected:    "docker-20.10.17-linux-amd64.tgz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultAssetMatchingConfig()
			config.ProjectName = tc.projectName
			config.Strategy = FlexibleStrategy

			matcher := NewAssetMatcher(config)
			matcher.arch = tc.arch
			matcher.os = tc.os

			bestMatch, err := matcher.FindBestMatch(tc.assetNames)
			if err != nil {
				t.Fatalf("Expected to find a match for %s, got error: %v", tc.name, err)
			}

			if bestMatch != tc.expected {
				t.Errorf("For %s: expected %s, got %s", tc.name, tc.expected, bestMatch)
			}
		})
	}
}

func TestAssetMatcher_CustomPattern(t *testing.T) {
	// Test custom regex patterns
	assetNames := []string{
		"custom-app-v1.0.0-linux-x86_64.tar.gz",
		"custom-app-v1.0.0-darwin-arm64.tar.gz",
		"custom-app-v1.0.0-windows-x86_64.zip",
	}

	config := DefaultAssetMatchingConfig()
	config.Strategy = CustomStrategy
	config.CustomPatterns = []string{
		`custom-app-.*-{OS}-{ARCH}\..*`,
	}

	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux"

	bestMatch, err := matcher.FindBestMatch(assetNames)
	if err != nil {
		t.Fatalf("Expected to find a match with custom pattern, got error: %v", err)
	}

	expected := "custom-app-v1.0.0-linux-x86_64.tar.gz"
	if bestMatch != expected {
		t.Errorf("Expected %s, got %s", expected, bestMatch)
	}
}

func TestAssetMatcher_ArchitectureVariants(t *testing.T) {
	// Test architecture variant matching
	assetNames := []string{
		"app-linux-x86_64.tar.gz",
		"app-linux-aarch64.tar.gz",
		"app-linux-i386.tar.gz",
	}

	testCases := []struct {
		arch     string
		expected string
	}{
		{"amd64", "app-linux-x86_64.tar.gz"},
		{"arm64", "app-linux-aarch64.tar.gz"},
		{"386", "app-linux-i386.tar.gz"},
	}

	for _, tc := range testCases {
		t.Run(tc.arch, func(t *testing.T) {
			config := DefaultAssetMatchingConfig()
			config.Strategy = FlexibleStrategy

			matcher := NewAssetMatcher(config)
			matcher.arch = tc.arch
			matcher.os = "linux"

			bestMatch, err := matcher.FindBestMatch(assetNames)
			if err != nil {
				t.Fatalf("Expected to find a match for %s, got error: %v", tc.arch, err)
			}

			if bestMatch != tc.expected {
				t.Errorf("For arch %s: expected %s, got %s", tc.arch, tc.expected, bestMatch)
			}
		})
	}
}

func TestAssetMatcher_OSVariants(t *testing.T) {
	// Test OS variant matching
	assetNames := []string{
		"app-linux-amd64.tar.gz",
		"app-darwin-amd64.tar.gz",
		"app-macos-amd64.tar.gz",
		"app-windows-amd64.zip",
		"app-win-amd64.zip",
	}

	testCases := []struct {
		os       string
		expected []string // Multiple valid matches
	}{
		{"linux", []string{"app-linux-amd64.tar.gz"}},
		{"darwin", []string{"app-darwin-amd64.tar.gz", "app-macos-amd64.tar.gz"}},
		{"windows", []string{"app-windows-amd64.zip", "app-win-amd64.zip"}},
	}

	for _, tc := range testCases {
		t.Run(tc.os, func(t *testing.T) {
			config := DefaultAssetMatchingConfig()
			config.Strategy = FlexibleStrategy

			matcher := NewAssetMatcher(config)
			matcher.arch = "amd64"
			matcher.os = tc.os

			bestMatch, err := matcher.FindBestMatch(assetNames)
			if err != nil {
				t.Fatalf("Expected to find a match for %s, got error: %v", tc.os, err)
			}

			// Check if the match is one of the expected values
			found := false
			for _, expected := range tc.expected {
				if bestMatch == expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("For OS %s: expected one of %v, got %s", tc.os, tc.expected, bestMatch)
			}
		})
	}
}

func TestAssetMatcher_NoMatch(t *testing.T) {
	// Test when no suitable asset is found
	assetNames := []string{
		"app-windows-amd64.zip",
		"app-darwin-arm64.tar.gz",
	}

	config := DefaultAssetMatchingConfig()
	config.Strategy = FlexibleStrategy

	matcher := NewAssetMatcher(config)
	matcher.arch = "amd64"
	matcher.os = "linux" // No Linux asset available

	_, err := matcher.FindBestMatch(assetNames)
	if err == nil {
		t.Error("Expected error when no suitable asset found, got success")
	}
}

func TestAssetMatcher_DirectBinaryConfiguration(t *testing.T) {
	// Test direct binary configuration
	config := DefaultAssetMatchingConfig()
	config.IsDirectBinary = true
	config.FileExtensions = []string{} // No extensions for direct binaries

	if config.IsDirectBinary != true {
		t.Error("Expected IsDirectBinary to be true")
	}

	if len(config.FileExtensions) != 0 {
		t.Error("Expected no file extensions for direct binary")
	}
}

func BenchmarkAssetMatcher_FlexibleStrategy(b *testing.B) {
	assetNames := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64",
		"k0s-v1.33.2+k0s.0-amd64.exe",
		"kubectl-linux-amd64",
		"helm-v3.12.0-linux-amd64.tar.gz",
		"terraform_1.5.0_linux_amd64.zip",
		"docker-20.10.17-linux-amd64.tgz",
	}

	config := DefaultAssetMatchingConfig()
	config.Strategy = FlexibleStrategy
	matcher := NewAssetMatcher(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.FindBestMatch(assetNames)
	}
}
