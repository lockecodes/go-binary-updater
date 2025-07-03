# Go Binary Updater

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/locke-codes/go-binary-updater)](https://goreportcard.com/report/gitlab.com/locke-codes/go-binary-updater)

A powerful Go library that makes it easy to automatically update CLI binaries from GitHub and GitLab releases. Perfect for implementing self-updating CLI tools with minimal effort.

## 🚀 Features

- **Multi-Platform Support**: Automatically detects and downloads the correct binary for your OS and architecture
- **Flexible Asset Matching**: Supports various naming conventions (k0s, kubectl, helm, terraform, etc.)
- **Direct Binary Support**: Handles both archived and direct binary downloads
- **Dual Provider Support**: Works with both GitHub and GitLab releases
- **Interface-Based Design**: Easily switch between providers or add new ones
- **Versioned Installation**: Maintains multiple versions with automatic symlink management
- **Authentication Support**: Supports GitHub tokens and GitLab access tokens
- **Comprehensive Testing**: Extensive test suite with mock servers
- **Production Ready**: Used in production environments with robust error handling

## 📦 Installation

```bash
go get gitlab.com/locke-codes/go-binary-updater
```

## 🎯 Quick Start

### GitHub Releases

```go
package main

import (
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

    // GitHub releases
    githubRelease := release.NewGithubRelease("owner/repo", config)

    err := githubRelease.DownloadLatestRelease()
    if err != nil {
        log.Fatal(err)
    }

    err = githubRelease.InstallLatestRelease()
    if err != nil {
        log.Fatal(err)
    }
}
```

### GitLab Releases

```go
// GitLab releases (using project ID)
gitlabRelease := release.NewGitlabRelease("12345678", config)

err := gitlabRelease.DownloadLatestRelease()
if err != nil {
    log.Fatal(err)
}

err = gitlabRelease.InstallLatestRelease()
if err != nil {
    log.Fatal(err)
}
```

### k0s Direct Binary Example

```go
// k0s uses direct binaries with names like: k0s-v1.33.2+k0s.0-amd64
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "k0s",
    BinaryName:             "k0s",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/k0s-latest",

    // k0s-specific configuration
    IsDirectBinary:         true,    // k0s releases direct binaries
    ProjectName:            "k0s",   // For intelligent asset matching
    AssetMatchingStrategy:  "flexible",
}

githubRelease := release.NewGithubRelease("k0sproject/k0s", config)

err := githubRelease.DownloadLatestRelease()
if err != nil {
    log.Fatal(err)
}

err = githubRelease.InstallLatestRelease()
if err != nil {
    log.Fatal(err)
}
```

## 🔧 How It Works

The library follows a simple but powerful workflow:

1. **Query Repository**: Fetches all releases from GitHub or GitLab API
2. **Select Latest**: Identifies the most recent release by date
3. **Platform Detection**: Automatically detects your OS and architecture
4. **Asset Selection**: Finds the matching binary asset (e.g., `Linux_x86_64.tar.gz`)
5. **Download**: Downloads the release archive to a temporary location
6. **Extract**: Extracts the binary from the archive
7. **Install**: Moves binary to versioned directory (e.g., `/usr/local/bin/versions/v1.2.3/`)
8. **Symlink**: Creates or updates symlink to the latest version

## 🏗️ Architecture

### Release Interface

The library uses an interface-based design for maximum flexibility:

```go
type Release interface {
    GetLatestRelease() error
    DownloadLatestRelease() error
    InstallLatestRelease() error
}
```

This allows you to:
- Switch between GitHub and GitLab seamlessly
- Mock providers for testing
- Add custom release providers
- Use polymorphic code patterns

### Supported Providers

| Provider | Repository Format | Authentication | Rate Limits |
|----------|------------------|----------------|-------------|
| **GitHub** | `owner/repo` | GitHub Token (optional) | 60/hour (unauth), 5,000/hour (auth) |
| **GitLab** | Project ID (numeric) | GitLab Token (planned) | 2,000/min (public) |

## ⚙️ Configuration

### FileConfig Options

```go
type FileConfig struct {
    VersionedDirectoryName string  // Directory name for versions (default: "versions")
    SourceBinaryName       string  // Binary name in the archive
    BinaryName             string  // Installed binary name
    CreateGlobalSymlink    bool    // Create symlink in PATH
    BaseBinaryDirectory    string  // Base installation directory
    SourceArchivePath      string  // Download location
}
```

### Example Configurations

#### Basic CLI Tool
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "mycli",
    BinaryName:             "mycli",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/usr/local/bin",
    SourceArchivePath:      "/tmp/mycli.tar.gz",
}
```

#### User-Local Installation
```go
config := fileUtils.FileConfig{
    VersionedDirectoryName: "versions",
    SourceBinaryName:       "tool",
    BinaryName:             "tool",
    CreateGlobalSymlink:    true,
    BaseBinaryDirectory:    "/home/user/.local/bin",
    SourceArchivePath:      "/tmp/tool.tar.gz",
}
```

## 🔐 Authentication

### GitHub Authentication

```go
// With GitHub token for private repos or higher rate limits
token := os.Getenv("GITHUB_TOKEN")
githubRelease := release.NewGithubReleaseWithToken("owner/repo", token, config)
```

### GitLab Authentication

GitLab authentication support is planned for future releases. Currently works with public projects.

## 🌍 Platform Support

The library automatically detects your platform and selects the appropriate binary:

### Supported Operating Systems
- Linux
- macOS (Darwin)
- Windows

### Supported Architectures
- amd64 (x86_64)
- arm64
- 386 (i386)

### Asset Naming Convention

Your release assets should follow this naming pattern:
```
{BinaryName}-{OS}_{ARCH}.tar.gz
```

Examples:
- `myapp-Linux_x86_64.tar.gz`
- `myapp-Darwin_arm64.tar.gz`
- `myapp-Windows_x86_64.zip`

## 📚 Documentation

### Provider-Specific Guides
- [GitHub Usage Guide](docs/GITHUB_USAGE.md) - Complete GitHub integration documentation
- [GitLab Usage Guide](docs/GITLAB_USAGE.md) - Complete GitLab integration documentation
- [Flexible Asset Matching](docs/FLEXIBLE_ASSET_MATCHING.md) - Guide for different project naming conventions

### Examples
- [GitHub Examples](examples/github_example.go) - Working GitHub examples
- [GitLab Examples](examples/gitlab_example.go) - Working GitLab examples
- [k0s Example](examples/k0s_example.go) - Direct binary download example

## 🧪 Testing

Run the complete test suite:

```bash
# All tests
go test ./...

