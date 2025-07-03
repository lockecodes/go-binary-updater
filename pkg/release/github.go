package release

import (
	"encoding/json"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"log"
	"net/http"
	"path"
	"runtime"
	"strings"
)

const githubApiUrl = "https://api.github.com/repos/%s/releases/latest"

type GithubRelease struct {
	Repository  string               `json:"repository"`   // Format: "owner/repo"
	ReleaseLink string               `json:"release_link"` // Download URL for the selected asset
	Version     string               `json:"version"`      // Tag name of the release
	Config      fileUtils.FileConfig `json:"config"`       // File configuration
	BaseURL     string               // Added to allow overriding API URL for tests
	Token       string               // Optional GitHub token for authentication
	AssetMatchingConfig AssetMatchingConfig `json:"asset_matching_config"` // Configuration for asset matching
}

func (g *GithubRelease) getTempSourceArchivePath() string {
	if g.Config.SourceArchivePath != "" {
		return g.Config.SourceArchivePath
	}
	return path.Join("/tmp", fmt.Sprintf("binary-%s.tar.gz", g.Version))
}

func (g *GithubRelease) GetApiUrl() (string, error) {
	if g.Repository == "" {
		return "", fmt.Errorf("repository cannot be empty")
	}

	// Validate repository format (should be "owner/repo")
	parts := strings.Split(g.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format: %s (expected 'owner/repo')", g.Repository)
	}

	if g.BaseURL == "" {
		return fmt.Sprintf(githubApiUrl, g.Repository), nil
	}
	// Construct the request URL for testing
	return g.BaseURL + "/" + g.Repository + "/releases/latest", nil
}

func (g *GithubRelease) GetLatestRelease() error {
	log.Println("Fetching latest release from GitHub")
	apiURL, err := g.GetApiUrl()
	if err != nil {
		return fmt.Errorf("error constructing GitHub API URL: %w", err)
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Add authentication header if token is provided
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making HTTP request to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from GitHub: %d", resp.StatusCode)
	}

	var response GithubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response from GitHub: %w", err)
	}

	// Extract release information
	g.Version = response.TagName
	releaseLink := response.GetReleaseLinkWithConfig(g.AssetMatchingConfig)
	if releaseLink == "" {
		return fmt.Errorf("no suitable asset found for current platform (%s/%s) in GitHub release %s",
			runtime.GOOS, runtime.GOARCH, response.TagName)
	}
	g.ReleaseLink = releaseLink

	return nil
}

func (g *GithubRelease) DownloadLatestRelease() error {
	// Handle CDN downloads
	if g.AssetMatchingConfig.Strategy == CDNStrategy || g.AssetMatchingConfig.Strategy == HybridStrategy {
		return g.downloadFromCDN()
	}

	err := g.GetLatestRelease()
	if err != nil {
		return fmt.Errorf("error getting latest release from GitHub: %w", err)
	}
	if g.Version == "" || g.ReleaseLink == "" {
		return fmt.Errorf("could not find a valid release to download")
	}
	err = fileUtils.DownloadFile(g.ReleaseLink, g.Config.SourceArchivePath)
	if err != nil {
		return fmt.Errorf("error downloading latest release from GitHub: %w", err)
	}
	return nil
}

// downloadFromCDN downloads binary from CDN instead of GitHub releases
func (g *GithubRelease) downloadFromCDN() error {
	if g.Version == "" {
		// For CDN downloads, we still need version info, so get it from GitHub
		err := g.GetLatestRelease()
		if err != nil {
			return fmt.Errorf("error getting version information from GitHub: %w", err)
		}
	}

	cdnDownloader := NewCDNDownloader(g.AssetMatchingConfig.CDNBaseURL, g.AssetMatchingConfig.CDNPattern)
	versionFormat := g.AssetMatchingConfig.CDNVersionFormat
	if versionFormat == "" {
		versionFormat = "as-is" // Default to as-is if not specified
	}
	return cdnDownloader.DownloadWithVersionFormat(g.Version, g.Config.SourceArchivePath, versionFormat)
}

func (g *GithubRelease) InstallLatestRelease() error {
	// Use enhanced installation with extraction config if available
	if g.AssetMatchingConfig.ExtractionConfig != nil && !g.Config.IsDirectBinary {
		// Convert ExtractionConfig to fileUtils.ExtractionConfig
		fileUtilsConfig := &fileUtils.ExtractionConfig{
			StripComponents: g.AssetMatchingConfig.ExtractionConfig.StripComponents,
			BinaryPath:      g.AssetMatchingConfig.ExtractionConfig.BinaryPath,
		}
		return fileUtils.InstallArchivedBinaryWithConfig(g.Config, g.Version, fileUtilsConfig)
	}
	return fileUtils.InstallBinary(g.Config, g.Version)
}

func NewGithubRelease(repository string, fileConfig fileUtils.FileConfig) *GithubRelease {
	assetConfig := DefaultAssetMatchingConfig()
	assetConfig.ProjectName = fileConfig.ProjectName
	assetConfig.IsDirectBinary = fileConfig.IsDirectBinary

	// Configure asset matching strategy based on FileConfig
	switch fileConfig.AssetMatchingStrategy {
	case "standard":
		assetConfig.Strategy = StandardStrategy
	case "flexible":
		assetConfig.Strategy = FlexibleStrategy
	case "custom":
		assetConfig.Strategy = CustomStrategy
		assetConfig.CustomPatterns = fileConfig.CustomAssetPatterns
	default:
		assetConfig.Strategy = FlexibleStrategy
	}

	return &GithubRelease{
		Repository:          repository,
		Config:              fileConfig,
		AssetMatchingConfig: assetConfig,
	}
}

func NewGithubReleaseWithToken(repository string, token string, fileConfig fileUtils.FileConfig) *GithubRelease {
	release := NewGithubRelease(repository, fileConfig)
	release.Token = token
	return release
}

// GetInstalledBinaryPath returns the preferred path to the installed binary
// Prefers symlink path when available, falls back to versioned directory path
func (g *GithubRelease) GetInstalledBinaryPath() (string, error) {
	if g.Version == "" {
		return "", fmt.Errorf("no version information available - call GetLatestRelease() first")
	}
	return fileUtils.GetInstalledBinaryPath(g.Config, g.Version)
}

// GetInstallationInfo returns comprehensive information about the installed binary
func (g *GithubRelease) GetInstallationInfo() (*fileUtils.InstallationInfo, error) {
	if g.Version == "" {
		return nil, fmt.Errorf("no version information available - call GetLatestRelease() first")
	}
	return fileUtils.GetInstallationInfo(g.Config, g.Version)
}
