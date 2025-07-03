# CDN Strategy Priority Fix

## Problem Statement

The go-binary-updater library had a critical issue where CDN download functionality was not working correctly despite having comprehensive CDN support infrastructure. The library incorrectly prioritized GitHub releases over CDN downloads even when CDN configuration was explicitly provided.

## Root Cause

The issue was in the `NewGithubRelease()` and `NewGitlabRelease()` constructors, which would:

1. Create a `DefaultAssetMatchingConfig()` with `FlexibleStrategy`
2. Override the strategy based on `fileConfig.AssetMatchingStrategy`
3. **Overwrite any CDN strategy** that was set in preset configurations like `GetHelmCDNConfig()`

This meant that even when users configured CDN settings correctly, the strategy would be reset to `FlexibleStrategy`, causing the library to attempt GitHub releases instead of CDN downloads.

## Solution

### 1. Enhanced Constructors

Added new constructor functions that preserve CDN strategy settings:

```go
// Preserves any CDN strategy settings in the provided configuration
func NewGithubReleaseWithAssetConfig(repository string, fileConfig fileUtils.FileConfig, assetConfig AssetMatchingConfig) *GithubRelease

// Convenience function that enforces CDN strategy
func NewGithubReleaseWithCDNConfig(repository string, fileConfig fileUtils.FileConfig, cdnConfig AssetMatchingConfig) *GithubRelease

// Similar functions for GitLab
func NewGitlabReleaseWithAssetConfig(projectId string, fileConfig fileUtils.FileConfig, assetConfig AssetMatchingConfig) *GitLabRelease
func NewGitlabReleaseWithCDNConfig(projectId string, fileConfig fileUtils.FileConfig, cdnConfig AssetMatchingConfig) *GitLabRelease
```

### 2. Auto-Detection Logic

Added automatic CDN strategy detection when CDN configuration is present:

```go
// Auto-detect CDN strategy if CDN configuration is present but strategy is not CDN/Hybrid
if assetConfig.CDNBaseURL != "" && assetConfig.CDNPattern != "" {
    if assetConfig.Strategy != CDNStrategy && assetConfig.Strategy != HybridStrategy {
        assetConfig.Strategy = CDNStrategy
    }
}
```

### 3. FileConfig Strategy Support

Enhanced the original constructors to support CDN and Hybrid strategies via `FileConfig`:

```go
switch fileConfig.AssetMatchingStrategy {
case "standard":
    assetConfig.Strategy = StandardStrategy
case "flexible":
    assetConfig.Strategy = FlexibleStrategy
case "custom":
    assetConfig.Strategy = CustomStrategy
case "cdn":           // NEW
    assetConfig.Strategy = CDNStrategy
case "hybrid":        // NEW
    assetConfig.Strategy = HybridStrategy
default:
    assetConfig.Strategy = FlexibleStrategy
}
```

## Usage Examples

### Before (Broken)

```go
// This would NOT work correctly - CDN strategy would be overwritten
config := fileUtils.FileConfig{...}
helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubRelease("helm/helm", config)
githubRelease.AssetMatchingConfig = helmConfig  // Strategy already overwritten!
```

### After (Fixed)

#### Method 1: Using NewGithubReleaseWithAssetConfig

```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "helm",
    BinaryName:             "helm",
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/helm-latest.tar.gz",
}

helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubReleaseWithAssetConfig("helm/helm", config, helmConfig)

// Downloads from get.helm.sh instead of GitHub
err := githubRelease.DownloadLatestRelease()
```

#### Method 2: Using NewGithubReleaseWithCDNConfig

```go
config := fileUtils.FileConfig{...}
helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

// Guaranteed to use CDN strategy
err := githubRelease.DownloadLatestRelease()
```

#### Method 3: Using FileConfig Strategy

```go
config := fileUtils.FileConfig{
    // ... other fields
    AssetMatchingStrategy: "cdn",
}

githubRelease := release.NewGithubRelease("helm/helm", config)
// Must still set CDN configuration manually
githubRelease.AssetMatchingConfig.CDNBaseURL = "https://get.helm.sh/"
githubRelease.AssetMatchingConfig.CDNPattern = "helm-{version}-{os}-{arch}.tar.gz"
```

## Preset Configurations

All preset configurations now work correctly:

### Helm CDN

```go
helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)
// Downloads from https://get.helm.sh/
```

### kubectl CDN

```go
kubectlConfig := release.GetKubectlCDNConfig()
githubRelease := release.NewGithubReleaseWithCDNConfig("kubernetes/kubernetes", config, kubectlConfig)
// Downloads from https://dl.k8s.io/release/
```

### Terraform Hybrid

```go
terraformConfig := release.GetTerraformConfig()
githubRelease := release.NewGithubReleaseWithAssetConfig("hashicorp/terraform", config, terraformConfig)
// Tries GitHub first, falls back to https://releases.hashicorp.com/terraform/
```

## Backward Compatibility

All existing code continues to work without changes:

```go
// This still works exactly as before
config := fileUtils.FileConfig{
    AssetMatchingStrategy: "flexible",
    // ... other fields
}
githubRelease := release.NewGithubRelease("owner/repo", config)
```

## Testing

The fix includes comprehensive tests:

- `TestCDNStrategyPriority`: Verifies CDN strategy is preserved
- `TestBackwardCompatibility`: Ensures existing code still works
- `TestCDNStrategyFromFileConfig`: Tests FileConfig strategy support
- `TestHybridStrategyFromFileConfig`: Tests Hybrid strategy support

## Migration Guide

### For New Code

Use the new constructors for CDN configurations:

```go
// Recommended approach
helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)
```

### For Existing Code

No changes required, but you can optionally migrate to the new constructors for better clarity:

```go
// Old way (still works)
githubRelease := release.NewGithubRelease("owner/repo", config)

// New way (more explicit)
assetConfig := release.DefaultAssetMatchingConfig()
githubRelease := release.NewGithubReleaseWithAssetConfig("owner/repo", config, assetConfig)
```

## Benefits

1. **CDN Downloads Work**: Helm, kubectl, and other CDN-based binaries now download correctly
2. **Deterministic Strategy Selection**: Strategy selection is now predictable and respects user configuration
3. **Backward Compatibility**: Existing code continues to work without changes
4. **Auto-Detection**: CDN strategy is automatically detected when CDN configuration is present
5. **Multiple Configuration Methods**: Users can configure CDN strategy via constructors or FileConfig

## Verification

Run the demonstration to see the fix in action:

```bash
go run examples/cdn_strategy_fix_demo.go
```

This will show that:
- Helm CDN configuration preserves CDN strategy
- kubectl CDN configuration works correctly
- Terraform hybrid strategy is maintained
- Backward compatibility is preserved
- Auto-detection works as expected
