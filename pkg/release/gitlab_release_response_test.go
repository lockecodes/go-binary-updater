package release

import (
	"testing"
	"time"
)

func TestGitlabReleaseResponse_GetReleaseLink(t *testing.T) {
	type fields struct {
		Name        string
		TagName     string
		Description string
		CreatedAt   time.Time
		ReleasedAt  time.Time
		Assets      struct {
			Links []struct {
				Id             int
				Name           string
				Url            string
				DirectAssetUrl string
			}
		}
	}
	tableTests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test with expected link",
			fields: fields{
				Assets: struct {
					Links []struct {
						Id             int
						Name           string
						Url            string
						DirectAssetUrl string
					}
				}{
					Links: []struct {
						Id             int
						Name           string
						Url            string
						DirectAssetUrl string
					}{
						{
							Name:           "Linux_x86_64",
							DirectAssetUrl: "http://direct_link_to_asset.com/linux_amd64_binary.tar.gz",
						},
					},
				},
			},
			want: "http://direct_link_to_asset.com/linux_amd64_binary.tar.gz",
		},
		{
			name: "Test with missing asset link",
			fields: fields{
				Assets: struct {
					Links []struct {
						Id             int
						Name           string
						Url            string
						DirectAssetUrl string
					}
				}{},
			},
			want: "",
		},
	}

	for _, tt := range tableTests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitlabReleaseResponse{
				Name:        tt.fields.Name,
				TagName:     tt.fields.TagName,
				Description: tt.fields.Description,
				CreatedAt:   tt.fields.CreatedAt,
				ReleasedAt:  tt.fields.ReleasedAt,
				Assets: struct {
					Links []struct {
						Id             int    `json:"id"`
						Name           string `json:"name"`
						Url            string `json:"url"`
						DirectAssetUrl string `json:"direct_asset_url"`
					} `json:"links"`
				}(tt.fields.Assets),
			}
			if got := g.GetReleaseLink(); got != tt.want {
				t.Errorf("GitlabReleaseResponse.GetReleaseLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