# Release package tests only
go test ./pkg/release -v

# Specific provider tests
go test ./pkg/release -v -run="TestGithub"
go test ./pkg/release -v -run="TestGitlab"
```

## 🔍 Troubleshooting

### Common Issues

#### Asset Not Found
```
Error: no suitable asset found for current platform
```
**Solution**: Ensure your release assets follow the `{OS}_{ARCH}` naming convention.

#### Authentication Required
```
Error: unexpected status code: 403
```
**Solution**: Use authentication tokens for private repositories or to avoid rate limits.

#### Invalid Repository Format
```
Error: invalid repository format
```
**Solution**:
- GitHub: Use `owner/repo` format
- GitLab: Use numeric project ID (found in project settings)

### Debug Tips

1. **Enable Verbose Logging**: The library logs API requests and responses
2. **Check Asset Names**: Verify your release assets match the expected naming pattern
3. **Test with Public Repos**: Start with public repositories before using private ones
4. **Verify Project IDs**: For GitLab, ensure you're using the numeric project ID, not the project path

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://gitlab.com/locke-codes/go-binary-updater.git
cd go-binary-updater

# Run tests
go test ./...

# Run linting
golangci-lint run
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the need for simple, reliable binary updates in CLI tools
- Built with Go's excellent standard library and minimal external dependencies
- Tested against real GitHub and GitLab APIs for reliability

## 📞 Support

- **Issues**: [GitLab Issues](https://gitlab.com/locke-codes/go-binary-updater/-/issues)
- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)

---

**Made with ❤️ for the Go community**
