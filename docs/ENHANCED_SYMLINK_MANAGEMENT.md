# Enhanced Symlink Management

The go-binary-updater library has been enhanced with refined symlink control while preserving its core value proposition: **automatic symlink management**. This document explains the enhanced symlink functionality, path resolution API, and installation information features.

## Core Principle: Symlink-First Approach

**Symlinks remain the default and primary behavior** because they are one of the main benefits of this library. The enhancements provide flexibility for edge cases while maintaining the symlink-first approach that makes binary management seamless.

## Enhanced Symlink Control

### New Configuration Options

The `FileConfig` struct now includes enhanced symlink control:

```go
type FileConfig struct {
    // Existing fields...
    CreateGlobalSymlink    bool   `json:"create_global_symlink"`    // Create global symlink in /usr/local/bin (requires sudo)
    
    // New enhanced symlink control
    CreateLocalSymlink     bool   `json:"create_local_symlink"`     // Create local symlink in BaseBinaryDirectory (default: true)
    
    // Other enhanced fields...
}
```

### Default Configuration

Use `DefaultFileConfig()` to get sensible defaults that preserve the symlink-first approach:

```go
config := fileUtils.DefaultFileConfig()
// config.CreateLocalSymlink = true   (preserves core value proposition)
// config.CreateGlobalSymlink = false (requires sudo, so disabled by default)
// config.AssetMatchingStrategy = "flexible"
// config.IsDirectBinary = false
```

### Symlink Configuration Examples

#### 1. Default Configuration (Recommended)
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateLocalSymlink:     true,  // Core value proposition: easy access via symlink
    CreateGlobalSymlink:    false, // No sudo required
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp.tar.gz",
}
```

#### 2. System-Wide Installation
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "systemtool",
    BinaryName:             "systemtool",
    CreateLocalSymlink:     true,  // Local access
    CreateGlobalSymlink:    true,  // System-wide access (requires sudo)
    BaseBinaryDirectory:    "/opt/systemtool",
    SourceArchivePath:      "/tmp/systemtool.tar.gz",
}
```

#### 3. Restricted Environments (No Symlinks)
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "restrictedapp",
    BinaryName:             "restrictedapp",
    CreateLocalSymlink:     false, // Symlinks not allowed
    CreateGlobalSymlink:    false, // Symlinks not allowed
    BaseBinaryDirectory:    "/opt/restrictedapp",
    SourceArchivePath:      "/tmp/restrictedapp.tar.gz",
}
```

#### 4. CI/CD Environments
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "ci-tool",
    BinaryName:             "ci-tool",
    CreateLocalSymlink:     true,  // Easy access in CI
    CreateGlobalSymlink:    false, // No sudo in CI
    BaseBinaryDirectory:    "/tmp/ci-tools",
    SourceArchivePath:      "/tmp/ci-tool.tar.gz",
}
```

## Graceful Symlink Fallback

The library now handles symlink creation failures gracefully:

### TryUpdateSymlink Function

```go
// TryUpdateSymlink attempts to create a symlink with graceful fallback
success := fileUtils.TryUpdateSymlink(targetPath, symlinkPath)
if success {
    fmt.Println("Symlink created successfully")
} else {
    fmt.Println("Symlink creation failed, but binary is still available")
}
```

### Installation Behavior

When symlink creation fails:
1. **Warning is logged** (not an error)
2. **Installation continues successfully**
3. **Binary remains accessible** at versioned path
4. **User is informed** of both symlink status and binary location

Example output:
```
Creating local symlink...
Warning: Failed to create symlink /home/user/.local/bin/myapp -> /home/user/.local/bin/versions/v1.0.0/myapp: permission denied
Binary is still available at: /home/user/.local/bin/versions/v1.0.0/myapp
Installation successful!
Binary installed at: /home/user/.local/bin/versions/v1.0.0/myapp
```

## Enhanced Path Resolution API

### GetInstalledBinaryPath Method

Returns the preferred path to the installed binary:

