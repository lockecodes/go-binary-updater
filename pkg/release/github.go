package release

type GithubRelease struct {
	Repository string
	// Add any necessary fields for GitHub-specific logic
}

func (g *GithubRelease) InstallLatestRelease() error {
	panic("InstallLatestRelease not implemented")
}

func (g *GithubRelease) GetLatestRelease() Release {
	panic("GetLatestRelease not implemented")
}

func (g *GithubRelease) DownloadLatestRelease() error {
	panic("DownloadLatestRelease not implemented")
}
