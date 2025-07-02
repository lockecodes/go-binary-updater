package release

import (
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"net/http"
	"net/http/httptest"
	"testing"
)

var GithubApiResponse string
var GithubApiStatusCode int

type githubTestCase struct {
	description    string
	expectedLink   string
	expectedErr    string
	responseObject string
	release        GithubRelease
}

func TestGithubRelease_GetLatestRelease(t *testing.T) {
	mockServer := mockGithubServer()
	defer mockServer.Close()

	testCases := []githubTestCase{
		successfulReleaseTest(mockServer.URL),
		releaseWithoutAssetTest(mockServer.URL),
		releaseWithoutMatchingAssetTest(mockServer.URL),
		invalidRepositoryFormatTest(mockServer.URL),
		emptyRepositoryTest(mockServer.URL),
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			GithubApiResponse = testCase.responseObject
			GithubApiStatusCode = http.StatusOK

			if testCase.expectedErr != "" {
				GithubApiStatusCode = http.StatusNotFound
			}

			err := testCase.release.GetLatestRelease()

			if testCase.expectedErr != "" {
				if err == nil {
					t.Errorf("Expected error: %s, but got nil", testCase.expectedErr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if testCase.release.ReleaseLink != testCase.expectedLink {
					t.Errorf("Expected link: %s, got: %s", testCase.expectedLink, testCase.release.ReleaseLink)
				}
			}
		})
	}
}

func TestGithubRelease_GetApiUrl(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		baseURL    string
		want       string
		wantErr    bool
	}{
		{
			name:       "Valid repository format",
			repository: "owner/repo",
			baseURL:    "",
			want:       "https://api.github.com/repos/owner/repo/releases/latest",
			wantErr:    false,
		},
		{
			name:       "Valid repository with custom base URL",
			repository: "owner/repo",
			baseURL:    "https://api.example.com",
			want:       "https://api.example.com/owner/repo/releases/latest",
			wantErr:    false,
		},
		{
			name:       "Invalid repository format - missing slash",
			repository: "ownerrepo",
			baseURL:    "",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "Invalid repository format - empty owner",
			repository: "/repo",
			baseURL:    "",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "Invalid repository format - empty repo",
			repository: "owner/",
			baseURL:    "",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "Empty repository",
			repository: "",
			baseURL:    "",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GithubRelease{
				Repository: tt.repository,
				BaseURL:    tt.baseURL,
			}
			got, err := g.GetApiUrl()
			if (err != nil) != tt.wantErr {
				t.Errorf("GithubRelease.GetApiUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GithubRelease.GetApiUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func successfulReleaseTest(mockURL string) githubTestCase {
	return githubTestCase{
		description:  "Successful release fetch",
		expectedLink: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Linux_x86_64.tar.gz",
		expectedErr:  "",
		responseObject: `{
			"id": 1,
			"tag_name": "v1.0.0",
			"name": "Release v1.0.0",
			"body": "Release description",
			"draft": false,
			"prerelease": false,
			"created_at": "2023-01-01T00:00:00Z",
			"published_at": "2023-01-01T00:00:00Z",
			"assets": [
				{
					"id": 1,
					"name": "myapp-Linux_x86_64.tar.gz",
					"label": "",
					"content_type": "application/gzip",
					"size": 1024,
					"download_count": 42,
					"browser_download_url": "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Linux_x86_64.tar.gz",
					"created_at": "2023-01-01T00:00:00Z",
					"updated_at": "2023-01-01T00:00:00Z"
				}
			]
		}`,
		release: GithubRelease{
			Repository: "owner/repo",
			Config:     fileUtils.FileConfig{},
			BaseURL:    mockURL,
		},
	}
}

func releaseWithoutAssetTest(mockURL string) githubTestCase {
	return githubTestCase{
		description:  "Release without assets",
		expectedLink: "",
		expectedErr:  "no suitable asset found for current platform",
		responseObject: `{
			"id": 1,
			"tag_name": "v1.0.0",
			"name": "Release v1.0.0",
			"body": "Release description",
			"draft": false,
			"prerelease": false,
			"created_at": "2023-01-01T00:00:00Z",
			"published_at": "2023-01-01T00:00:00Z",
			"assets": []
		}`,
		release: GithubRelease{
			Repository: "owner/repo",
			Config:     fileUtils.FileConfig{},
			BaseURL:    mockURL,
		},
	}
}

func releaseWithoutMatchingAssetTest(mockURL string) githubTestCase {
	return githubTestCase{
		description:  "Release without matching asset for current platform",
		expectedLink: "",
		expectedErr:  "no suitable asset found for current platform",
		responseObject: `{
			"id": 1,
			"tag_name": "v1.0.0",
			"name": "Release v1.0.0",
			"body": "Release description",
			"draft": false,
			"prerelease": false,
			"created_at": "2023-01-01T00:00:00Z",
			"published_at": "2023-01-01T00:00:00Z",
			"assets": [
				{
					"id": 1,
					"name": "myapp-Windows_x86_64.zip",
					"browser_download_url": "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Windows_x86_64.zip"
				}
			]
		}`,
		release: GithubRelease{
			Repository: "owner/repo",
			Config:     fileUtils.FileConfig{},
			BaseURL:    mockURL,
		},
	}
}

func invalidRepositoryFormatTest(mockURL string) githubTestCase {
	return githubTestCase{
		description:    "Invalid repository format",
		expectedLink:   "",
		expectedErr:    "invalid repository format",
		responseObject: "",
		release: GithubRelease{
			Repository: "invalid-repo-format",
			Config:     fileUtils.FileConfig{},
			BaseURL:    mockURL,
		},
	}
}

func emptyRepositoryTest(mockURL string) githubTestCase {
	return githubTestCase{
		description:    "Empty repository",
		expectedLink:   "",
		expectedErr:    "repository cannot be empty",
		responseObject: "",
		release: GithubRelease{
			Repository: "",
			Config:     fileUtils.FileConfig{},
			BaseURL:    mockURL,
		},
	}
}

func mockGithubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(GithubApiStatusCode)
		rw.Write([]byte(GithubApiResponse))
	}))
}

func TestGithubRelease_ImplementsReleaseInterface(t *testing.T) {
	// This test ensures that GithubRelease implements the Release interface
	var _ Release = &GithubRelease{}
}

func TestNewGithubRelease(t *testing.T) {
	config := fileUtils.FileConfig{
		BinaryName: "test-binary",
	}

	release := NewGithubRelease("owner/repo", config)

	if release.Repository != "owner/repo" {
		t.Errorf("Expected repository 'owner/repo', got '%s'", release.Repository)
	}
	if release.Config.BinaryName != "test-binary" {
		t.Errorf("Expected binary name 'test-binary', got '%s'", release.Config.BinaryName)
	}
	if release.Token != "" {
		t.Errorf("Expected empty token, got '%s'", release.Token)
	}
}

func TestNewGithubReleaseWithToken(t *testing.T) {
	config := fileUtils.FileConfig{
		BinaryName: "test-binary",
	}

	release := NewGithubReleaseWithToken("owner/repo", "test-token", config)

	if release.Repository != "owner/repo" {
		t.Errorf("Expected repository 'owner/repo', got '%s'", release.Repository)
	}
	if release.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", release.Token)
	}
	if release.Config.BinaryName != "test-binary" {
		t.Errorf("Expected binary name 'test-binary', got '%s'", release.Config.BinaryName)
	}
}
