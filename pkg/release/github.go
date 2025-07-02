package release

import (
	"encoding/json"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"log"
	"net/http"
	"path"
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
	releaseLink := response.GetReleaseLink()
	if releaseLink == "" {
		return fmt.Errorf("no suitable asset found for current platform in GitHub release %s", response.TagName)
	}
	g.ReleaseLink = releaseLink

	return nil
}

func (g *GithubRelease) DownloadLatestRelease() error {
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

func (g *GithubRelease) InstallLatestRelease() error {
	return fileUtils.InstallBinary(g.Config, g.Version)
}

func NewGithubRelease(repository string, fileConfig fileUtils.FileConfig) *GithubRelease {
	return &GithubRelease{
		Repository: repository,
		Config:     fileConfig,
	}
}

func NewGithubReleaseWithToken(repository string, token string, fileConfig fileUtils.FileConfig) *GithubRelease {
	return &GithubRelease{
		Repository: repository,
		Token:      token,
		Config:     fileConfig,
	}
}