```go
// For any Release implementation (GitHub or GitLab)
binaryPath, err := release.GetInstalledBinaryPath()
if err != nil {
    log.Fatalf("Failed to get binary path: %v", err)
}

fmt.Printf("Binary available at: %s\n", binaryPath)
```

**Path Resolution Logic:**
1. **Prefers symlink path** if symlink exists and points to correct version
2. **Falls back to versioned path** if symlink doesn't exist or is incorrect
3. **Returns error** if binary not found at either location

### GetInstallationInfo Method

Returns comprehensive installation information:

```go
info, err := release.GetInstallationInfo()
if err != nil {
    log.Fatalf("Failed to get installation info: %v", err)
}

fmt.Printf("Installation Details:\n")
fmt.Printf("  Version: %s\n", info.Version)
fmt.Printf("  Type: %s\n", info.InstallationType)
fmt.Printf("  Binary Path: %s\n", info.BinaryPath)
fmt.Printf("  Symlink Status: %s\n", info.SymlinkStatus)
if info.LocalSymlinkCreated {
    fmt.Printf("  Local Symlink: %s\n", info.LocalSymlinkPath)
}
if info.GlobalSymlinkNeeded {
    fmt.Printf("  Global Symlink: %s (requires sudo)\n", info.GlobalSymlinkPath)
}
```

### InstallationInfo Structure

```go
type InstallationInfo struct {
    BinaryPath          string `json:"binary_path"`           // Preferred path (symlink or versioned)
    Version             string `json:"version"`               // Installed version
    InstallationType    string `json:"installation_type"`     // "direct_binary" or "extracted_archive"
    SymlinkStatus       string `json:"symlink_status"`        // "created", "failed", "disabled", "not_attempted"
    LocalSymlinkPath    string `json:"local_symlink_path"`    // Path to local symlink
    GlobalSymlinkPath   string `json:"global_symlink_path"`   // Path to global symlink
    VersionedPath       string `json:"versioned_path"`        // Path in versioned directory
    LocalSymlinkCreated bool   `json:"local_symlink_created"` // Whether local symlink exists
    GlobalSymlinkNeeded bool   `json:"global_symlink_needed"` // Whether global symlink was requested
}
```

## Direct Binary Support Enhancements

### Robust Direct Binary Handling

The `IsDirectBinary` functionality has been enhanced:

```go
config := fileUtils.FileConfig{
    // ... other fields
    IsDirectBinary: true, // Completely bypasses archive extraction
}
```

**Enhanced Validation:**
- **Explicit validation** prevents extraction logic from running on direct binaries
- **Clear error messages** distinguish between direct binary and archive handling
- **Consistent behavior** across both installation workflows

### Installation Type Detection

The system automatically detects and reports installation type:

```go
info, _ := release.GetInstallationInfo()
switch info.InstallationType {
case "direct_binary":
    fmt.Println("Installed from direct binary download")
case "extracted_archive":
    fmt.Println("Installed from extracted archive")
}
```

## Backward Compatibility

### Automatic Defaults Application

Old configurations automatically get enhanced defaults:

```go
// Old configuration (both symlink options false)
oldConfig := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "oldapp",
    BinaryName:             "oldapp",
    CreateLocalSymlink:     false, // Old default
    CreateGlobalSymlink:    false, // Old default
    // ... other fields
}

// The installation functions automatically apply defaults:
// - If both symlink options are false, CreateLocalSymlink is set to true
// - This preserves the symlink-first approach for old configurations
```

### Migration Path

**No changes required** for existing code:
- All existing configurations continue to work
- Default behavior still creates local symlinks
- Enhanced features are opt-in

**Optional enhancements:**
```go
// Before (still works)
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateGlobalSymlink:    false,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp.tar.gz",
}

// After (enhanced, optional)
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateLocalSymlink:     true,  // Explicit (was implicit)
    CreateGlobalSymlink:    false, // Same as before
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp.tar.gz",
}
```

## Cross-Platform Support

The enhanced symlink functionality works across platforms:

### Linux and macOS
- **Full symlink support** with graceful fallback
- **Permission handling** for restricted environments
- **Path resolution** works with both symlinks and direct paths

