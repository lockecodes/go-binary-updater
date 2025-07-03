package release

import (
	"encoding/json"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Default GitLab API configuration
const (
	DefaultGitLabAPIURL = "https://gitlab.com/api/v4"
	GitLabAPIVersion    = "v4"
)

// GitLabConfig holds configuration for GitLab API access
type GitLabConfig struct {
	BaseURL       string            // GitLab instance base URL (e.g., "https://gitlab.example.com/api/v4")
	Token         string            // Personal Access Token or Project Access Token
	HTTPConfig    HTTPClientConfig  // HTTP client configuration with retry logic
	CustomHeaders map[string]string // Additional headers for requests
}

// DefaultGitLabConfig returns a default GitLab configuration
func DefaultGitLabConfig() GitLabConfig {
	return GitLabConfig{
		BaseURL:       DefaultGitLabAPIURL,
		HTTPConfig:    DefaultHTTPClientConfig(),
		CustomHeaders: make(map[string]string),
	}
}

type GitLabRelease struct {
	ProjectId   string               `json:"project_id"`
	ReleaseLink string               `json:"latest_release_link"`
	Version     string               `json:"version"`
	Config      fileUtils.FileConfig `json:"config"`
	GitLabConfig GitLabConfig        `json:"gitlab_config"` // Enhanced configuration
	httpClient  *RetryableHTTPClient // HTTP client with retry logic
	AssetMatchingConfig AssetMatchingConfig `json:"asset_matching_config"` // Configuration for asset matching
}

func (r *GitLabRelease) getTempSourceArchivePath() string {
	if r.Config.SourceArchivePath != "" {
		return r.Config.SourceArchivePath
	}
	return path.Join("/tmp", fmt.Sprintf("binary-%s.tar.gz", r.Version))
}

// initializeHTTPClient initializes the HTTP client if not already done
func (r *GitLabRelease) initializeHTTPClient() {
	if r.httpClient == nil {
		// Ensure GitLabConfig is initialized
		if r.GitLabConfig.BaseURL == "" && r.GitLabConfig.HTTPConfig.MaxRetries == 0 {
			r.GitLabConfig = DefaultGitLabConfig()
		}
		r.httpClient = NewRetryableHTTPClient(r.GitLabConfig.HTTPConfig)
	}
}

// GetApiUrl constructs the GitLab API URL for releases
func (r *GitLabRelease) GetApiUrl() (string, error) {
	// Validate project ID
	projectId, err := strconv.Atoi(r.ProjectId)
	if err != nil {
		return "", fmt.Errorf("invalid project ID format '%s': %w", r.ProjectId, err)
	}

	if projectId <= 0 {
		return "", fmt.Errorf("invalid project ID: %s (must be positive integer)", r.ProjectId)
	}

	// Use configured base URL or default
	baseURL := r.GitLabConfig.BaseURL
	if baseURL == "" {
		// Initialize config if not set
		if r.GitLabConfig.HTTPConfig.MaxRetries == 0 {
			r.GitLabConfig = DefaultGitLabConfig()
		}
		baseURL = DefaultGitLabAPIURL
	}

	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Construct the releases endpoint URL
	return fmt.Sprintf("%s/projects/%s/releases", baseURL, r.ProjectId), nil
}

// getAuthHeaders returns authentication headers if token is configured
func (r *GitLabRelease) getAuthHeaders() map[string]string {
	headers := make(map[string]string)

	// Add authentication header if token is provided
	if r.GitLabConfig.Token != "" {
		headers["Authorization"] = "Bearer " + r.GitLabConfig.Token
	}

	// Add custom headers
	for key, value := range r.GitLabConfig.CustomHeaders {
		headers[key] = value
	}

	// Add standard headers
	headers["Accept"] = "application/json"
	headers["User-Agent"] = "go-binary-updater/1.0"

	return headers
}

func (r *GitLabRelease) GetLatestRelease() error {
	log.Println("Fetching latest release from GitLab")

	// Initialize HTTP client
	r.initializeHTTPClient()

	apiURL, err := r.GetApiUrl()
	if err != nil {
		return fmt.Errorf("error constructing GitLab API URL: %w", err)
	}

	// Get authentication headers
	headers := r.getAuthHeaders()

	// Make request with retry logic
	resp, err := r.httpClient.GetWithHeaders(apiURL, headers)
	if err != nil {
		return fmt.Errorf("error making HTTP request to GitLab: %w", err)
	}
	defer resp.Body.Close()

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - continue processing
	case http.StatusNotFound:
		return fmt.Errorf("GitLab project not found (ID: %s). Check project ID and permissions", r.ProjectId)
	case http.StatusForbidden:
		return fmt.Errorf("access denied to GitLab project (ID: %s). Check authentication token and permissions", r.ProjectId)
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed for GitLab project (ID: %s). Check token validity", r.ProjectId)
	default:
		return fmt.Errorf("unexpected status code from GitLab: %d", resp.StatusCode)
	}

	// Read response body
	body, err := ReadResponseBody(resp)
	if err != nil {
		return fmt.Errorf("error reading response body from GitLab: %w", err)
	}

	var responses []GitlabReleaseResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return fmt.Errorf("error decoding response from GitLab: %w", err)
	}

	if len(responses) == 0 {
		return fmt.Errorf("no GitLab releases found for project ID %s", r.ProjectId)
	}

	// Sort releases by release date (most recent first)
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].ReleasedAt.After(responses[j].ReleasedAt)
	})

	// Get the latest release
	latestRelease := responses[0]
	r.Version = latestRelease.TagName

	// Find platform-specific release link
	releaseLink := latestRelease.GetReleaseLinkWithConfig(r.AssetMatchingConfig)
	if releaseLink == "" {
		return fmt.Errorf("no suitable asset found for current platform (%s/%s) in GitLab release %s",
			runtime.GOOS, runtime.GOARCH, latestRelease.TagName)
	}

	r.ReleaseLink = releaseLink
	return nil
}

