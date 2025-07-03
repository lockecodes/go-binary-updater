# Enhanced Binary Download Patterns

The go-binary-updater library now supports complex binary distribution patterns that address real-world challenges with projects like k0s, Helm, kubectl, and Terraform. This document explains the enhanced asset filtering, CDN download support, and hybrid strategies implemented to handle diverse binary distribution mechanisms.

## Problem Statement

Real-world binary distribution presents several challenges:

1. **k0s Asset Selection**: k0s releases both direct binaries (`k0s-v1.33.2+k0s.0-amd64`) and airgap bundles (`k0s-airgap-bundle-v1.33.2+k0s.0-amd64`), requiring precise filtering
2. **Helm CDN Distribution**: Helm binaries are distributed via external CDN (get.helm.sh) rather than GitHub releases
3. **kubectl Google CDN**: kubectl uses Google's CDN infrastructure with different URL patterns
4. **Signature File Pollution**: Many projects include signature files (.asc, .sig) that cause extraction errors

## Enhanced Asset Filtering System

### Exclusion Patterns

The new `ExcludePatterns` field allows explicit exclusion of unwanted assets:

```go
config := release.AssetMatchingConfig{
    Strategy: release.FlexibleStrategy,
    ExcludePatterns: []string{
        "airgap",     // Exclude airgap bundles (k0s)
        "\\.asc$",    // Exclude signature files
        "\\.sig$",    // Exclude signature files
        "\\.sha256$", // Exclude checksum files
    },
}
```

### Priority Patterns

The `PriorityPatterns` field gives higher scores to preferred asset types:

```go
config := release.AssetMatchingConfig{
    Strategy: release.FlexibleStrategy,
    PriorityPatterns: []string{
        "^k0s-v.*-amd64$",     // Prefer direct k0s binaries for amd64
        "^k0s-v.*-arm64$",     // Prefer direct k0s binaries for arm64
    },
}
```

### Automatic Signature File Detection

The default configuration automatically excludes common signature and checksum files:

```go
// Default exclusion patterns
ExcludePatterns: []string{
    "airgap",     // Exclude airgap bundles
    "\\.asc$",    // Exclude signature files
    "\\.sig$",    // Exclude signature files
    "\\.sha256$", // Exclude checksum files
    "\\.sha512$", // Exclude checksum files
    "\\.md5$",    // Exclude checksum files
}
```

## CDN Download Support

### CDN Strategy

The new `CDNStrategy` downloads binaries from external CDNs instead of GitHub/GitLab releases:

```go
config := release.AssetMatchingConfig{
    Strategy:    release.CDNStrategy,
    CDNBaseURL:  "https://get.helm.sh/",
    CDNPattern:  "helm-{version}-{os}-{arch}.tar.gz",
    IsDirectBinary: false,
}
```

### URL Construction

CDN URLs are constructed using placeholders:
- `{version}`: Release version (e.g., "v3.12.0" or "3.12.0")
- `{os}`: Operating system (e.g., "linux", "darwin", "windows")
- `{arch}`: Architecture (e.g., "amd64", "arm64")

### Version Format Configuration

The `CDNVersionFormat` field controls how version strings are formatted for CDN URLs:

- **`"with-v"`**: Ensures version has "v" prefix (e.g., "3.18.3" → "v3.18.3")
- **`"without-v"`**: Ensures version doesn't have "v" prefix (e.g., "v3.18.3" → "3.18.3")
- **`"as-is"`**: Uses version exactly as provided (default)

Example: `https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz`

### CDN-Specific Architecture Mapping

Different CDNs have different architecture naming conventions. The library uses configurable architecture mapping to handle these differences:

#### Configurable Architecture Mapping
Each CDN configuration can specify custom architecture mappings via the `CDNArchMapping` field:

```go
config.CDNArchMapping = map[string]string{
    "amd64":   "amd64",  // Preserve amd64 (don't convert to x86_64)
    "x86_64":  "amd64",  // Convert x86_64 to amd64
    "x64":     "amd64",  // Convert x64 to amd64
    "arm64":   "arm64",  // Preserve arm64
    "aarch64": "arm64",  // Convert aarch64 to arm64
}
```