### Windows
- **Symlink creation** attempted (requires appropriate permissions)
- **Graceful fallback** to direct paths when symlinks fail
- **Consistent API** across all platforms

## Usage Examples

### Complete k0s Example with Enhanced Features

```go
package main

import (
    "fmt"
    "log"
    "gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
    "gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
    // k0s configuration with enhanced symlink control
    config := fileUtils.FileConfig{
        VersionedDirectoryName: "versions",
        SourceBinaryName:       "k0s",
        BinaryName:             "k0s",
        CreateLocalSymlink:     true,  // Core value: easy access
        CreateGlobalSymlink:    true,  // System-wide access
        BaseBinaryDirectory:    "/home/user/.local/bin",
        SourceArchivePath:      "/tmp/k0s-latest",
        IsDirectBinary:         true,  // k0s releases direct binaries
        ProjectName:            "k0s",
        AssetMatchingStrategy:  "flexible",
    }

    githubRelease := release.NewGithubRelease("k0sproject/k0s", config)

    // Download and install
    err := githubRelease.GetLatestRelease()
    if err != nil {
        log.Fatalf("Failed to get latest release: %v", err)
    }

    err = githubRelease.DownloadLatestRelease()
    if err != nil {
        log.Fatalf("Failed to download: %v", err)
    }

    err = githubRelease.InstallLatestRelease()
    if err != nil {
        log.Fatalf("Failed to install: %v", err)
    }

    // Use enhanced path resolution API
    binaryPath, err := githubRelease.GetInstalledBinaryPath()
    if err != nil {
        log.Printf("Warning: %v", err)
    } else {
        fmt.Printf("k0s available at: %s\n", binaryPath)
    }

    // Get comprehensive installation information
    info, err := githubRelease.GetInstallationInfo()
    if err != nil {
        log.Printf("Warning: %v", err)
    } else {
        fmt.Printf("Installation Details:\n")
        fmt.Printf("  Version: %s\n", info.Version)
        fmt.Printf("  Type: %s\n", info.InstallationType)
        fmt.Printf("  Binary Path: %s\n", info.BinaryPath)
        fmt.Printf("  Symlink Status: %s\n", info.SymlinkStatus)
        
        if info.LocalSymlinkCreated {
            fmt.Printf("  Local Symlink: %s\n", info.LocalSymlinkPath)
        }
        
        if info.GlobalSymlinkNeeded {
            fmt.Printf("  Global Symlink: %s (requires sudo)\n", info.GlobalSymlinkPath)
        }
    }
}
```

## Best Practices

### 1. Preserve Symlink-First Approach
- **Always enable local symlinks** unless there's a specific reason not to
- **Use `CreateLocalSymlink: true`** as the default
- **Only disable symlinks** in restricted environments

### 2. Handle Symlink Failures Gracefully
- **Don't treat symlink failures as fatal errors**
- **Always provide fallback paths** to users
- **Use the enhanced path resolution API** for consistent access

### 3. Provide Clear User Feedback
- **Show both symlink and versioned paths** in output
- **Explain symlink status** (created, failed, disabled)
- **Provide instructions** for global symlink creation

### 4. Use Enhanced APIs
- **Use `GetInstalledBinaryPath()`** instead of manually constructing paths
- **Use `GetInstallationInfo()`** for comprehensive status information
- **Check installation type** to understand the deployment method

## Summary

The enhanced symlink management preserves the core value proposition of automatic symlink creation while providing:

1. **Refined Control**: Separate local and global symlink options
2. **Graceful Fallback**: Installation continues even when symlinks fail
3. **Enhanced APIs**: Easy path resolution and installation information
4. **Robust Direct Binary Support**: Complete bypass of extraction logic
5. **Backward Compatibility**: Existing code continues to work unchanged
6. **Cross-Platform Support**: Consistent behavior across operating systems

The symlink-first approach remains the default, ensuring that the library continues to provide its core benefit of seamless binary management while accommodating edge cases where symlinks cannot be created.