func (r *GitLabRelease) DownloadLatestRelease() error {
	// Handle CDN downloads
	if r.AssetMatchingConfig.Strategy == CDNStrategy || r.AssetMatchingConfig.Strategy == HybridStrategy {
		return r.downloadFromCDN()
	}

	err := r.GetLatestRelease()
	if err != nil {
		return fmt.Errorf("error getting latest release from GitLab: %w", err)
	}
	if r.Version == "" || r.ReleaseLink == "" {
		return fmt.Errorf("could not find a valid release to download")
	}
	err = fileUtils.DownloadFile(r.ReleaseLink, r.Config.SourceArchivePath)
	if err != nil {
		return fmt.Errorf(
			"error downloading latest release from GitLab: %w",
			err)
	}
	return nil
}

// downloadFromCDN downloads binary from CDN instead of GitLab releases
func (r *GitLabRelease) downloadFromCDN() error {
	if r.Version == "" {
		// For CDN downloads, we still need version info, so get it from GitLab
		err := r.GetLatestRelease()
		if err != nil {
			return fmt.Errorf("error getting version information from GitLab: %w", err)
		}
	}

	// Create CDN downloader with custom architecture mapping if configured
	var cdnDownloader *CDNDownloader
	if r.AssetMatchingConfig.CDNArchMapping != nil {
		cdnDownloader = NewCDNDownloaderWithArchMapping(
			r.AssetMatchingConfig.CDNBaseURL,
			r.AssetMatchingConfig.CDNPattern,
			r.AssetMatchingConfig.CDNArchMapping,
		)
	} else {
		cdnDownloader = NewCDNDownloader(r.AssetMatchingConfig.CDNBaseURL, r.AssetMatchingConfig.CDNPattern)
	}

	versionFormat := r.AssetMatchingConfig.CDNVersionFormat
	if versionFormat == "" {
		versionFormat = "as-is" // Default to as-is if not specified
	}
	return cdnDownloader.DownloadWithVersionFormat(r.Version, r.Config.SourceArchivePath, versionFormat)
}

func (r *GitLabRelease) InstallLatestRelease() error {
	// Use enhanced installation with extraction config if available
	if r.AssetMatchingConfig.ExtractionConfig != nil && !r.Config.IsDirectBinary {
		// Convert ExtractionConfig to fileUtils.ExtractionConfig
		fileUtilsConfig := &fileUtils.ExtractionConfig{
			StripComponents: r.AssetMatchingConfig.ExtractionConfig.StripComponents,
			BinaryPath:      r.AssetMatchingConfig.ExtractionConfig.BinaryPath,
		}
		return fileUtils.InstallArchivedBinaryWithConfig(r.Config, r.Version, fileUtilsConfig)
	}
	return fileUtils.InstallBinary(r.Config, r.Version)
}



// NewGitlabRelease creates a new GitLab release instance with default configuration
func NewGitlabRelease(projectId string, fileConfig fileUtils.FileConfig) *GitLabRelease {
	config := DefaultGitLabConfig()

	// Check for environment variables
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		config.Token = token
	}
	if baseURL := os.Getenv("GITLAB_API_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	// Configure asset matching
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
	case "cdn":
		assetConfig.Strategy = CDNStrategy
	case "hybrid":
		assetConfig.Strategy = HybridStrategy
	default:
		assetConfig.Strategy = FlexibleStrategy
	}

	return &GitLabRelease{
		ProjectId:           projectId,
		Config:              fileConfig,
		GitLabConfig:        config,
		AssetMatchingConfig: assetConfig,
	}
}

