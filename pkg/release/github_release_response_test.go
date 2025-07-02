package release

import (
	"testing"
	"time"
)

func TestGithubReleaseResponse_GetReleaseLink(t *testing.T) {
	type fields struct {
		ID          int
		TagName     string
		Name        string
		Body        string
		Draft       bool
		Prerelease  bool
		CreatedAt   time.Time
		PublishedAt time.Time
		Assets      []struct {
			ID                 int    `json:"id"`
			Name               string `json:"name"`
			Label              string `json:"label"`
			ContentType        string `json:"content_type"`
			Size               int    `json:"size"`
			DownloadCount      int    `json:"download_count"`
			BrowserDownloadUrl string `json:"browser_download_url"`
			CreatedAt          time.Time `json:"created_at"`
			UpdatedAt          time.Time `json:"updated_at"`
		}
	}
	tableTests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test with expected Linux x86_64 asset",
			fields: fields{
				Assets: []struct {
					ID                 int    `json:"id"`
					Name               string `json:"name"`
					Label              string `json:"label"`
					ContentType        string `json:"content_type"`
					Size               int    `json:"size"`
					DownloadCount      int    `json:"download_count"`
					BrowserDownloadUrl string `json:"browser_download_url"`
					CreatedAt          time.Time `json:"created_at"`
					UpdatedAt          time.Time `json:"updated_at"`
				}{
					{
						Name:               "myapp-Linux_x86_64.tar.gz",
						BrowserDownloadUrl: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Linux_x86_64.tar.gz",
					},
					{
						Name:               "myapp-Darwin_x86_64.tar.gz",
						BrowserDownloadUrl: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Darwin_x86_64.tar.gz",
					},
				},
			},
			want: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Linux_x86_64.tar.gz",
		},
		{
			name: "Test with missing asset for current platform",
			fields: fields{
				Assets: []struct {
					ID                 int    `json:"id"`
					Name               string `json:"name"`
					Label              string `json:"label"`
					ContentType        string `json:"content_type"`
					Size               int    `json:"size"`
					DownloadCount      int    `json:"download_count"`
					BrowserDownloadUrl string `json:"browser_download_url"`
					CreatedAt          time.Time `json:"created_at"`
					UpdatedAt          time.Time `json:"updated_at"`
				}{
					{
						Name:               "myapp-Windows_x86_64.zip",
						BrowserDownloadUrl: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Windows_x86_64.zip",
					},
				},
			},
			want: "",
		},
		{
			name: "Test with no assets",
			fields: fields{
				Assets: []struct {
					ID                 int    `json:"id"`
					Name               string `json:"name"`
					Label              string `json:"label"`
					ContentType        string `json:"content_type"`
					Size               int    `json:"size"`
					DownloadCount      int    `json:"download_count"`
					BrowserDownloadUrl string `json:"browser_download_url"`
					CreatedAt          time.Time `json:"created_at"`
					UpdatedAt          time.Time `json:"updated_at"`
				}{},
			},
			want: "",
		},
		{
			name: "Test with multiple matching assets (should return first match)",
			fields: fields{
				Assets: []struct {
					ID                 int    `json:"id"`
					Name               string `json:"name"`
					Label              string `json:"label"`
					ContentType        string `json:"content_type"`
					Size               int    `json:"size"`
					DownloadCount      int    `json:"download_count"`
					BrowserDownloadUrl string `json:"browser_download_url"`
					CreatedAt          time.Time `json:"created_at"`
					UpdatedAt          time.Time `json:"updated_at"`
				}{
					{
						Name:               "myapp-v1.0.0-Linux_x86_64.tar.gz",
						BrowserDownloadUrl: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-v1.0.0-Linux_x86_64.tar.gz",
					},
					{
						Name:               "myapp-Linux_x86_64-debug.tar.gz",
						BrowserDownloadUrl: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-Linux_x86_64-debug.tar.gz",
					},
				},
			},
			want: "https://github.com/owner/repo/releases/download/v1.0.0/myapp-v1.0.0-Linux_x86_64.tar.gz",
		},
	}

	for _, tt := range tableTests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GithubReleaseResponse{
				ID:          tt.fields.ID,
				TagName:     tt.fields.TagName,
				Name:        tt.fields.Name,
				Body:        tt.fields.Body,
				Draft:       tt.fields.Draft,
				Prerelease:  tt.fields.Prerelease,
				CreatedAt:   tt.fields.CreatedAt,
				PublishedAt: tt.fields.PublishedAt,
				Assets:      tt.fields.Assets,
			}
			if got := g.GetReleaseLink(); got != tt.want {
				t.Errorf("GithubReleaseResponse.GetReleaseLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
