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
	return g.GetReleaseLinkWithConfig(DefaultAssetMatchingConfig())
}

func (g *GitlabReleaseResponse) GetReleaseLinkWithConfig(config AssetMatchingConfig) string {
	// Extract asset names
	assetNames := make([]string, len(g.Assets.Links))
	assetMap := make(map[string]string)

	for i, link := range g.Assets.Links {
		assetNames[i] = link.Name
		assetMap[link.Name] = link.DirectAssetUrl
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
func (g *GitlabReleaseResponse) getLegacyReleaseLink() string {
	runtimeOS := runtime.GOOS
	arch := MapArch(runtime.GOARCH)

	title := cases.Title(language.AmericanEnglish)
	primarySearchKey := fmt.Sprintf("%s_%s", title.String(runtimeOS), arch)

	// Try exact match first
	for _, link := range g.Assets.Links {
		if strings.Contains(link.Name, primarySearchKey) {
			return link.DirectAssetUrl
		}
	}

	// Try with architecture variants for better compatibility
	archVariants := GetArchVariants(runtime.GOARCH)
	for _, archVariant := range archVariants {
		searchKey := fmt.Sprintf("%s_%s", title.String(runtimeOS), archVariant)
		for _, link := range g.Assets.Links {
			if strings.Contains(link.Name, searchKey) {
				return link.DirectAssetUrl
			}
		}
	}

	// Try case-insensitive matching as fallback
	lowerOS := strings.ToLower(runtimeOS)
	lowerArch := strings.ToLower(arch)
	fallbackSearchKey := fmt.Sprintf("%s_%s", lowerOS, lowerArch)

	for _, link := range g.Assets.Links {
		linkNameLower := strings.ToLower(link.Name)
		if strings.Contains(linkNameLower, fallbackSearchKey) {
			return link.DirectAssetUrl
		}
	}

	// No suitable asset found
	return ""
}