// NewGitlabReleaseWithToken creates a new GitLab release instance with authentication token
func NewGitlabReleaseWithToken(projectId string, token string, fileConfig fileUtils.FileConfig) *GitLabRelease {
	release := NewGitlabRelease(projectId, fileConfig)
	release.GitLabConfig.Token = token
	return release
}

// NewGitlabReleaseWithConfig creates a new GitLab release instance with full configuration
func NewGitlabReleaseWithConfig(projectId string, fileConfig fileUtils.FileConfig, gitlabConfig GitLabConfig) *GitLabRelease {
	// Configure asset matching
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

	return &GitLabRelease{
		ProjectId:           projectId,
		Config:              fileConfig,
		GitLabConfig:        gitlabConfig,
		AssetMatchingConfig: assetConfig,
	}
}

// GetInstalledBinaryPath returns the preferred path to the installed binary
// Prefers symlink path when available, falls back to versioned directory path
func (r *GitLabRelease) GetInstalledBinaryPath() (string, error) {
	if r.Version == "" {
		return "", fmt.Errorf("no version information available - call GetLatestRelease() first")
	}
	return fileUtils.GetInstalledBinaryPath(r.Config, r.Version)
}

// GetInstallationInfo returns comprehensive information about the installed binary
func (r *GitLabRelease) GetInstallationInfo() (*fileUtils.InstallationInfo, error) {
	if r.Version == "" {
		return nil, fmt.Errorf("no version information available - call GetLatestRelease() first")
	}
	return fileUtils.GetInstallationInfo(r.Config, r.Version)
}

// SetCustomHeaders allows setting custom headers for GitLab API requests
func (r *GitLabRelease) SetCustomHeaders(headers map[string]string) {
	if r.GitLabConfig.CustomHeaders == nil {
		r.GitLabConfig.CustomHeaders = make(map[string]string)
	}
	for key, value := range headers {
		r.GitLabConfig.CustomHeaders[key] = value
	}
}

// NewGitlabReleaseWithAssetConfig creates a new GitLab release instance with custom asset matching configuration
// This preserves any CDN strategy settings in the provided configuration
func NewGitlabReleaseWithAssetConfig(projectId string, fileConfig fileUtils.FileConfig, assetConfig AssetMatchingConfig) *GitLabRelease {
	config := DefaultGitLabConfig()

	// Check for environment variables
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		config.Token = token
	}
	if baseURL := os.Getenv("GITLAB_API_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	// Merge fileConfig properties into assetConfig while preserving CDN strategy
	if assetConfig.ProjectName == "" {
		assetConfig.ProjectName = fileConfig.ProjectName
	}
	// Only override IsDirectBinary if it's not explicitly set in assetConfig
	if fileConfig.IsDirectBinary {
		assetConfig.IsDirectBinary = fileConfig.IsDirectBinary
	}

	// Auto-detect CDN strategy if CDN configuration is present but strategy is not CDN/Hybrid
	if assetConfig.CDNBaseURL != "" && assetConfig.CDNPattern != "" {
		if assetConfig.Strategy != CDNStrategy && assetConfig.Strategy != HybridStrategy {
			assetConfig.Strategy = CDNStrategy
		}
	}

	return &GitLabRelease{
		ProjectId:           projectId,
		Config:              fileConfig,
		GitLabConfig:        config,
		AssetMatchingConfig: assetConfig,
	}
}

// NewGitlabReleaseWithCDNConfig creates a new GitLab release instance configured for CDN downloads
// This is a convenience function for common CDN configurations like Helm, kubectl, etc.
func NewGitlabReleaseWithCDNConfig(projectId string, fileConfig fileUtils.FileConfig, cdnConfig AssetMatchingConfig) *GitLabRelease {
	// Ensure CDN strategy is set
	cdnConfig.Strategy = CDNStrategy
	return NewGitlabReleaseWithAssetConfig(projectId, fileConfig, cdnConfig)
}

// SetHTTPConfig allows customizing the HTTP client configuration
func (r *GitLabRelease) SetHTTPConfig(config HTTPClientConfig) {
	r.GitLabConfig.HTTPConfig = config
	// Reset HTTP client to pick up new configuration
	r.httpClient = nil
}
