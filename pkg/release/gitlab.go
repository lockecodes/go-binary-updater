package release

import (
	"encoding/json"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"
)

const gitlabApiUrl = "https://gitlab.com/api/v4/projects/%s/releases"

type GitLabRelease struct {
	ProjectId   string               `json:"project_id"`
	ReleaseLink string               `json:"latest_release_link"`
	Version     string               `json:"version"`
	Config      fileUtils.FileConfig `json:"config"`
	BaseURL     string               // Added to allow overriding API URL for tests
}

func (r *GitLabRelease) getTempSourceArchivePath() string {
	if r.Config.SourceArchivePath != "" {
		return r.Config.SourceArchivePath
	}
	return path.Join("/tmp", fmt.Sprintf("binary-%s.tar.gz", r.Version))
}

func (r *GitLabRelease) GetApiUrl() (string, error) {
	// Convert the string to an integer
	projectId, err := strconv.Atoi(r.ProjectId)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	if projectId <= 0 {
		return "", fmt.Errorf("invalid project ID: %s", r.ProjectId)
	}
	if r.BaseURL == "" {
		urlString := fmt.Sprintf(gitlabApiUrl, r.ProjectId)
		return urlString, nil
	}
	// Construct the request URL
	reqUrl := r.BaseURL + "/" + r.ProjectId + "/releases"
	return reqUrl, nil
}

func (r *GitLabRelease) GetLatestRelease() error {
	log.Println("Fetching latest release from GitLab")
	apiURL, err := r.GetApiUrl()
	if err != nil {
		return fmt.Errorf("error constructing GitLab API URL: %w", err)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("error making HTTP request to GitLab: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from GitLab: %d", resp.StatusCode)
	}

	var responses []GitlabReleaseResponse

	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return fmt.Errorf(
			"error decoding response from GitLab: %w",
			err)
	}

	if len(responses) == 0 {
		return fmt.Errorf(
			"no GitLab releases found for project ID %s",
			r.ProjectId)
	}

	// Sort releases and get the latest one
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].ReleasedAt.After(responses[j].ReleasedAt)
	})
	r.ReleaseLink = responses[0].GetReleaseLink()
	r.Version = responses[0].TagName
	return nil
}

func (r *GitLabRelease) DownloadLatestRelease() error {
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

func (r *GitLabRelease) InstallLatestRelease() error {
	return fileUtils.InstallBinary(r.Config, r.Version)
}

func NewGitlabRelease(projectId string, fileConfig fileUtils.FileConfig) *GitLabRelease {
	return &GitLabRelease{
		ProjectId: projectId,
		Config:    fileConfig,
	}
}
