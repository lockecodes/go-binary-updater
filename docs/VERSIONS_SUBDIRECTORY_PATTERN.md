# Versions Subdirectory Pattern

## Overview

The go-binary-updater library now supports a cleaner directory structure that separates version storage from symlinks using a `versions/` subdirectory pattern. This enhancement addresses the issue where version directories had the same name as the desired symlinks, causing conflicts and cluttered bin directories.

## Problem Statement

### Before (Legacy Pattern)
The library previously created version directories with the same name as the desired symlink:

```
~/.config/central-air/bin/
├── bat/                    # Should be symlink, not directory
│   └── v0.25.0/
│       └── bat
├── helm/                   # Should be symlink, not directory  
│   └── v3.18.3/
│       └── helm
└── k0s/                    # Should be symlink, not directory
    └── v1.33.2+k0s.0/
        └── k0s
```

**Issues:**
- Cluttered bin directory with version directories
- Naming conflicts between symlinks and directories
- Difficult to distinguish between actual binaries and version storage
- Not intuitive for users expecting clean bin directories

### After (New Pattern)
The new pattern creates a clean separation:

```
~/.config/central-air/bin/
├── bat -> versions/bat/v0.25.0/bat        # Clean symlinks
├── helm -> versions/helm/v3.18.3/helm     
├── k0s -> versions/k0s/v1.33.2+k0s.0/k0s  
└── versions/                              # All versions stored here
    ├── bat/v0.25.0/bat
    ├── helm/v3.18.3/helm
    └── k0s/v1.33.2+k0s.0/k0s
```

**Benefits:**
- ✅ Clean bin directory with only symlinks
- ✅ All versions organized under `versions/` subdirectory
- ✅ Easy to see which binaries are installed
- ✅ Version history preserved and organized by project
- ✅ No naming conflicts between binaries and version directories

## Implementation

### New FileConfig Field

A new boolean field `UseVersionsSubdirectory` has been added to `FileConfig`:

```go
type FileConfig struct {
    // ... existing fields
    
    // Enhanced directory structure control
    UseVersionsSubdirectory bool `json:"use_versions_subdirectory"` // Use versions/{ProjectName}/ subdirectory pattern (default: false for backward compatibility)
    
    // ... other fields
}
```

### Path Construction Logic

The library now uses different path construction based on the `UseVersionsSubdirectory` setting:

#### Legacy Pattern (`UseVersionsSubdirectory: false`)
- **Version Directory:** `BaseBinaryDirectory/{VersionedDirectoryName}/{version}/`
- **Binary Path:** `BaseBinaryDirectory/{VersionedDirectoryName}/{version}/{BinaryName}`
- **Symlink Target:** `{VersionedDirectoryName}/{version}/{BinaryName}`

#### New Pattern (`UseVersionsSubdirectory: true`)
- **Version Directory:** `BaseBinaryDirectory/versions/{ProjectName}/{version}/`
- **Binary Path:** `BaseBinaryDirectory/versions/{ProjectName}/{version}/{BinaryName}`
- **Symlink Target:** `versions/{ProjectName}/{version}/{BinaryName}`

### Helper Functions

New helper functions provide consistent path construction:

```go
// GetVersionedDirectoryPath returns the path to the versioned directory based on configuration
func GetVersionedDirectoryPath(config FileConfig, version string) string

// GetVersionedBinaryPath returns the full path to the binary in the versioned directory
func GetVersionedBinaryPath(config FileConfig, version string) string

// GetSymlinkTargetPath returns the relative path from symlink to target for proper symlink creation
func GetSymlinkTargetPath(config FileConfig, version string) string
```

## Usage

### Basic Configuration

```go
config := fileUtils.FileConfig{
    BaseBinaryDirectory:     "/home/user/.local/bin",
    VersionedDirectoryName:  "versions",
    SourceBinaryName:        "helm",
    BinaryName:              "helm",
    ProjectName:             "helm",
    UseVersionsSubdirectory: true,  // Enable new pattern
    CreateLocalSymlink:      true,
    CreateGlobalSymlink:     false,
    SourceArchivePath:       "/tmp/helm-latest.tar.gz",
    IsDirectBinary:          false,
}
```

