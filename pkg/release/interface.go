package release

import "gitlab.com/locke-codes/go-binary-updater/pkg/fileUtils"

type Release interface {
	GetLatestRelease() error      // Returns the latest release information
	DownloadLatestRelease() error // Downloads the latest release binary
	InstallLatestRelease() error  // Updates and installs the binary

	// Enhanced path resolution and installation info methods
	GetInstalledBinaryPath() (string, error)                    // Returns the preferred path to the installed binary
	GetInstallationInfo() (*fileUtils.InstallationInfo, error) // Returns comprehensive installation information
}