#### Preset CDN Mappings
- **Helm and kubectl**: Use "amd64" format (not "x86_64")
- **Terraform and others**: Fall back to standard MapArch function (amd64 → x86_64)

#### Custom CDN Support
Any CDN can be configured with custom architecture mappings:

```go
customConfig := release.AssetMatchingConfig{
    CDNArchMapping: map[string]string{
        "amd64": "x86_64",   // Custom mapping for this CDN
        "arm64": "aarch64",  // Custom mapping for this CDN
    },
}
```

This ensures that each CDN receives the exact architecture format it expects.

### Supported CDN Providers

#### Helm (get.helm.sh)
```go
config := release.GetHelmCDNConfig()
// CDN Base URL: https://get.helm.sh/
// CDN Pattern: helm-{version}-{os}-{arch}.tar.gz
// CDN Version Format: "with-v" (ensures v3.18.3 format)
// Architecture Mapping: amd64 preserved (not converted to x86_64)
// Extraction: Binary located in {os}-{arch}/helm subdirectory
```

#### kubectl (Google CDN)
```go
config := release.GetKubectlCDNConfig()
// CDN Base URL: https://dl.k8s.io/release/
// CDN Pattern: {version}/bin/{os}/{arch}/kubectl
// CDN Version Format: "as-is" (uses version exactly as provided)
// Architecture Mapping: amd64 preserved (not converted to x86_64)
// Direct Binary: No extraction required
```

#### Terraform (HashiCorp CDN)
```go
config := release.GetTerraformConfig()
// CDN Base URL: https://releases.hashicorp.com/terraform/
// CDN Pattern: {version}/terraform_{version}_{os}_{arch}.zip
// CDN Version Format: "without-v" (removes v prefix: v1.5.0 → 1.5.0)
// Hybrid Strategy: Try GitHub first, then CDN
```

## Hybrid Download Strategy

### Hybrid Strategy

The `HybridStrategy` tries GitHub/GitLab releases first, then falls back to CDN:

```go
config := release.AssetMatchingConfig{
    Strategy:    release.HybridStrategy,
    CDNBaseURL:  "https://releases.hashicorp.com/terraform/",
    CDNPattern:  "{version}/terraform_{version}_{os}_{arch}.zip",
}
```

### Fallback Logic

1. **Primary**: Attempt to find matching assets in GitHub/GitLab releases
2. **Fallback**: If no suitable assets found, construct CDN download URL
3. **Error Handling**: Provide clear feedback about which method was attempted

## Enhanced Archive Extraction

### Extraction Configuration

The new `ExtractionConfig` supports complex archive structures:

```go
type ExtractionConfig struct {
    StripComponents int    // Number of directory components to strip
    BinaryPath      string // Specific path to binary within archive
}
```

### Complex Archive Handling

#### Helm Archive Structure
```
helm-v3.12.0-linux-amd64.tar.gz
├── linux-amd64/
│   ├── helm          # Binary location
│   ├── LICENSE
│   └── README.md
```

Configuration:
```go
config.ExtractionConfig = &ExtractionConfig{
    BinaryPath: "{os}-{arch}/helm",
}
```

#### Standard Archive Structure
```
myapp-v1.0.0-linux-amd64.tar.gz
├── myapp             # Binary at root
├── LICENSE
└── README.md
```

No special configuration needed - uses standard binary finding logic.

## Specific Binary Configurations

### k0s Configuration

Addresses the airgap bundle selection problem:

```go
config := release.GetK0sConfig()
// Strategy: FlexibleStrategy
// ExcludePatterns: ["airgap", "bundle", "\\.asc$", "\\.sha256$"]
// PriorityPatterns: ["^k0s-v.*-amd64$", "^k0s-v.*-arm64$"]
// IsDirectBinary: true
```

**Problem Solved**: Correctly selects `k0s-v1.33.2+k0s.0-amd64` instead of `k0s-airgap-bundle-v1.33.2+k0s.0-amd64`

