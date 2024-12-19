package release

import (
	"encoding/json"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"log"
	"net/http"
	"path"
	"sort"
)

const gitlabApiUrl = "https://gitlab.com/api/v4/projects/%d/releases"

type FileConfig struct {
	VersionedDirectoryName string `json:"versioned_directory_name"`
	SourceBinaryName       string `json:"source_binary_name"`
	BinaryName             string `json:"binary_name"`
	CreateGlobalSymlink    bool   `json:"create_global_symlink"`
}

type GitLabRelease struct {
	ProjectId   int        `json:"project_id"`
	ReleaseLink string     `json:"latest_release_link"`
	Version     string     `json:"version"`
	Config      FileConfig `json:"config"`
}

func (r *GitLabRelease) getTempSourceArchivePath() string {
	return path.Join("/tmp", fmt.Sprintf("binary-%s.tar.gz", r.Version))
}

func (r *GitLabRelease) GetApiUrl() (string, error) {
	if r.ProjectId <= 0 {
		return "", fmt.Errorf("invalid project ID: %d", r.ProjectId)
	}
	urlString := fmt.Sprintf(gitlabApiUrl, r.ProjectId)

	return urlString, nil
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
			"no GitLab releases found for project ID %d",
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
	err = fileUtils.DownloadFile(r.ReleaseLink, r.getTempSourceArchivePath())
	if err != nil {
		return fmt.Errorf(
			"error downloading latest release from GitLab: %w",
			err)
	}
	return nil
}

func (r *GitLabRelease) InstallLatestRelease() error {
	return fileUtils.InstallBinary(r.getTempSourceArchivePath(), r.Config.VersionedDirectoryName, r.Config.SourceBinaryName, r.Config.BinaryName, r.Version, r.Config.CreateGlobalSymlink)
}

func NewGitlabRelease(projectId int, fileConfig FileConfig) *GitLabRelease {
	return &GitLabRelease{
		ProjectId: projectId,
		Config:    fileConfig,
	}
}
