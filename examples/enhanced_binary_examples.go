package main

import (
	"fmt"
	"log"
	"os"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
	fmt.Println("Enhanced Binary Download Examples")
	fmt.Println("=================================")
	
	// Demonstrate different binary configurations
	exampleK0sWithStrictFiltering()
	exampleHelmWithCDN()
	exampleKubectlWithGoogleCDN()
	exampleTerraformWithHybridStrategy()
	exampleDockerWithEnhancedFiltering()
}

// exampleK0sWithStrictFiltering demonstrates k0s with strict filtering to avoid airgap bundles
func exampleK0sWithStrictFiltering() {
	fmt.Println("\n1. k0s with Strict Filtering (Excludes Airgap Bundles)")
	fmt.Println("-----------------------------------------------------")
	
	// Get preset k0s configuration with enhanced filtering
	assetConfig := release.GetK0sConfig()
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "k0s",
		BinaryName:             "k0s",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/k0s-latest",
		
		// Enhanced k0s configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "flexible",
	}

	githubRelease := release.NewGithubRelease("k0sproject/k0s", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Strategy: %v\n", assetConfig.Strategy)
	fmt.Printf("  Exclude Patterns: %v\n", assetConfig.ExcludePatterns)
	fmt.Printf("  Priority Patterns: %v\n", assetConfig.PriorityPatterns)
	fmt.Printf("  Direct Binary: %t\n", assetConfig.IsDirectBinary)
	
	// In a real scenario, you would call:
	// err := githubRelease.DownloadLatestRelease()
	// err = githubRelease.InstallLatestRelease()
	
	fmt.Println("  ✓ Configured to exclude airgap bundles and prefer direct binaries")
}

// exampleHelmWithCDN demonstrates Helm using CDN instead of GitHub releases
func exampleHelmWithCDN() {
	fmt.Println("\n2. Helm with CDN Download (get.helm.sh)")
	fmt.Println("---------------------------------------")
	
	// Get preset Helm CDN configuration
	assetConfig := release.GetHelmCDNConfig()
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "helm",
		BinaryName:             "helm",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/helm-latest.tar.gz",
		
		// Helm CDN configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "cdn",
	}

	githubRelease := release.NewGithubRelease("helm/helm", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Strategy: CDN\n")
	fmt.Printf("  CDN Base URL: %s\n", assetConfig.CDNBaseURL)
	fmt.Printf("  CDN Pattern: %s\n", assetConfig.CDNPattern)
	fmt.Printf("  Extraction Config: %+v\n", assetConfig.ExtractionConfig)
	
	// In a real scenario, you would call:
	// err := githubRelease.DownloadLatestRelease() // Downloads from get.helm.sh
	// err = githubRelease.InstallLatestRelease()   // Extracts from os-arch subdirectory
	
	fmt.Println("  ✓ Configured to download from Helm's CDN instead of GitHub releases")
}

// exampleKubectlWithGoogleCDN demonstrates kubectl using Google's CDN
func exampleKubectlWithGoogleCDN() {
	fmt.Println("\n3. kubectl with Google CDN (dl.k8s.io)")
	fmt.Println("--------------------------------------")
	
	// Get preset kubectl CDN configuration
	assetConfig := release.GetKubectlCDNConfig()
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "kubectl",
		BinaryName:             "kubectl",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/kubectl-latest",
		
		// kubectl CDN configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "cdn",
	}

	githubRelease := release.NewGithubRelease("kubernetes/kubernetes", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Strategy: CDN\n")
	fmt.Printf("  CDN Base URL: %s\n", assetConfig.CDNBaseURL)
	fmt.Printf("  CDN Pattern: %s\n", assetConfig.CDNPattern)
	fmt.Printf("  Direct Binary: %t\n", assetConfig.IsDirectBinary)
	
	// In a real scenario, you would call:
	// err := githubRelease.DownloadLatestRelease() // Downloads from Google's CDN
	// err = githubRelease.InstallLatestRelease()   // Direct binary installation
	
	fmt.Println("  ✓ Configured to download from Google's Kubernetes CDN")
}

// exampleTerraformWithHybridStrategy demonstrates Terraform with hybrid strategy
func exampleTerraformWithHybridStrategy() {
	fmt.Println("\n4. Terraform with Hybrid Strategy (GitHub + HashiCorp CDN)")
	fmt.Println("----------------------------------------------------------")
	
	// Get preset Terraform hybrid configuration
	assetConfig := release.GetTerraformConfig()
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "terraform",
		BinaryName:             "terraform",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/terraform-latest.zip",
		
		// Terraform hybrid configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "hybrid",
	}

	githubRelease := release.NewGithubRelease("hashicorp/terraform", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Strategy: Hybrid (GitHub first, then CDN)\n")
	fmt.Printf("  CDN Base URL: %s\n", assetConfig.CDNBaseURL)
	fmt.Printf("  CDN Pattern: %s\n", assetConfig.CDNPattern)
	fmt.Printf("  File Extensions: %v\n", assetConfig.FileExtensions)
	
	// In a real scenario, you would call:
	// err := githubRelease.DownloadLatestRelease() // Tries GitHub first, falls back to CDN
	// err = githubRelease.InstallLatestRelease()   // Extracts ZIP archive
	
	fmt.Println("  ✓ Configured to try GitHub releases first, then fall back to HashiCorp CDN")
}

