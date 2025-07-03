package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
	// Example configuration for downloading and installing a binary from GitLab
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    true,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Create a GitLab release instance
	// Project ID format: numeric ID (e.g., "12345678")
	// You can find this in your GitLab project settings or main page
	gitlabRelease := release.NewGitlabRelease("12345678", config)

	// Optional: Use authentication for private repos or higher rate limits
	// token := os.Getenv("GITLAB_TOKEN")
	// if token != "" {
	//     gitlabRelease = release.NewGitlabReleaseWithToken("12345678", token, config)
	// }

	// Example 1: Get latest release information
	fmt.Println("Fetching latest release information...")
	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		log.Fatalf("Error getting latest release: %v", err)
	}
	fmt.Printf("Latest version: %s\n", gitlabRelease.Version)
	fmt.Printf("Download URL: %s\n", gitlabRelease.ReleaseLink)

	// Example 2: Download the latest release
	fmt.Println("\nDownloading latest release...")
	err = gitlabRelease.DownloadLatestRelease()
	if err != nil {
		log.Fatalf("Error downloading latest release: %v", err)
	}
	fmt.Println("Download completed successfully!")

	// Example 3: Install the latest release
	fmt.Println("\nInstalling latest release...")
	err = gitlabRelease.InstallLatestRelease()
	if err != nil {
		log.Fatalf("Error installing latest release: %v", err)
	}
	fmt.Println("Installation completed successfully!")

	// Example 4: Using the Release interface
	fmt.Println("\nUsing Release interface...")
	var releaseProvider release.Release = gitlabRelease

	err = releaseProvider.GetLatestRelease()
	if err != nil {
		log.Fatalf("Error with interface: %v", err)
	}
	fmt.Printf("Interface works! Version: %s\n", gitlabRelease.Version)
}

// Example function showing how to use GitLab releases with authentication
func exampleWithAuthentication() {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		log.Println("No GITLAB_TOKEN environment variable set")
		return
	}

	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Create GitLab release with authentication
	gitlabRelease := release.NewGitlabReleaseWithToken("12345678", token, config)

	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		log.Printf("Error with authenticated request: %v", err)
		return
	}

	fmt.Printf("Authenticated request successful! Version: %s\n", gitlabRelease.Version)
}

// Example function showing how to use GitLab releases with self-hosted instances
func exampleWithSelfHostedGitLab() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Create GitLab configuration for self-hosted instance
	gitlabConfig := release.DefaultGitLabConfig()
	gitlabConfig.BaseURL = "https://gitlab.example.com/api/v4"

	gitlabRelease := release.NewGitlabReleaseWithConfig("12345678", config, gitlabConfig)

	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		log.Printf("Error with self-hosted GitLab: %v", err)
		return
	}

	fmt.Printf("Self-hosted GitLab request successful! Version: %s\n", gitlabRelease.Version)
}

// Example function showing how to find your GitLab project ID
func exampleFindProjectID() {
	fmt.Println("How to find your GitLab Project ID:")
	fmt.Println("1. Go to your GitLab project page")
	fmt.Println("2. Look below the project name - you'll see 'Project ID: 12345678'")
	fmt.Println("3. Or go to Settings → General to find the project ID")
	fmt.Println("4. You can also use the GitLab API:")
	fmt.Println("   curl 'https://gitlab.com/api/v4/projects/owner%2Frepo'")
	fmt.Println("   (replace 'owner/repo' with your project path)")
}

// Example function demonstrating error handling patterns
func exampleErrorHandling() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Example with invalid project ID
	gitlabRelease := release.NewGitlabRelease("invalid", config)

	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		fmt.Printf("Expected error with invalid project ID: %v\n", err)
	}

	// Example with non-existent project
	gitlabRelease = release.NewGitlabRelease("999999999", config)

	err = gitlabRelease.GetLatestRelease()
	if err != nil {
		fmt.Printf("Expected error with non-existent project: %v\n", err)
	}
}

