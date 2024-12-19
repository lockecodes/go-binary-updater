package release

import (
	"gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"
	"net/http"
	"net/http/httptest"
	"testing"
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
		release: GitLabRelease{
			ProjectId: "1",
			Version:   "v1.2.3",
			Config:    fileUtils.FileConfig{},
			BaseURL:   mockURL, // Use the mock server's URL
		},
	}
}

func releaseWithoutLinkTest(mockURL string) testCase {
	return testCase{
		description:    "Release without link",
		expectedLink:   "",
		expectedErr:    "no GitLab releases found for project ID 1",
		responseObject: "[]",
		release: GitLabRelease{
			ProjectId: "1",
			Version:   "v1.2.3",
			Config:    fileUtils.FileConfig{},
			BaseURL:   mockURL, // Use the mock server's URL
		},
	}
}

func mockGitLabServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(GitLabApiStatusCode)
		rw.Write([]byte(GitLabApiResponse))
	}))
}
