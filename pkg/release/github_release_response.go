package release

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"runtime"
	"strings"
	"time"
)

type GithubReleaseResponse struct {
	ID          int       `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
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
	} `json:"assets"`
}

func (g *GithubReleaseResponse) GetReleaseLink() string {
	runtimeOS := runtime.GOOS
	arch := MapArch(runtime.GOARCH)

	title := cases.Title(language.AmericanEnglish)
	searchKey := fmt.Sprintf("%s_%s", title.String(runtimeOS), arch)

	releaseLink := ""
	for _, asset := range g.Assets {
		if strings.Contains(asset.Name, searchKey) {
			releaseLink = asset.BrowserDownloadUrl
			break
		}
	}
	return releaseLink
}
