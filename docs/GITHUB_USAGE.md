# GitHub Release Functionality

This document describes how to use the GitHub release functionality in the go-binary-updater library.

## Overview

The GitHub release functionality allows you to:
- Fetch the latest release information from GitHub repositories
- Download platform-specific binaries from GitHub releases
- Install and manage binary versions with automatic symlink creation

## Basic Usage

### Creating a GitHub Release Instance

```go
import (
    "gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
    "gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

// Configure file handling
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "myapp",
    BinaryName:             "myapp",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
}

// Create GitHub release instance
githubRelease := release.NewGithubRelease("owner/repo", config)
```

### With Authentication (Recommended)

```go
// For private repositories or higher rate limits
token := os.Getenv("GITHUB_TOKEN")
githubRelease := release.NewGithubReleaseWithToken("owner/repo", token, config)
```

## API Methods

### GetLatestRelease()

Fetches the latest release information from GitHub and populates the `Version` and `ReleaseLink` fields.

```go
err := githubRelease.GetLatestRelease()
if err != nil {
    log.Fatalf("Error getting latest release: %v", err)
}
fmt.Printf("Latest version: %s\n", githubRelease.Version)
fmt.Printf("Download URL: %s\n", githubRelease.ReleaseLink)
```

### DownloadLatestRelease()

Downloads the latest release binary to the configured path. This method automatically calls `GetLatestRelease()` first.

```go
err := githubRelease.DownloadLatestRelease()
if err != nil {
    log.Fatalf("Error downloading: %v", err)
}
```

### InstallLatestRelease()

Extracts and installs the binary with proper versioning and symlink management.

```go
err := githubRelease.InstallLatestRelease()
if err != nil {
    log.Fatalf("Error installing: %v", err)
}
```

## Platform Support

The library automatically detects your platform and selects the appropriate binary asset from the GitHub release. It looks for assets with names containing:

- `{OS}_{ARCH}` pattern (e.g., `Linux_x86_64`, `Darwin_arm64`, `Windows_x86_64`)
- Supported architectures: `amd64` → `x86_64`, `arm64` → `arm64`

## Authentication

### GitHub Token

For private repositories or to avoid rate limiting, set up a GitHub token:

1. Create a Personal Access Token in GitHub Settings
2. Set the `GITHUB_TOKEN` environment variable
3. Use `NewGithubReleaseWithToken()` constructor

```bash
export GITHUB_TOKEN=your_token_here
```

### Rate Limiting

- **Unauthenticated**: 60 requests per hour per IP
- **Authenticated**: 5,000 requests per hour per user

## Error Handling

Common errors and their meanings:

- `"repository cannot be empty"` - Repository field is empty
- `"invalid repository format"` - Repository should be in "owner/repo" format
- `"no suitable asset found for current platform"` - No matching binary for your OS/architecture
- `"unexpected status code from GitHub: 404"` - Repository not found or private
- `"unexpected status code from GitHub: 403"` - Rate limited or authentication required

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
    "gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
    config := fileUtils.FileConfig{
        VersionedDirectoryName: "versions",
        SourceBinaryName:       "myapp",
        BinaryName:             "myapp",
        CreateGlobalSymlink:    true,
        BaseBinaryDirectory:    "/home/user/.local/bin",
        SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
    }

    // Create GitHub release instance
    var githubRelease *release.GithubRelease
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        githubRelease = release.NewGithubReleaseWithToken("owner/repo", token, config)
    } else {
        githubRelease = release.NewGithubRelease("owner/repo", config)
    }

    // Get, download, and install in one go
    err := githubRelease.DownloadLatestRelease()
    if err != nil {
        log.Fatalf("Error downloading: %v", err)
    }

    err = githubRelease.InstallLatestRelease()
    if err != nil {
        log.Fatalf("Error installing: %v", err)
    }

    fmt.Printf("Successfully installed %s version %s\n", 
        config.BinaryName, githubRelease.Version)
}
```

## Interface Compatibility

The `GithubRelease` struct implements the `Release` interface, making it interchangeable with other release providers like GitLab:

```go
var releaseProvider release.Release

// Can be either GitHub or GitLab
releaseProvider = release.NewGithubRelease("owner/repo", config)
// or
releaseProvider = release.NewGitlabRelease("project-id", config)

// Same interface for both
err := releaseProvider.GetLatestRelease()
err = releaseProvider.DownloadLatestRelease()
err = releaseProvider.InstallLatestRelease()
```

## Testing

The implementation includes comprehensive tests with mock servers. Run tests with:

```bash
go test ./pkg/release -v
```

For GitHub-specific tests only:

```bash
go test ./pkg/release -v -run="TestGithub"
```
