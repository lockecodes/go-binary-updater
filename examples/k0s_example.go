package main

import (
	"fmt"
	"log"
	"os"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
	// Example configuration for downloading and installing k0s from GitHub
	// k0s uses a different naming convention: k0s-v1.33.2+k0s.0-amd64
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "k0s",           // Binary name in the release (for direct binaries, this is the final name)
		BinaryName:             "k0s",           // Name to use for installed binary
		CreateLocalSymlink:     true,            // Create local symlink (default, core value proposition)
		CreateGlobalSymlink:    true,            // Create global symlink in PATH (requires sudo)
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/k0s-latest", // Download location (no extension for direct binary)

		// Enhanced configuration for k0s
		IsDirectBinary:         true,            // k0s releases are direct binaries, not archives
		ProjectName:            "k0s",           // Project name for asset matching
		AssetMatchingStrategy:  "flexible",     // Use flexible matching for k0s naming pattern
	}

	// Create GitHub release instance for k0s
	githubRelease := release.NewGithubRelease("k0sproject/k0s", config)

	fmt.Println("Fetching latest k0s release...")
	err := githubRelease.GetLatestRelease()
	if err != nil {
		log.Fatalf("Error getting latest k0s release: %v", err)
	}

	fmt.Printf("Latest k0s version: %s\n", githubRelease.Version)
	fmt.Printf("Download URL: %s\n", githubRelease.ReleaseLink)

	fmt.Println("\nDownloading latest k0s release...")
	err = githubRelease.DownloadLatestRelease()
	if err != nil {
		log.Fatalf("Error downloading k0s: %v", err)
	}

	fmt.Println("Installing k0s...")
	err = githubRelease.InstallLatestRelease()
	if err != nil {
		log.Fatalf("Error installing k0s: %v", err)
	}

	fmt.Printf("Successfully installed k0s version %s!\n", githubRelease.Version)

	// Demonstrate the enhanced path resolution API
	binaryPath, err := githubRelease.GetInstalledBinaryPath()
	if err != nil {
		log.Printf("Warning: Could not get binary path: %v", err)
	} else {
		fmt.Printf("k0s is available at: %s\n", binaryPath)
	}

	// Demonstrate the installation information API
	info, err := githubRelease.GetInstallationInfo()
	if err != nil {
		log.Printf("Warning: Could not get installation info: %v", err)
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

	fmt.Println("You can now use k0s from your command line.")
}

// Example function showing how to configure for other direct binary projects
func exampleOtherDirectBinaries() {
	// kubectl example
	kubectlConfig := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "kubectl",
		BinaryName:             "kubectl",
		CreateGlobalSymlink:    true,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/kubectl-latest",
		IsDirectBinary:         true,
		ProjectName:            "kubectl",
		AssetMatchingStrategy:  "flexible",
	}

	// Note: kubectl is distributed differently, this is just an example
	fmt.Println("kubectl configuration example:")
	fmt.Printf("Project: %s, Direct Binary: %t\n", kubectlConfig.ProjectName, kubectlConfig.IsDirectBinary)

	// helm example (archived binary)
	helmConfig := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "helm",
		BinaryName:             "helm",
		CreateGlobalSymlink:    true,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/helm-latest.tar.gz",
		IsDirectBinary:         false, // helm releases are archived
		ProjectName:            "helm",
		AssetMatchingStrategy:  "flexible",
	}

	fmt.Println("helm configuration example:")
	fmt.Printf("Project: %s, Direct Binary: %t\n", helmConfig.ProjectName, helmConfig.IsDirectBinary)
}

// Example function showing custom asset patterns
func exampleCustomPatterns() {
	// Example for a project with very specific naming
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    true,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
		IsDirectBinary:         false,
		ProjectName:            "myapp",
		AssetMatchingStrategy:  "custom",
		CustomAssetPatterns: []string{
			`myapp-v\d+\.\d+\.\d+-{OS}-{ARCH}\.tar\.gz`,
			`myapp-{OS}-{ARCH}-v\d+\.\d+\.\d+\.tar\.gz`,
		},
	}

	fmt.Println("Custom pattern configuration example:")
	fmt.Printf("Strategy: %s\n", config.AssetMatchingStrategy)
	fmt.Printf("Patterns: %v\n", config.CustomAssetPatterns)
}

// Example function showing how to handle different architectures
func exampleArchitectureHandling() {
	fmt.Println("Architecture mapping examples:")
	
	// The library automatically maps Go architectures to common release naming
	architectures := []string{"amd64", "arm64", "arm", "386"}
	
	for _, arch := range architectures {
		mapped := release.MapArch(arch)
		fmt.Printf("Go arch '%s' maps to '%s' for asset matching\n", arch, mapped)
	}
	
	// The flexible matcher also handles variants
	fmt.Println("\nArchitecture variants handled:")
	fmt.Println("- amd64: amd64, x86_64, x64")
	fmt.Println("- arm64: arm64, aarch64")
	fmt.Println("- arm: arm, armv6, armv7, armhf")
	fmt.Println("- 386: 386, i386, i686, x86")
}

// Example function showing error handling for asset matching
func exampleErrorHandling() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "nonexistent",
		BinaryName:             "nonexistent",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test",
		SourceArchivePath:      "/tmp/nonexistent",
		IsDirectBinary:         true,
		ProjectName:            "nonexistent",
		AssetMatchingStrategy:  "flexible",
	}

	// This would fail because the project doesn't exist
	githubRelease := release.NewGithubRelease("nonexistent/project", config)
	
	err := githubRelease.GetLatestRelease()
	if err != nil {
		fmt.Printf("Expected error for nonexistent project: %v\n", err)
	}
}