// Example function showing polymorphic usage with GitHub
func examplePolymorphicUsage() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Array of different release providers
	var providers []release.Release

	// Add GitLab provider
	providers = append(providers, release.NewGitlabRelease("12345678", config))

	// Add GitHub provider (if you have both)
	providers = append(providers, release.NewGithubRelease("owner/repo", config))

	// Use the same interface for both
	for i, provider := range providers {
		fmt.Printf("Testing provider %d...\n", i+1)

		err := provider.GetLatestRelease()
		if err != nil {
			fmt.Printf("Provider %d failed: %v\n", i+1, err)
			continue
		}

		fmt.Printf("Provider %d succeeded!\n", i+1)
	}
}

// Example function showing configuration options
func exampleAdvancedConfiguration() {
	// Advanced configuration with all options
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",          // Directory name for versioned binaries
		SourceBinaryName:       "myapp",             // Name of binary in the archive
		BinaryName:             "myapp-cli",         // Name to use for installed binary
		CreateGlobalSymlink:    true,                // Create symlink in PATH
		BaseBinaryDirectory:    "/opt/myapp",        // Base installation directory
		SourceArchivePath:      "/tmp/myapp.tar.gz", // Download location
	}

	gitlabRelease := release.NewGitlabRelease("12345678", config)

	// This will:
	// 1. Download to /tmp/myapp.tar.gz
	// 2. Extract 'myapp' binary from archive
	// 3. Install to /opt/myapp/versions/v1.2.3/myapp-cli
	// 4. Create symlink /opt/myapp/myapp-cli → /opt/myapp/versions/v1.2.3/myapp-cli

	err := gitlabRelease.DownloadLatestRelease()
	if err != nil {
		log.Printf("Download failed: %v", err)
		return
	}

	err = gitlabRelease.InstallLatestRelease()
	if err != nil {
		log.Printf("Installation failed: %v", err)
		return
	}

	fmt.Printf("Advanced installation completed for version %s\n", gitlabRelease.Version)
}

// Example showing advanced HTTP configuration with retry logic
func exampleAdvancedHTTPConfig() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Create advanced GitLab configuration
	gitlabConfig := release.DefaultGitLabConfig()
	gitlabConfig.BaseURL = "https://gitlab.example.com/api/v4" // Self-hosted GitLab
	gitlabConfig.Token = os.Getenv("GITLAB_TOKEN")

	// Configure retry logic
	gitlabConfig.HTTPConfig.MaxRetries = 5
	gitlabConfig.HTTPConfig.InitialDelay = 2 * time.Second
	gitlabConfig.HTTPConfig.MaxDelay = 60 * time.Second
	gitlabConfig.HTTPConfig.BackoffFactor = 2.0
	gitlabConfig.HTTPConfig.Timeout = 45 * time.Second
	gitlabConfig.HTTPConfig.CircuitBreaker = true

	// Add custom headers
	gitlabConfig.CustomHeaders = map[string]string{
		"X-Custom-Header": "custom-value",
		"User-Agent":      "my-app/1.0",
	}

	gitlabRelease := release.NewGitlabReleaseWithConfig("12345678", config, gitlabConfig)

	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		log.Printf("Advanced configuration failed: %v", err)
		return
	}

	fmt.Printf("Advanced configuration successful! Version: %s\n", gitlabRelease.Version)
}

// Example showing how to check if a release exists before downloading
func exampleCheckReleaseExists() {
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    false,
		BaseBinaryDirectory:    "/tmp/test-install",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	gitlabRelease := release.NewGitlabRelease("12345678", config)

	// Check if release exists without downloading
	err := gitlabRelease.GetLatestRelease()
	if err != nil {
		log.Printf("No release available: %v", err)
		return
	}

	fmt.Printf("Latest release available: %s\n", gitlabRelease.Version)
	fmt.Printf("Download URL: %s\n", gitlabRelease.ReleaseLink)

	// Now decide whether to download based on version, size, etc.
	if gitlabRelease.Version != "" {
		fmt.Println("Proceeding with download...")
		err = gitlabRelease.DownloadLatestRelease()
		if err != nil {
			log.Printf("Download failed: %v", err)
			return
		}
		fmt.Println("Download successful!")
	}
}
