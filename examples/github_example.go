package main

import (
	"fmt"
	"log"
	"os"

	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"gitlab.com/locke-codes/go-binary-updater/pkg/release"
)

func main() {
	// Example configuration for downloading and installing a binary from GitHub
	config := fileUtils.FileConfig{
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "myapp",
		BinaryName:             "myapp",
		CreateGlobalSymlink:    true,
		BaseBinaryDirectory:    "/home/user/.local/bin",
		SourceArchivePath:      "/tmp/myapp-latest.tar.gz",
	}

	// Create a GitHub release instance
	// Repository format: "owner/repo"
	githubRelease := release.NewGithubRelease("owner/repo", config)

	// Optional: Add GitHub token for authentication (recommended for private repos or higher rate limits)
	// githubToken := os.Getenv("GITHUB_TOKEN")
	// if githubToken != "" {
	//     githubRelease = release.NewGithubReleaseWithToken("owner/repo", githubToken, config)
	// }

	// Example 1: Get latest release information
	fmt.Println("Fetching latest release information...")
	err := githubRelease.GetLatestRelease()
	if err != nil {
		log.Fatalf("Error getting latest release: %v", err)
	}
	fmt.Printf("Latest version: %s\n", githubRelease.Version)
	fmt.Printf("Download URL: %s\n", githubRelease.ReleaseLink)

	// Example 2: Download the latest release
	fmt.Println("\nDownloading latest release...")
	err = githubRelease.DownloadLatestRelease()
	if err != nil {
		log.Fatalf("Error downloading latest release: %v", err)
	}
	fmt.Println("Download completed successfully!")

	// Example 3: Install the latest release
	fmt.Println("\nInstalling latest release...")
	err = githubRelease.InstallLatestRelease()
	if err != nil {
		log.Fatalf("Error installing latest release: %v", err)
	}
	fmt.Println("Installation completed successfully!")

	// Example 4: Using the Release interface
	fmt.Println("\nUsing Release interface...")
	var releaseProvider release.Release = githubRelease

	err = releaseProvider.GetLatestRelease()
	if err != nil {
		log.Fatalf("Error with interface: %v", err)
	}
	fmt.Printf("Interface works! Version: %s\n", githubRelease.Version)
}

// Example function showing how to use GitHub releases with authentication
func exampleWithAuthentication() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Println("No GITHUB_TOKEN environment variable set")
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

	// Create GitHub release with authentication
	githubRelease := release.NewGithubReleaseWithToken("owner/private-repo", token, config)

	err := githubRelease.GetLatestRelease()
	if err != nil {
		log.Printf("Error with authenticated request: %v", err)
		return
	}

	fmt.Printf("Authenticated request successful! Version: %s\n", githubRelease.Version)
}
