# Flexible Asset Matching

The go-binary-updater library now supports flexible asset matching to handle different project naming conventions and direct binary downloads. This enhancement makes the library compatible with projects like k0s, kubectl, helm, and many others that don't follow the traditional `{OS}_{ARCH}` naming pattern.

## Overview

The flexible asset matching system provides three key capabilities:

1. **Multiple Naming Pattern Support**: Handles various asset naming conventions
2. **Direct Binary Downloads**: Supports projects that release direct binaries (not archived)
3. **Intelligent Matching**: Uses scoring algorithms to find the best asset for your platform

## Configuration

### FileConfig Enhancements

The `FileConfig` struct has been enhanced with new fields for flexible asset handling:

```go
type FileConfig struct {
    // Existing fields...
    VersionedDirectoryName string `json:"versioned_directory_name"`
    SourceBinaryName       string `json:"source_binary_name"`
    BinaryName             string `json:"binary_name"`
    CreateGlobalSymlink    bool   `json:"create_global_symlink"`
    BaseBinaryDirectory    string `json:"base_binary_directory"`
    SourceArchivePath      string `json:"source_archive_path"`
    
    // New fields for flexible asset handling
    IsDirectBinary         bool     `json:"is_direct_binary"`         // True if downloaded asset is a direct binary
    ProjectName            string   `json:"project_name"`             // Project name for asset matching
    AssetMatchingStrategy  string   `json:"asset_matching_strategy"`  // "standard", "flexible", or "custom"
    CustomAssetPatterns    []string `json:"custom_asset_patterns"`    // Custom regex patterns
}
```

### Asset Matching Strategies

#### 1. Standard Strategy (`"standard"`)
Uses the traditional `{OS}_{ARCH}` pattern (e.g., `Linux_x86_64`).

```go
config := fileUtils.FileConfig{
    // ... other fields
    AssetMatchingStrategy: "standard",
}
```

#### 2. Flexible Strategy (`"flexible"`) - **Recommended**
Intelligently matches various naming patterns using scoring algorithms.

```go
config := fileUtils.FileConfig{
    // ... other fields
    ProjectName:           "k0s",
    AssetMatchingStrategy: "flexible",
}
```

#### 3. Custom Strategy (`"custom"`)
Uses user-defined regex patterns for very specific naming conventions.

```go
config := fileUtils.FileConfig{
    // ... other fields
    AssetMatchingStrategy: "custom",
    CustomAssetPatterns: []string{
        `{PROJECT}-v\d+\.\d+\.\d+-{OS}-{ARCH}\.tar\.gz`,
        `{PROJECT}-{ARCH}-{OS}\.zip`,
    },
}
```

## Supported Project Types

### k0s (Direct Binary)

k0s releases direct binaries with names like `k0s-v1.33.2+k0s.0-amd64`.

```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "k0s",
    BinaryName:             "k0s",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/k0s-latest",
    
    // k0s-specific configuration
    IsDirectBinary:         true,
    ProjectName:            "k0s",
    AssetMatchingStrategy:  "flexible",
}

githubRelease := release.NewGithubRelease("k0sproject/k0s", config)
```

### kubectl (Direct Binary)

kubectl releases with names like `kubectl-linux-amd64`.

```go
config := fileUtils.FileConfig{
    // ... basic configuration
    IsDirectBinary:         true,
    ProjectName:            "kubectl",
    AssetMatchingStrategy:  "flexible",
}
```

### helm (Archived Binary)

helm releases with names like `helm-v3.12.0-linux-amd64.tar.gz`.

```go
config := fileUtils.FileConfig{
    // ... basic configuration
    IsDirectBinary:         false, // helm releases are archived
    ProjectName:            "helm",
    AssetMatchingStrategy:  "flexible",
}
```

### terraform (Archived Binary)

terraform releases with names like `terraform_1.5.0_linux_amd64.zip`.

```go
config := fileUtils.FileConfig{
    // ... basic configuration
    IsDirectBinary:         false,
    ProjectName:            "terraform",
    AssetMatchingStrategy:  "flexible",
}
```

## Architecture Support

The flexible matching system supports extensive architecture mapping:

### Primary Architectures
- **amd64**: Maps to `x86_64`, `amd64`, `x64`
- **arm64**: Maps to `arm64`, `aarch64`
- **arm**: Maps to `arm`, `armv6`, `armv7`, `armhf`
- **386**: Maps to `i386`, `386`, `i686`, `x86`

### Specialized Architectures
- **MIPS**: `mips`, `mipsle`, `mips64`, `mips64le`
- **PowerPC**: `ppc64`, `ppc64le`
- **IBM System z**: `s390x`
- **RISC-V**: `riscv64`
- **WebAssembly**: `wasm`

## Operating System Support

