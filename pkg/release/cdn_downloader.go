package release

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// CDNDownloader handles downloading binaries from external CDNs
type CDNDownloader struct {
	BaseURL    string
	Pattern    string
	HTTPClient *http.Client
}

// NewCDNDownloader creates a new CDN downloader with the given configuration
func NewCDNDownloader(baseURL, pattern string) *CDNDownloader {
	return &CDNDownloader{
		BaseURL: baseURL,
		Pattern: pattern,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large binaries
		},
	}
}

// ConstructURL builds the download URL for the given version and platform
func (c *CDNDownloader) ConstructURL(version, os, arch string) string {
	url := c.BaseURL + c.Pattern
	
	// Replace placeholders
	url = strings.ReplaceAll(url, "{version}", version)
	url = strings.ReplaceAll(url, "{os}", os)
	url = strings.ReplaceAll(url, "{arch}", arch)
	
	// Handle version variations (remove 'v' prefix if present in pattern but not in version)
	if strings.Contains(c.Pattern, "{version}") && strings.HasPrefix(version, "v") {
		versionWithoutV := strings.TrimPrefix(version, "v")
		url = strings.ReplaceAll(url, version, versionWithoutV)
	}
	
	return url
}

// Download downloads a binary from the CDN to the specified path
func (c *CDNDownloader) Download(version, destinationPath string) error {
	// Use current platform for CDN downloads
	osName := runtime.GOOS
	archName := MapArch(runtime.GOARCH)
	
	// Map OS names for CDN compatibility
	switch osName {
	case "darwin":
		osName = "darwin" // Some CDNs use "darwin", others use "macos"
	case "windows":
		// Some CDNs expect "windows", others expect "win"
		// This will be handled by the specific CDN configuration
	}
	
	url := c.ConstructURL(version, osName, archName)
	
	fmt.Printf("Downloading from CDN: %s\n", url)
	
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	
	// Set user agent
	req.Header.Set("User-Agent", "go-binary-updater/1.0")
	
	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from CDN: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDN download failed with status %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Create destination file
	destFile, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()
	
	// Copy response body to file
	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded content: %v", err)
	}
	
	fmt.Printf("Successfully downloaded to: %s\n", destinationPath)
	return nil
}

// GetHelmCDNConfig returns configuration for Helm's CDN
func GetHelmCDNConfig() AssetMatchingConfig {
	config := DefaultAssetMatchingConfig()
	config.Strategy = CDNStrategy
	config.CDNBaseURL = "https://get.helm.sh/"
	config.CDNPattern = "helm-{version}-{os}-{arch}.tar.gz"
	config.IsDirectBinary = false
	config.ProjectName = "helm"
	config.ExtractionConfig = &ExtractionConfig{
		BinaryPath: "{os}-{arch}/helm", // Helm extracts to os-arch subdirectory
	}
	return config
}

// GetKubectlCDNConfig returns configuration for kubectl's Google CDN
func GetKubectlCDNConfig() AssetMatchingConfig {
	config := DefaultAssetMatchingConfig()
	config.Strategy = CDNStrategy
	config.CDNBaseURL = "https://dl.k8s.io/release/"
	config.CDNPattern = "{version}/bin/{os}/{arch}/kubectl"
	config.IsDirectBinary = true
	config.ProjectName = "kubectl"
	// Add .exe extension for Windows
	if runtime.GOOS == "windows" {
		config.CDNPattern += ".exe"
	}
	return config
}

// GetK0sConfig returns enhanced configuration for k0s with strict exclusion patterns
func GetK0sConfig() AssetMatchingConfig {
	config := DefaultAssetMatchingConfig()
	config.Strategy = FlexibleStrategy
	config.ProjectName = "k0s"
	config.IsDirectBinary = true
	
	// Strict exclusion patterns for k0s to avoid airgap bundles
	config.ExcludePatterns = []string{
		"airgap",           // Exclude airgap bundles
		"bundle",           // Exclude any bundles
		"\\.asc$",          // Exclude signature files
		"\\.sha256$",       // Exclude checksum files
	}
	
	// Priority patterns to prefer direct binaries
	config.PriorityPatterns = []string{
		"^k0s-v.*-amd64$",     // Prefer direct k0s binaries for amd64
		"^k0s-v.*-arm64$",     // Prefer direct k0s binaries for arm64
		"^k0s-v.*-amd64\\.exe$", // Prefer direct k0s binaries for Windows
	}
	
	return config
}

// GetTerraformConfig returns configuration for Terraform with HashiCorp's CDN
func GetTerraformConfig() AssetMatchingConfig {
	config := DefaultAssetMatchingConfig()
	config.Strategy = HybridStrategy // Try GitHub first, then CDN
	config.CDNBaseURL = "https://releases.hashicorp.com/terraform/"
	config.CDNPattern = "{version}/terraform_{version}_{os}_{arch}.zip"
	config.IsDirectBinary = false
	config.ProjectName = "terraform"
	config.FileExtensions = []string{".zip"}
	return config
}

// GetDockerConfig returns configuration for Docker with enhanced patterns
func GetDockerConfig() AssetMatchingConfig {
	config := DefaultAssetMatchingConfig()
	config.Strategy = FlexibleStrategy
	config.ProjectName = "docker"
	config.IsDirectBinary = false
	config.FileExtensions = []string{".tgz", ".tar.gz"}
	
	// Exclude Docker Desktop and other non-CLI packages
	config.ExcludePatterns = []string{
		"desktop",
		"rootless",
		"static",
		"\\.asc$",
		"\\.sha256$",
	}
	
	// Priority patterns for Docker CLI
	config.PriorityPatterns = []string{
		"docker-.*-{os}-{arch}\\.tgz$",
		"docker-.*-{os}-{arch}\\.tar\\.gz$",
	}
	
	return config
}

// ValidateCDNConfig validates that a CDN configuration is properly set up
func ValidateCDNConfig(config AssetMatchingConfig) error {
	if config.Strategy == CDNStrategy || config.Strategy == HybridStrategy {
		if config.CDNBaseURL == "" {
			return fmt.Errorf("CDN strategy requires CDNBaseURL to be set")
		}
		if config.CDNPattern == "" {
			return fmt.Errorf("CDN strategy requires CDNPattern to be set")
		}
		
		// Validate that pattern contains required placeholders
		if !strings.Contains(config.CDNPattern, "{version}") {
			return fmt.Errorf("CDN pattern must contain {version} placeholder")
		}
		
		// For non-direct binaries, we need OS and arch placeholders
		if !config.IsDirectBinary {
			if !strings.Contains(config.CDNPattern, "{os}") {
				return fmt.Errorf("CDN pattern for archived binaries must contain {os} placeholder")
			}
			if !strings.Contains(config.CDNPattern, "{arch}") {
				return fmt.Errorf("CDN pattern for archived binaries must contain {arch} placeholder")
			}
		}
	}
	
	return nil
}

// GetPresetConfig returns a preset configuration for common binaries
func GetPresetConfig(binaryName string) (AssetMatchingConfig, error) {
	switch strings.ToLower(binaryName) {
	case "helm":
		return GetHelmCDNConfig(), nil
	case "kubectl":
		return GetKubectlCDNConfig(), nil
	case "k0s":
		return GetK0sConfig(), nil
	case "terraform":
		return GetTerraformConfig(), nil
	case "docker":
		return GetDockerConfig(), nil
	default:
		return AssetMatchingConfig{}, fmt.Errorf("no preset configuration available for binary: %s", binaryName)
	}
}
