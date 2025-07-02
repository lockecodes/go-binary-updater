# GitLab Release Functionality

This document describes how to use the GitLab release functionality in the go-binary-updater library.

## Overview

The GitLab release functionality allows you to:
- Fetch the latest release information from GitLab projects
- Download platform-specific binaries from GitLab releases
- Install and manage binary versions with automatic symlink creation
- Support both public and private GitLab projects

## Basic Usage

### Creating a GitLab Release Instance

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

// Create GitLab release instance using project ID
gitlabRelease := release.NewGitlabRelease("12345678", config)
```

### Finding Your GitLab Project ID

The GitLab project ID is a numeric identifier found in several places:

1. **Project Settings**: Go to your project → Settings → General
2. **Project URL**: Visible in the project's main page below the project name
3. **API Response**: Available in GitLab API responses as the `id` field

Example: For project `https://gitlab.com/owner/repo`, the project ID might be `12345678`.

## API Methods

### GetLatestRelease()

Fetches all releases from GitLab, sorts them by release date, and selects the most recent one. Populates the `Version` and `ReleaseLink` fields.

```go
err := gitlabRelease.GetLatestRelease()
if err != nil {
    log.Fatalf("Error getting latest release: %v", err)
}
fmt.Printf("Latest version: %s\n", gitlabRelease.Version)
fmt.Printf("Download URL: %s\n", gitlabRelease.ReleaseLink)
```

### DownloadLatestRelease()

Downloads the latest release binary to the configured path. This method automatically calls `GetLatestRelease()` first.

```go
err := gitlabRelease.DownloadLatestRelease()
if err != nil {
    log.Fatalf("Error downloading: %v", err)
}
```

### InstallLatestRelease()

Extracts and installs the binary with proper versioning and symlink management.

```go
err := gitlabRelease.InstallLatestRelease()
if err != nil {
    log.Fatalf("Error installing: %v", err)
}
```

## Platform Support

The library automatically detects your platform and selects the appropriate binary asset from the GitLab release. It looks for assets with names containing:

- `{OS}_{ARCH}` pattern (e.g., `Linux_x86_64`, `Darwin_arm64`, `Windows_x86_64`)
- Supported architectures: `amd64` → `x86_64`, `arm64` → `arm64`

## Authentication

### GitLab Token

For private projects or to avoid rate limiting, you can use GitLab Personal Access Tokens or Project Access Tokens:

1. **Personal Access Token**: Go to GitLab → User Settings → Access Tokens
2. **Project Access Token**: Go to Project → Settings → Access Tokens
3. Required scopes: `read_api` for public projects, `read_repository` for private projects

```bash
export GITLAB_TOKEN=your_token_here
```

**Note**: The current implementation doesn't include built-in token authentication, but you can extend the HTTP client to include authentication headers if needed.

### Rate Limiting

- **Public GitLab.com**: 2,000 requests per minute per IP
- **Authenticated**: Higher limits based on your GitLab plan
- **Self-hosted GitLab**: Depends on instance configuration

## Error Handling

Common errors and their meanings:

- `"invalid project ID"` - Project ID must be a positive integer
- `"no GitLab releases found for project ID"` - Project has no releases or is private/inaccessible
- `"unexpected status code from GitLab: 404"` - Project not found or private
- `"unexpected status code from GitLab: 403"` - Access denied (authentication may be required)
- `"could not find a valid release to download"` - No suitable asset found for current platform

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    
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

    // Create GitLab release instance with project ID
    gitlabRelease := release.NewGitlabRelease("12345678", config)

    // Get, download, and install in one go
    err := gitlabRelease.DownloadLatestRelease()
    if err != nil {
        log.Fatalf("Error downloading: %v", err)
    }

    err = gitlabRelease.InstallLatestRelease()
    if err != nil {
        log.Fatalf("Error installing: %v", err)
    }

    fmt.Printf("Successfully installed %s version %s\n", 
        config.BinaryName, gitlabRelease.Version)
}
```

## Interface Compatibility

The `GitLabRelease` struct implements the `Release` interface, making it interchangeable with other release providers like GitHub:

```go
var releaseProvider release.Release

// Can be either GitLab or GitHub
releaseProvider = release.NewGitlabRelease("12345678", config)
// or
releaseProvider = release.NewGithubRelease("owner/repo", config)

// Same interface for both
err := releaseProvider.GetLatestRelease()
err = releaseProvider.DownloadLatestRelease()
err = releaseProvider.InstallLatestRelease()
```

## GitLab API Integration

### API Version
- Uses GitLab API v4: `https://gitlab.com/api/v4/projects/{id}/releases`
- Supports both GitLab.com and self-hosted GitLab instances

### Release Selection Logic
1. Fetches all releases for the project
2. Sorts releases by `released_at` timestamp (most recent first)
3. Selects the first (latest) release
4. Searches release assets for platform-specific downloads

### Asset Structure
GitLab releases use a different asset structure than GitHub:
- Assets are stored in `assets.links[]` array
- Each link has `name`, `url`, and `direct_asset_url` fields
- The library uses `direct_asset_url` for downloads

## Testing

The implementation includes comprehensive tests with mock servers. Run tests with:

```bash
go test ./pkg/release -v
```

For GitLab-specific tests only:

```bash
go test ./pkg/release -v -run="TestGitlab"
```

## Self-Hosted GitLab

For self-hosted GitLab instances, you can override the base URL for testing:

```go
gitlabRelease := release.NewGitlabRelease("12345678", config)
gitlabRelease.BaseURL = "https://gitlab.example.com/api/v4/projects"
```

## Troubleshooting

### Common Issues

1. **Project ID Format**: Ensure you're using the numeric project ID, not the project path
2. **Private Projects**: May require authentication (consider extending with token support)
3. **Asset Naming**: Ensure your release assets follow the `OS_ARCH` naming convention
4. **Network Issues**: Check connectivity to GitLab.com or your self-hosted instance

### Debug Tips

- Enable verbose logging to see API requests
- Verify project ID by visiting `https://gitlab.com/api/v4/projects/{id}` in your browser
- Check release assets in GitLab UI to verify naming conventions
- Test with public projects first before using private ones