### With CDN Configuration

```go
config := fileUtils.FileConfig{
    BaseBinaryDirectory:     "/home/user/.local/bin",
    BinaryName:              "helm",
    ProjectName:             "helm",
    UseVersionsSubdirectory: true,
    CreateLocalSymlink:      true,
    IsDirectBinary:          false,
}

helmConfig := release.GetHelmCDNConfig()
githubRelease := release.NewGithubReleaseWithCDNConfig("helm/helm", config, helmConfig)

err := githubRelease.DownloadLatestRelease()
if err != nil {
    log.Fatalf("Download failed: %v", err)
}

err = githubRelease.InstallLatestRelease()
if err != nil {
    log.Fatalf("Installation failed: %v", err)
}
```

### Multiple Binaries Example

```go
binaries := []struct {
    name        string
    projectName string
    repository  string
}{
    {"helm", "helm", "helm/helm"},
    {"kubectl", "kubernetes", "kubernetes/kubernetes"},
    {"k0s", "k0s", "k0sproject/k0s"},
    {"bat", "bat", "sharkdp/bat"},
}

for _, binary := range binaries {
    config := fileUtils.FileConfig{
        BaseBinaryDirectory:     "/home/user/.local/bin",
        BinaryName:              binary.name,
        ProjectName:             binary.projectName,
        UseVersionsSubdirectory: true,
        CreateLocalSymlink:      true,
        IsDirectBinary:          false,
    }
    
    githubRelease := release.NewGithubRelease(binary.repository, config)
    // Download and install...
}
```

## Backward Compatibility

The new feature maintains full backward compatibility:

### Default Behavior
- `UseVersionsSubdirectory` defaults to `false`
- Existing configurations continue to work unchanged
- No breaking changes to the API

### Migration Path
Users can opt into the new pattern by:
1. Setting `UseVersionsSubdirectory: true` in their configuration
2. Ensuring `ProjectName` is set (falls back to `BinaryName` if empty)
3. Both patterns can coexist in the same directory

### Legacy Configuration (Still Works)
```go
config := fileUtils.FileConfig{
    BaseBinaryDirectory:    "/home/user/.local/bin",
    VersionedDirectoryName: "versions",
    BinaryName:             "helm",
    CreateLocalSymlink:     true,
    // UseVersionsSubdirectory: false (default)
}
```

## Technical Details

### ProjectName Fallback
If `ProjectName` is not set when using the new pattern, the library falls back to `BinaryName`:

```go
projectName := config.ProjectName
if projectName == "" {
    projectName = config.BinaryName
}
```

### Symlink Handling
The library correctly handles relative symlinks in both patterns:
- Legacy: `versions/v1.0.0/binary`
- New: `versions/project/v1.0.0/binary`

### Path Resolution
Updated path resolution logic handles both absolute and relative symlink targets correctly.

## Testing

Comprehensive tests verify:
- Path construction for both patterns
- Installation process with new directory structure
- Backward compatibility with legacy configurations
- Symlink creation and resolution
- Multiple binary scenarios

Run tests with:
```bash
go test ./pkg/fileUtils -run TestVersionsSubdirectory -v
```

## Examples

See `examples/versions_subdirectory_example.go.example` for a complete demonstration of the new pattern.

## Migration Recommendations

### For New Projects
Use the new pattern for cleaner directory structure:
```go
config.UseVersionsSubdirectory = true
```

### For Existing Projects
- Keep existing configuration for stability
- Optionally migrate to new pattern for cleaner structure
- Both patterns can coexist during transition

### For Library Maintainers
- The new pattern works with all existing strategies (GitHub, GitLab, CDN, Hybrid)
- No changes required to download or installation logic
- Symlink creation automatically uses correct paths

## Benefits Summary

1. **Clean Directories**: Bin directories contain only symlinks, not version directories
2. **Better Organization**: All versions stored under dedicated `versions/` subdirectory
3. **No Conflicts**: Eliminates naming conflicts between symlinks and directories
4. **Backward Compatible**: Existing configurations continue to work unchanged
5. **Future Proof**: Provides foundation for additional directory structure enhancements
6. **User Friendly**: Matches patterns used by other successful tools like container-cli