### Helm Configuration

Addresses the CDN distribution challenge:

```go
config := release.GetHelmCDNConfig()
// Strategy: CDNStrategy
// CDNBaseURL: "https://get.helm.sh/"
// CDNPattern: "helm-{version}-{os}-{arch}.tar.gz"
// ExtractionConfig: BinaryPath: "{os}-{arch}/helm"
```

**Problem Solved**: Downloads from get.helm.sh instead of failing on GitHub signature files

### kubectl Configuration

Addresses the Google CDN requirement:

```go
config := release.GetKubectlCDNConfig()
// Strategy: CDNStrategy
// CDNBaseURL: "https://dl.k8s.io/release/"
// CDNPattern: "{version}/bin/{os}/{arch}/kubectl"
// IsDirectBinary: true
```

**Problem Solved**: Downloads from Google's official CDN with correct URL structure

### Docker Configuration

Addresses package filtering challenges:

```go
config := release.GetDockerConfig()
// Strategy: FlexibleStrategy
// ExcludePatterns: ["desktop", "rootless", "static"]
// PriorityPatterns: ["docker-.*-{os}-{arch}\\.tgz$"]
```

**Problem Solved**: Excludes Docker Desktop and rootless packages, selects CLI packages

## Usage Examples

### Complete k0s Example

```go
package main

import (
    "log"
    "gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
    "gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
    // Get preset k0s configuration
    assetConfig := release.GetK0sConfig()
    
    config := fileUtils.FileConfig{
        VersionedDirectoryName: "versions",
        SourceBinaryName:       "k0s",
        BinaryName:             "k0s",
        CreateLocalSymlink:     true,
        BaseBinaryDirectory:    "/home/user/.local/bin",
        SourceArchivePath:      "/tmp/k0s-latest",
        IsDirectBinary:         true,
        ProjectName:            "k0s",
        AssetMatchingStrategy:  "flexible",
    }

    githubRelease := release.NewGithubRelease("k0sproject/k0s", config)
    githubRelease.AssetMatchingConfig = assetConfig

    // Download and install
    if err := githubRelease.DownloadLatestRelease(); err != nil {
        log.Fatalf("Download failed: %v", err)
    }
    
    if err := githubRelease.InstallLatestRelease(); err != nil {
        log.Fatalf("Installation failed: %v", err)
    }
    
    // Get installation info
    info, _ := githubRelease.GetInstallationInfo()
    log.Printf("k0s installed at: %s", info.BinaryPath)
}
```

### Complete Helm CDN Example

```go
func installHelmFromCDN() {
    // Get preset Helm CDN configuration
    assetConfig := release.GetHelmCDNConfig()
    
    config := fileUtils.FileConfig{
        VersionedDirectoryName: "versions",
        SourceBinaryName:       "helm",
        BinaryName:             "helm",
        CreateLocalSymlink:     true,
        BaseBinaryDirectory:    "/home/user/.local/bin",
        SourceArchivePath:      "/tmp/helm-latest.tar.gz",
        IsDirectBinary:         false,
        ProjectName:            "helm",
        AssetMatchingStrategy:  "cdn",
    }

    githubRelease := release.NewGithubRelease("helm/helm", config)
    githubRelease.AssetMatchingConfig = assetConfig

    // Downloads from get.helm.sh instead of GitHub
    if err := githubRelease.DownloadLatestRelease(); err != nil {
        log.Fatalf("CDN download failed: %v", err)
    }
    
    // Extracts from linux-amd64/helm subdirectory
    if err := githubRelease.InstallLatestRelease(); err != nil {
        log.Fatalf("Installation failed: %v", err)
    }
}
```

### Hybrid Strategy Example

