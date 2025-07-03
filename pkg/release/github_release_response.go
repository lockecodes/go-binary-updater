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
	return g.GetReleaseLinkWithConfig(DefaultAssetMatchingConfig())
}

func (g *GithubReleaseResponse) GetReleaseLinkWithConfig(config AssetMatchingConfig) string {
	// Extract asset names
	assetNames := make([]string, len(g.Assets))
	assetMap := make(map[string]string)

	for i, asset := range g.Assets {
		assetNames[i] = asset.Name
		assetMap[asset.Name] = asset.BrowserDownloadUrl
	}

	// Use asset matcher to find the best match
	matcher := NewAssetMatcher(config)
	bestMatch, err := matcher.FindBestMatch(assetNames)
	if err != nil {
		// Fallback to legacy matching for backward compatibility
		return g.getLegacyReleaseLink()
	}

	return assetMap[bestMatch]
}

// getLegacyReleaseLink provides backward compatibility with the old matching logic
func (g *GithubReleaseResponse) getLegacyReleaseLink() string {
	runtimeOS := runtime.GOOS
	arch := MapArch(runtime.GOARCH)

	title := cases.Title(language.AmericanEnglish)
	searchKey := fmt.Sprintf("%s_%s", title.String(runtimeOS), arch)

	for _, asset := range g.Assets {
		if strings.Contains(asset.Name, searchKey) {
			return asset.BrowserDownloadUrl
		}
	}
	return ""
}