### OS Aliases
- **Linux**: `linux`, `Linux`
- **macOS**: `darwin`, `Darwin`, `macos`, `macOS`, `osx`, `OSX`
- **Windows**: `windows`, `Windows`, `win`, `Win`
- **BSD**: `freebsd`, `openbsd`, `netbsd`

## Custom Patterns

For projects with very specific naming conventions, you can define custom regex patterns:

### Pattern Placeholders
- `{OS}`: Replaced with OS aliases (linux|darwin|windows)
- `{ARCH}`: Replaced with architecture aliases (amd64|x86_64|arm64)
- `{PROJECT}`: Replaced with the project name

### Example Custom Patterns

```go
config := fileUtils.FileConfig{
    // ... other fields
    AssetMatchingStrategy: "custom",
    CustomAssetPatterns: []string{
        // Pattern for: myapp-v1.2.3-linux-amd64.tar.gz
        `{PROJECT}-v\d+\.\d+\.\d+-{OS}-{ARCH}\.tar\.gz`,
        
        // Pattern for: myapp-linux-amd64-v1.2.3.zip
        `{PROJECT}-{OS}-{ARCH}-v\d+\.\d+\.\d+\.zip`,
        
        // Pattern for: myapp_1.2.3_linux_amd64.deb
        `{PROJECT}_\d+\.\d+\.\d+_{OS}_{ARCH}\.deb`,
    },
}
```

## Direct Binary vs Archived Binary

### Direct Binary (`IsDirectBinary: true`)
- The downloaded asset is the binary itself
- No extraction step required
- Examples: k0s, kubectl (in some distributions)

### Archived Binary (`IsDirectBinary: false`)
- The downloaded asset is an archive (tar.gz, zip, etc.)
- Extraction step required to get the binary
- Examples: helm, terraform, docker

## Testing Asset Matching

You can test asset matching without downloading:

```go
// Create asset matcher configuration
config := release.DefaultAssetMatchingConfig()
config.ProjectName = "k0s"
config.IsDirectBinary = true
config.Strategy = release.FlexibleStrategy

// Example asset names
assetNames := []string{
    "k0s-v1.33.2+k0s.0-amd64",
    "k0s-v1.33.2+k0s.0-arm64",
    "k0s-v1.33.2+k0s.0-amd64.exe",
}

matcher := release.NewAssetMatcher(config)
bestMatch, err := matcher.FindBestMatch(assetNames)
if err != nil {
    log.Printf("No match found: %v", err)
} else {
    log.Printf("Best match: %s", bestMatch)
}
```

## Scoring Algorithm

The flexible matching system uses a scoring algorithm to find the best asset:

### Scoring Criteria
- **OS Match**: +10 points
- **Architecture Match**: +10 points
- **Both OS and Arch**: +5 bonus points
- **Architecture-only Match** (for projects like k0s): +8 points
- **Common Patterns**: +3 points
- **Expected File Extensions**: +2 points
- **Wrong Platform**: -20 points

### Example Scoring
For k0s assets on Linux amd64:
- `k0s-v1.33.2+k0s.0-amd64`: 18 points (arch match + pattern match)
- `k0s-v1.33.2+k0s.0-arm64`: 0 points (wrong arch)
- `k0s-v1.33.2+k0s.0-amd64.exe`: 18 points (arch match + pattern match)

## Migration Guide

### From Standard to Flexible Matching

**Before:**
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
}
```

**After:**
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
    
    // Add flexible matching
    ProjectName:            "myapp",
    AssetMatchingStrategy:  "flexible",
    IsDirectBinary:         false, // or true for direct binaries
}
```

## Backward Compatibility

The flexible asset matching system is fully backward compatible:

- Existing configurations continue to work without changes
- Default strategy is "flexible" which includes standard pattern matching
- Legacy `GetReleaseLink()` methods fall back to old logic if new matching fails

## Troubleshooting

### No Suitable Asset Found

If you get "no suitable asset found" errors:

1. **Check Asset Names**: Verify the actual asset names in the release
2. **Try Flexible Strategy**: Use `AssetMatchingStrategy: "flexible"`
3. **Set Project Name**: Provide the `ProjectName` for better pattern matching
4. **Use Custom Patterns**: Define specific regex patterns for unusual naming

### Wrong Asset Selected

If the wrong asset is selected:

1. **Check Platform Detection**: Verify `runtime.GOOS` and `runtime.GOARCH`
2. **Review Scoring**: The highest-scored asset wins
3. **Use Custom Patterns**: Define more specific patterns
4. **Test Matching**: Use the asset matcher directly to debug

## Examples

See the complete examples in:
- [`examples/k0s_example.go`](../examples/k0s_example.go) - k0s direct binary example
- [`examples/github_example.go`](../examples/github_example.go) - Traditional archived binary example
- [`examples/gitlab_example.go`](../examples/gitlab_example.go) - GitLab release example
