package release

type Release interface {
	GetLatestRelease() error      // Returns the latest release information
	DownloadLatestRelease() error // Downloads the latest release binary
	InstallLatestRelease() error  // Updates and installs the binary
}
