package release

import (
	"testing"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
)

// TestCDNStrategyPriority tests that CDN strategy is properly prioritized over GitHub releases
func TestCDNStrategyPriority(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() (*GithubRelease, AssetMatchingConfig)
		expectedStrategy AssetMatchingStrategy
		description    string
	}{
		{
			name: "GetHelmCDNConfig with NewGithubReleaseWithAssetConfig",
			setupFunc: func() (*GithubRelease, AssetMatchingConfig) {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "helm",
					BinaryName:             "helm",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/helm-latest.tar.gz",
					IsDirectBinary:         false,
					ProjectName:            "helm",
				}
				
				helmConfig := GetHelmCDNConfig()
				githubRelease := NewGithubReleaseWithAssetConfig("helm/helm", config, helmConfig)
				return githubRelease, helmConfig
			},
			expectedStrategy: CDNStrategy,
			description:      "Helm CDN config should preserve CDN strategy",
		},
		{
			name: "GetHelmCDNConfig with NewGithubReleaseWithCDNConfig",
			setupFunc: func() (*GithubRelease, AssetMatchingConfig) {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "helm",
					BinaryName:             "helm",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/helm-latest.tar.gz",
					IsDirectBinary:         false,
					ProjectName:            "helm",
				}
				
				helmConfig := GetHelmCDNConfig()
				githubRelease := NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)
				return githubRelease, helmConfig
			},
			expectedStrategy: CDNStrategy,
			description:      "CDN config constructor should enforce CDN strategy",
		},
		{
			name: "GetKubectlCDNConfig with NewGithubReleaseWithAssetConfig",
			setupFunc: func() (*GithubRelease, AssetMatchingConfig) {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "kubectl",
					BinaryName:             "kubectl",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/kubectl-latest",
					IsDirectBinary:         true,
					ProjectName:            "kubectl",
				}
				
				kubectlConfig := GetKubectlCDNConfig()
				githubRelease := NewGithubReleaseWithAssetConfig("kubernetes/kubernetes", config, kubectlConfig)
				return githubRelease, kubectlConfig
			},
			expectedStrategy: CDNStrategy,
			description:      "kubectl CDN config should preserve CDN strategy",
		},
		{
			name: "GetTerraformConfig with HybridStrategy",
			setupFunc: func() (*GithubRelease, AssetMatchingConfig) {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "terraform",
					BinaryName:             "terraform",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/terraform-latest.zip",
					IsDirectBinary:         false,
					ProjectName:            "terraform",
				}
				
				terraformConfig := GetTerraformConfig()
				githubRelease := NewGithubReleaseWithAssetConfig("hashicorp/terraform", config, terraformConfig)
				return githubRelease, terraformConfig
			},
			expectedStrategy: HybridStrategy,
			description:      "Terraform config should preserve Hybrid strategy",
		},
		{
			name: "Auto-detect CDN strategy from configuration",
			setupFunc: func() (*GithubRelease, AssetMatchingConfig) {
				config := fileUtils.FileConfig{
					VersionedDirectoryName: "versions",
					SourceBinaryName:       "custom",
					BinaryName:             "custom",
					BaseBinaryDirectory:    "/tmp/test",
					SourceArchivePath:      "/tmp/custom-latest.tar.gz",
					IsDirectBinary:         false,
					ProjectName:            "custom",
				}
				
				// Create a custom config with CDN settings but no explicit strategy
				customConfig := DefaultAssetMatchingConfig()
				customConfig.CDNBaseURL = "https://example.com/releases/"
				customConfig.CDNPattern = "custom-{version}-{os}-{arch}.tar.gz"
				customConfig.Strategy = FlexibleStrategy // This should be auto-detected as CDN
				
				githubRelease := NewGithubReleaseWithAssetConfig("example/custom", config, customConfig)
				return githubRelease, customConfig
			},
			expectedStrategy: CDNStrategy,
			description:      "Should auto-detect CDN strategy when CDN config is present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			githubRelease, originalConfig := tt.setupFunc()
			
			// Verify the strategy is correctly set
			if githubRelease.AssetMatchingConfig.Strategy != tt.expectedStrategy {
				t.Errorf("%s: expected strategy %v, got %v", 
					tt.description, tt.expectedStrategy, githubRelease.AssetMatchingConfig.Strategy)
			}
			
			// Verify CDN configuration is preserved
			if originalConfig.CDNBaseURL != "" {
				if githubRelease.AssetMatchingConfig.CDNBaseURL != originalConfig.CDNBaseURL {
					t.Errorf("%s: CDN base URL not preserved. Expected %s, got %s",
						tt.description, originalConfig.CDNBaseURL, githubRelease.AssetMatchingConfig.CDNBaseURL)
				}
			}
			
			if originalConfig.CDNPattern != "" {
				if githubRelease.AssetMatchingConfig.CDNPattern != originalConfig.CDNPattern {
					t.Errorf("%s: CDN pattern not preserved. Expected %s, got %s",
						tt.description, originalConfig.CDNPattern, githubRelease.AssetMatchingConfig.CDNPattern)
				}
			}
		})
	}
}

// TestBackwardCompatibility ensures existing code still works
func TestBackwardCompatibility(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "test",
		BinaryName:             "test",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test-latest.tar.gz",
		AssetMatchingStrategy:  "flexible",
	}
	
	// Old way should still work
	githubRelease := NewGithubRelease("owner/repo", config)
	
	if githubRelease.AssetMatchingConfig.Strategy != FlexibleStrategy {
		t.Errorf("Backward compatibility broken: expected FlexibleStrategy, got %v", 
			githubRelease.AssetMatchingConfig.Strategy)
	}
}

// TestCDNStrategyFromFileConfig tests that CDN strategy can be set via FileConfig
func TestCDNStrategyFromFileConfig(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "test",
		BinaryName:             "test",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test-latest.tar.gz",
		AssetMatchingStrategy:  "cdn",
	}
	
	githubRelease := NewGithubRelease("owner/repo", config)
	
	if githubRelease.AssetMatchingConfig.Strategy != CDNStrategy {
		t.Errorf("CDN strategy from FileConfig not working: expected CDNStrategy, got %v", 
			githubRelease.AssetMatchingConfig.Strategy)
	}
}

// TestHybridStrategyFromFileConfig tests that Hybrid strategy can be set via FileConfig
func TestHybridStrategyFromFileConfig(t *testing.T) {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "test",
		BinaryName:             "test",
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/test-latest.tar.gz",
		AssetMatchingStrategy:  "hybrid",
	}
	
	githubRelease := NewGithubRelease("owner/repo", config)
	
	if githubRelease.AssetMatchingConfig.Strategy != HybridStrategy {
		t.Errorf("Hybrid strategy from FileConfig not working: expected HybridStrategy, got %v", 
			githubRelease.AssetMatchingConfig.Strategy)
	}
}
