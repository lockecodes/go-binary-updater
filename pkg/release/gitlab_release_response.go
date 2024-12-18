package release

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"runtime"
	"strings"
	"time"
)

type GitlabReleaseResponse struct {
	Name        string    `json:"name"`
	TagName     string    `json:"tag_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ReleasedAt  time.Time `json:"released_at"`
	Assets      struct {
		Links []struct {
			Id             int    `json:"id"`
			Name           string `json:"name"`
			Url            string `json:"url"`
			DirectAssetUrl string `json:"direct_asset_url"`
		} `json:"links"`
	} `json:"assets"`
}

func (g *GitlabReleaseResponse) GetReleaseLink() string {
	runtimeOS := runtime.GOOS
	arch := MapArch(runtime.GOARCH)

	title := cases.Title(language.AmericanEnglish)
	searchKey := fmt.Sprintf("%s_%s", title.String(runtimeOS), arch)

	releaseLink := ""
	for _, link := range g.Assets.Links {
		if strings.Contains(link.Name, searchKey) {
			releaseLink = link.DirectAssetUrl
			break
		}
	}
	return releaseLink
}