// exampleDockerWithEnhancedFiltering demonstrates Docker with enhanced filtering
func exampleDockerWithEnhancedFiltering() {
	fmt.Println("\n5. Docker with Enhanced Filtering (Excludes Desktop/Rootless)")
	fmt.Println("------------------------------------------------------------")
	
	// Get preset Docker configuration with enhanced filtering
	assetConfig := release.GetDockerConfig()
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "docker",
		BinaryName:             "docker",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/docker-latest.tgz",
		
		// Docker enhanced configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "flexible",
	}

	githubRelease := release.NewGithubRelease("docker/cli", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Strategy: Flexible with enhanced filtering\n")
	fmt.Printf("  Exclude Patterns: %v\n", assetConfig.ExcludePatterns)
	fmt.Printf("  Priority Patterns: %v\n", assetConfig.PriorityPatterns)
	fmt.Printf("  File Extensions: %v\n", assetConfig.FileExtensions)
	
	// In a real scenario, you would call:
	// err := githubRelease.DownloadLatestRelease()
	// err = githubRelease.InstallLatestRelease()
	
	fmt.Println("  ✓ Configured to exclude Docker Desktop and rootless packages")
}

// exampleCustomConfiguration demonstrates creating a custom configuration
func exampleCustomConfiguration() {
	fmt.Println("\n6. Custom Configuration Example")
	fmt.Println("-------------------------------")
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
		
		// Custom configuration
		IsDirectBinary:         false,
		ProjectName:            "myapp",
		AssetMatchingStrategy:  "flexible",
	}

	// Create custom asset matching configuration
	assetConfig := release.DefaultAssetMatchingConfig()
	assetConfig.ProjectName = "myapp"
	assetConfig.ExcludePatterns = []string{
		"debug",     // Exclude debug builds
		"test",      // Exclude test builds
		"\\.asc$",   // Exclude signature files
	}
	assetConfig.PriorityPatterns = []string{
		"myapp-.*-{os}-{arch}\\.tar\\.gz$", // Prefer standard naming
	}

	githubRelease := release.NewGithubRelease("myorg/myapp", config)
	githubRelease.AssetMatchingConfig = assetConfig

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Custom exclude patterns: %v\n", assetConfig.ExcludePatterns)
	fmt.Printf("  Custom priority patterns: %v\n", assetConfig.PriorityPatterns)
	
	fmt.Println("  ✓ Custom configuration with specific filtering rules")
}

// exampleErrorHandling demonstrates comprehensive error handling
func exampleErrorHandling() {
	fmt.Println("\n7. Error Handling and Debugging")
	fmt.Println("-------------------------------")
	
	// Example of handling CDN configuration errors
	invalidConfig := release.AssetMatchingConfig{
		Strategy:   release.CDNStrategy,
		CDNBaseURL: "", // Missing required field
	}
	
	if err := release.ValidateCDNConfig(invalidConfig); err != nil {
		fmt.Printf("CDN validation error (expected): %v\n", err)
	}
	
	// Example of handling unknown binary preset
	if _, err := release.GetPresetConfig("unknown-binary"); err != nil {
		fmt.Printf("Preset config error (expected): %v\n", err)
	}
	
	// Example of asset matching with no results
	config := release.DefaultAssetMatchingConfig()
	config.ExcludePatterns = []string{".*"} // Exclude everything
	
	matcher := release.NewAssetMatcher(config)
	assetNames := []string{"some-binary-linux-amd64.tar.gz"}
	
	if _, err := matcher.FindBestMatch(assetNames); err != nil {
		fmt.Printf("Asset matching error (expected): %v\n", err)
	}
	
	fmt.Println("  ✓ Comprehensive error handling provides clear feedback")
}

// exampleEnvironmentBasedConfiguration shows environment-based setup
func exampleEnvironmentBasedConfiguration() {
	fmt.Println("\n8. Environment-Based Configuration")
	fmt.Println("----------------------------------")
	
	// Set environment variables for authentication
	os.Setenv("GITHUB_TOKEN", "your-github-token-here")
	defer os.Unsetenv("GITHUB_TOKEN")
	
	// Detect binary type and configure automatically
	binaryName := "helm" // Could come from command line args
	
	assetConfig, err := release.GetPresetConfig(binaryName)
	if err != nil {
		log.Printf("No preset config for %s, using defaults", binaryName)
		assetConfig = release.DefaultAssetMatchingConfig()
		assetConfig.ProjectName = binaryName
	}
	
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       binaryName,
		BinaryName:             binaryName,
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      fmt.Sprintf("/tmp/%s-latest", binaryName),
		
		// Use preset configuration
		IsDirectBinary:         assetConfig.IsDirectBinary,
		ProjectName:            assetConfig.ProjectName,
		AssetMatchingStrategy:  "flexible",
	}
	
	githubRelease := release.NewGithubRelease(fmt.Sprintf("helm/%s", binaryName), config)
	githubRelease.AssetMatchingConfig = assetConfig
	
	fmt.Printf("Auto-configured for %s:\n", binaryName)
	fmt.Printf("  Strategy: %v\n", assetConfig.Strategy)
	fmt.Printf("  Direct Binary: %t\n", assetConfig.IsDirectBinary)
	if assetConfig.CDNBaseURL != "" {
		fmt.Printf("  CDN URL: %s\n", assetConfig.CDNBaseURL)
	}
	
	fmt.Println("  ✓ Automatic configuration based on binary name and environment")
}