```go
func installTerraformWithHybrid() {
    // Get preset Terraform hybrid configuration
    assetConfig := release.GetTerraformConfig()
    
    config := fileUtils.FileConfig{
        VersionedDirectoryName: "versions",
        SourceBinaryName:       "terraform",
        BinaryName:             "terraform",
        CreateLocalSymlink:     true,
        BaseBinaryDirectory:    "/home/user/.local/bin",
        SourceArchivePath:      "/tmp/terraform-latest.zip",
        IsDirectBinary:         false,
        ProjectName:            "terraform",
        AssetMatchingStrategy:  "hybrid",
    }

    githubRelease := release.NewGithubRelease("hashicorp/terraform", config)
    githubRelease.AssetMatchingConfig = assetConfig

    // Tries GitHub first, falls back to HashiCorp CDN
    if err := githubRelease.DownloadLatestRelease(); err != nil {
        log.Fatalf("Hybrid download failed: %v", err)
    }
    
    if err := githubRelease.InstallLatestRelease(); err != nil {
        log.Fatalf("Installation failed: %v", err)
    }
}
```

## Error Handling and Debugging

### Comprehensive Error Messages

The enhanced system provides detailed error messages:

```go
// Asset filtering errors
"no assets remaining after applying exclusion filters. Original assets: [list], Excluded patterns: [patterns]"

// CDN configuration errors
"CDN strategy requires CDNBaseURL and CDNPattern to be configured"
"CDN pattern must contain {version} placeholder"

// Validation errors
"binary not found at specified path: /path/to/binary"
"InstallArchivedBinary called but IsDirectBinary is true - this indicates a configuration error"
```

### Configuration Validation

```go
// Validate CDN configuration
if err := release.ValidateCDNConfig(assetConfig); err != nil {
    log.Fatalf("Invalid CDN configuration: %v", err)
}

// Check preset availability
if _, err := release.GetPresetConfig("unknown-binary"); err != nil {
    log.Printf("No preset config available: %v", err)
    // Fall back to default configuration
}
```

## Migration Guide

### From Standard to Enhanced Matching

**Before:**
```go
config := fileUtils.FileConfig{
    ProjectName:            "k0s",
    AssetMatchingStrategy:  "flexible",
}
```

**After:**
```go
// Use preset configuration
assetConfig := release.GetK0sConfig()
config := fileUtils.FileConfig{
    ProjectName:            "k0s",
    AssetMatchingStrategy:  "flexible",
}
githubRelease.AssetMatchingConfig = assetConfig
```

### Adding CDN Support

**Before (GitHub only):**
```go
githubRelease := release.NewGithubRelease("helm/helm", config)
```

**After (CDN support):**
```go
assetConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubRelease("helm/helm", config)
githubRelease.AssetMatchingConfig = assetConfig
```

## Best Practices

### 1. Use Preset Configurations
- Always check for preset configurations first: `release.GetPresetConfig(binaryName)`
- Preset configurations are tested and optimized for specific binaries
- Fall back to custom configuration only when needed

### 2. Validate CDN Configurations
- Always validate CDN configurations: `release.ValidateCDNConfig(config)`
- Ensure required placeholders are present in CDN patterns
- Test CDN URLs manually before deploying

### 3. Handle Extraction Complexity
- Use `ExtractionConfig` for binaries in subdirectories
- Test extraction with sample archives
- Provide fallback binary finding logic

### 4. Implement Comprehensive Error Handling
- Check for configuration errors before attempting downloads
- Provide clear feedback about filtering decisions
- Log asset lists and matching results for debugging

## Summary

The enhanced binary download patterns successfully address real-world distribution challenges:

1. **✅ k0s Asset Selection**: Strict filtering excludes airgap bundles and prefers direct binaries
2. **✅ Helm CDN Distribution**: CDN strategy downloads from get.helm.sh with proper extraction
3. **✅ kubectl Google CDN**: Direct binary downloads from Google's official CDN
4. **✅ Signature File Handling**: Automatic exclusion of signature and checksum files
5. **✅ Hybrid Strategies**: Intelligent fallback from GitHub to CDN sources
6. **✅ Complex Archive Support**: Enhanced extraction for subdirectory structures

The implementation maintains 100% backward compatibility while providing powerful new capabilities for handling diverse binary distribution patterns. Users can now configure k0s, helm, kubectl, terraform, and other binaries with simple, intuitive configurations that "just work" regardless of the underlying distribution mechanism.