// Example function showing how to test asset matching without downloading
func exampleAssetMatchingTest() {
	// Create asset matcher configuration
	config := release.DefaultAssetMatchingConfig()
	config.ProjectName = "k0s"
	config.IsDirectBinary = true
	config.Strategy = release.FlexibleStrategy

	// Example k0s asset names
	assetNames := []string{
		"k0s-v1.33.2+k0s.0-amd64",
		"k0s-v1.33.2+k0s.0-arm64",
		"k0s-v1.33.2+k0s.0-amd64.exe",
		"k0s-v1.33.2+k0s.0-arm64.exe",
	}

	matcher := release.NewAssetMatcher(config)
	bestMatch, err := matcher.FindBestMatch(assetNames)
	
	if err != nil {
		fmt.Printf("Asset matching failed: %v\n", err)
		return
	}

	fmt.Printf("Best asset match for current platform: %s\n", bestMatch)
}

// Example function showing environment-based configuration
func exampleEnvironmentConfiguration() {
	// Set environment variables for authentication
	os.Setenv("GITHUB_TOKEN", "your-github-token-here")
	defer os.Unsetenv("GITHUB_TOKEN")

	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "k0s",
		BinaryName:             "k0s",
		CreateLocalSymlink:     true,  // Core value proposition: local symlinks
		CreateGlobalSymlink:    true,  // Optional: global symlinks
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/k0s-latest",
		IsDirectBinary:         true,
		ProjectName:            "k0s",
		AssetMatchingStrategy:  "flexible",
	}

	// The library will automatically use the GITHUB_TOKEN environment variable
	githubRelease := release.NewGithubRelease("k0sproject/k0s", config)

	fmt.Println("GitHub release configured with environment token")
	fmt.Printf("Repository: %s\n", githubRelease.Repository)
	fmt.Printf("Token configured: %t\n", githubRelease.Token != "")
}

// Example function showing different symlink configurations
func exampleSymlinkConfigurations() {
	fmt.Println("Symlink Configuration Examples:")

	// Default configuration (preserves symlink-first approach)
	defaultConfig := fileUtils.DefaultFileConfig()
	fmt.Printf("Default config - Local symlinks: %t, Global symlinks: %t\n",
		defaultConfig.CreateLocalSymlink, defaultConfig.CreateGlobalSymlink)

	// Configuration for environments where symlinks can't be created
	noSymlinkConfig := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateLocalSymlink:     false, // Disable local symlinks
		CreateGlobalSymlink:    false, // Disable global symlinks
		BaseBinaryDirectory:    "/opt/myapp",
		SourceArchivePath:      "/tmp/myapp.tar.gz",
		IsDirectBinary:         false,
		AssetMatchingStrategy:  "flexible",
	}

	fmt.Println("\nNo-symlink configuration (for restricted environments):")
	fmt.Printf("  Local symlinks: %t\n", noSymlinkConfig.CreateLocalSymlink)
	fmt.Printf("  Global symlinks: %t\n", noSymlinkConfig.CreateGlobalSymlink)
	fmt.Println("  Binary will be available at versioned path only")

	// Configuration for CI/CD environments
	ciConfig := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "tool",
		BinaryName:             "tool",
		CreateLocalSymlink:     true,  // Enable for easy access
		CreateGlobalSymlink:    false, // Disable (no sudo in CI)
		BaseBinaryDirectory:    "/tmp/ci-tools",
		SourceArchivePath:      "/tmp/tool.tar.gz",
		IsDirectBinary:         false,
		AssetMatchingStrategy:  "flexible",
	}

	fmt.Println("\nCI/CD configuration:")
	fmt.Printf("  Local symlinks: %t (easy access)\n", ciConfig.CreateLocalSymlink)
	fmt.Printf("  Global symlinks: %t (no sudo in CI)\n", ciConfig.CreateGlobalSymlink)

	// Configuration for system-wide installation
	systemConfig := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "systemtool",
		BinaryName:             "systemtool",
		CreateLocalSymlink:     true,  // Local access
		CreateGlobalSymlink:    true,  // System-wide access
		BaseBinaryDirectory:    "/opt/systemtool",
		SourceArchivePath:      "/tmp/systemtool.tar.gz",
		IsDirectBinary:         false,
		AssetMatchingStrategy:  "flexible",
	}

	fmt.Println("\nSystem-wide configuration:")
	fmt.Printf("  Local symlinks: %t\n", systemConfig.CreateLocalSymlink)
	fmt.Printf("  Global symlinks: %t (requires sudo)\n", systemConfig.CreateGlobalSymlink)
}

// Example function showing graceful symlink fallback
func exampleSymlinkFallback() {
	fmt.Println("Symlink Fallback Example:")

	// This demonstrates how the library handles symlink creation failures gracefully
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateLocalSymlink:     true,  // Attempt to create symlinks
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/fallback-test",
		SourceArchivePath:      "/tmp/myapp.tar.gz",
		IsDirectBinary:         false,
		AssetMatchingStrategy:  "flexible",
	}

	fmt.Println("Configuration attempts symlink creation but falls back gracefully:")
	fmt.Printf("  If symlink creation succeeds: Binary available via symlink\n")
	fmt.Printf("  If symlink creation fails: Binary still available at versioned path\n")
	fmt.Printf("  Installation continues successfully in both cases\n")

	// Example of checking installation status
	version := "v1.0.0"
	if info, err := fileUtils.GetInstallationInfo(config, version); err == nil {
		fmt.Printf("\nInstallation info would show:\n")
		fmt.Printf("  Symlink Status: %s\n", info.SymlinkStatus)
		fmt.Printf("  Binary Path: %s\n", info.BinaryPath)
		fmt.Printf("  Fallback ensures binary is always accessible\n")
	}
}
