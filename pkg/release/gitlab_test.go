package release

import (
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var GitLabApiResponse string
var GitLabApiStatusCode int

type testCase struct {
	description    string
	expectedLink   string
	expectedErr    string
	responseObject string
	release        GitLabRelease
}

func TestGitLabReleaseMethods(t *testing.T) {
	server := mockGitLabServer() // Start mock server
	defer server.Close()         // Ensure it is closed after the test

	var tests = []testCase{
		longTermSupportReleaseTest(server.URL),
		releaseWithoutLinkTest(server.URL),
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			testHelperSetGitLabResponse(tt.responseObject)
			err := tt.release.GetLatestRelease()
			if err != nil && err.Error() != tt.expectedErr {
				t.Errorf("got error %q, want error %q", err, tt.expectedErr)
			}
			if tt.release.ReleaseLink != tt.expectedLink {
				t.Errorf("got link %s, want link %s", tt.release.ReleaseLink, tt.expectedLink)
			}
		})
	}
}

func testHelperSetGitLabResponse(responseObject string) {
	GitLabApiResponse = responseObject
	GitLabApiStatusCode = 200
}

func longTermSupportReleaseTest(mockURL string) testCase {
	return testCase{
		description:  "Long term support release",
		expectedLink: "https://gitlab.com/locke-codes/container-cli/-/releases/v1.2.3/downloads/container-cli_Linux_x86_64.tar.gz",
		expectedErr:  "",
		responseObject: `[{
    "name": "v1.2.3",
    "tag_name": "v1.2.3",
    "created_at": "2024-12-19T03:37:43.664Z",
    "released_at": "2024-12-19T03:37:43.664Z",
    "assets": {
      "links": [
        {
          "id": 6461593,
          "name": "checksums.txt",
          "url": "https://gitlab.com//-/project/47137983/uploads/6b82b6bf9ffe3a4288b0b84eb111a73a/checksums.txt",
          "direct_asset_url": "https://gitlab.com/locke-codes/container-cli/-/releases/v1.2.3/downloads/checksums.txt",
          "link_type": "other"
        },
        {
          "id": 6461587,
          "name": "container-cli_Linux_x86_64.tar.gz",
          "url": "https://gitlab.com//-/project/47137983/uploads/be54011e62d628d80dc3a2e1414b0d75/container-cli_Linux_x86_64.tar.gz",
          "direct_asset_url": "https://gitlab.com/locke-codes/container-cli/-/releases/v1.2.3/downloads/container-cli_Linux_x86_64.tar.gz",
          "link_type": "other"
        }
      ]
    }
  }]`,
		release: func() GitLabRelease {
			r := GitLabRelease{
				ProjectId: "1",
				Version:   "v1.2.3",
				Config:    fileUtils.FileConfig{},
				GitLabConfig: DefaultGitLabConfig(),
			}
			r.GitLabConfig.BaseURL = mockURL
			return r
		}(),
	}
}

func releaseWithoutLinkTest(mockURL string) testCase {
	return testCase{
		description:    "Release without link",
		expectedLink:   "",
		expectedErr:    "no GitLab releases found for project ID 1",
		responseObject: "[]",
		release: func() GitLabRelease {
			r := GitLabRelease{
				ProjectId: "1",
				Version:   "v1.2.3",
				Config:    fileUtils.FileConfig{},
				GitLabConfig: DefaultGitLabConfig(),
			}
			r.GitLabConfig.BaseURL = mockURL
			return r
		}(),
	}
}

func mockGitLabServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(GitLabApiStatusCode)
		rw.Write([]byte(GitLabApiResponse))
	}))
}

func TestGitLabRelease_Authentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for authentication header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"tag_name": "v1.0.0",
			"released_at": "2023-01-01T00:00:00Z",
			"assets": {
				"links": [{
					"name": "myapp-Linux_x86_64.tar.gz",
					"direct_asset_url": "https://example.com/download"
				}]
			}
		}]`))
	}))
	defer server.Close()

	config := fileUtils.FileConfig{}
	release := NewGitlabReleaseWithToken("12345", "test-token", config)
	release.GitLabConfig.BaseURL = server.URL

	err := release.GetLatestRelease()
	if err != nil {
		t.Errorf("Expected success with authentication, got error: %v", err)
	}

	if release.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", release.Version)
	}
}

func TestGitLabRelease_CustomBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"tag_name": "v2.0.0",
			"released_at": "2023-01-01T00:00:00Z",
			"assets": {
				"links": [{
					"name": "myapp-Linux_x86_64.tar.gz",
					"direct_asset_url": "https://example.com/download"
				}]
			}
		}]`))
	}))
	defer server.Close()

	config := fileUtils.FileConfig{}
	gitlabConfig := DefaultGitLabConfig()
	gitlabConfig.BaseURL = server.URL

	release := NewGitlabReleaseWithConfig("12345", config, gitlabConfig)

	err := release.GetLatestRelease()
	if err != nil {
		t.Errorf("Expected success with custom base URL, got error: %v", err)
	}

	if release.Version != "v2.0.0" {
		t.Errorf("Expected version v2.0.0, got %s", release.Version)
	}
}

func TestGitLabRelease_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("GITLAB_TOKEN", "env-token")
	os.Setenv("GITLAB_API_URL", "https://gitlab.example.com/api/v4")
	defer func() {
		os.Unsetenv("GITLAB_TOKEN")
		os.Unsetenv("GITLAB_API_URL")
	}()

	config := fileUtils.FileConfig{}
	release := NewGitlabRelease("12345", config)

	if release.GitLabConfig.Token != "env-token" {
		t.Errorf("Expected token from environment, got %s", release.GitLabConfig.Token)
	}

	if release.GitLabConfig.BaseURL != "https://gitlab.example.com/api/v4" {
		t.Errorf("Expected base URL from environment, got %s", release.GitLabConfig.BaseURL)
	}
}

func TestGitLabRelease_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"tag_name": "v1.0.0",
			"released_at": "2023-01-01T00:00:00Z",
			"assets": {
				"links": [{
					"name": "myapp-Linux_x86_64.tar.gz",
					"direct_asset_url": "https://example.com/download"
				}]
			}
		}]`))
	}))
	defer server.Close()

	config := fileUtils.FileConfig{}
	gitlabConfig := DefaultGitLabConfig()
	gitlabConfig.BaseURL = server.URL
	gitlabConfig.HTTPConfig.MaxRetries = 3
	gitlabConfig.HTTPConfig.InitialDelay = 10 * time.Millisecond

	release := NewGitlabReleaseWithConfig("12345", config, gitlabConfig)

	err := release.GetLatestRelease()
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestGitLabRelease_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  string
	}{
		{
			name:          "Not Found",
			statusCode:    http.StatusNotFound,
			expectedError: "GitLab project not found",
		},
		{
			name:          "Forbidden",
			statusCode:    http.StatusForbidden,
			expectedError: "access denied to GitLab project",
		},
		{
			name:          "Unauthorized",
			statusCode:    http.StatusUnauthorized,
			expectedError: "authentication failed for GitLab project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			config := fileUtils.FileConfig{}
			release := NewGitlabRelease("12345", config)
			release.GitLabConfig.BaseURL = server.URL

			err := release.GetLatestRelease()
			if err == nil {
				t.Error("Expected error, got success")
			}

			if err != nil && !contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
