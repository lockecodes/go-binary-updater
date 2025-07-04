package fileUtils

import (
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/archiver"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type FileConfig struct {
	VersionedDirectoryName string `json:"versioned_directory_name"`
	SourceBinaryName       string `json:"source_binary_name"`
	BinaryName             string `json:"binary_name"`
	CreateGlobalSymlink    bool   `json:"create_global_symlink"`    // Create global symlink in /usr/local/bin (requires sudo)
	BaseBinaryDirectory    string `json:"base_binary_directory"`
	SourceArchivePath      string `json:"source_archive_path"`

	// Enhanced symlink control (preserving symlink-first approach)
	CreateLocalSymlink     bool   `json:"create_local_symlink"`     // Create local symlink in BaseBinaryDirectory (default: true)

	// Enhanced directory structure control
	UseVersionsSubdirectory bool   `json:"use_versions_subdirectory"` // Use versions/{ProjectName}/ subdirectory pattern (default: false for backward compatibility)

	// Enhanced configuration for flexible asset handling
	IsDirectBinary         bool   `json:"is_direct_binary"`         // True if the downloaded asset is a direct binary, not an archive
	ProjectName            string `json:"project_name"`             // Project name for asset matching (e.g., "k0s", "kubectl")
	AssetMatchingStrategy  string `json:"asset_matching_strategy"`  // Strategy for asset matching: "standard", "flexible", "custom"
	CustomAssetPatterns    []string `json:"custom_asset_patterns"`  // Custom regex patterns for asset matching
}

// InstallationInfo provides comprehensive information about an installed binary
type InstallationInfo struct {
	BinaryPath          string `json:"binary_path"`           // Preferred path to the binary (symlink if available, otherwise versioned path)
	Version             string `json:"version"`               // Version of the installed binary
	InstallationType    string `json:"installation_type"`     // "direct_binary" or "extracted_archive"
	SymlinkStatus       string `json:"symlink_status"`        // "created", "failed", "disabled", "not_attempted"
	LocalSymlinkPath    string `json:"local_symlink_path"`    // Path to local symlink (if created)
	GlobalSymlinkPath   string `json:"global_symlink_path"`   // Path to global symlink (if configured)
	VersionedPath       string `json:"versioned_path"`        // Path to binary in versioned directory
	LocalSymlinkCreated bool   `json:"local_symlink_created"` // Whether local symlink was successfully created
	GlobalSymlinkNeeded bool   `json:"global_symlink_needed"` // Whether global symlink creation was requested
}

// ExtractionConfig configures how binaries are extracted from archives
type ExtractionConfig struct {
	StripComponents int    `json:"strip_components"` // Number of directory components to strip (like tar --strip-components)
	BinaryPath      string `json:"binary_path"`      // Specific path to binary within archive (e.g., "linux-amd64/helm")
}

// DefaultFileConfig returns a FileConfig with sensible defaults that preserve symlink-first behavior
func DefaultFileConfig() FileConfig {
	return FileConfig{
		CreateLocalSymlink:      true,  // Default: create local symlinks (core value proposition)
		CreateGlobalSymlink:     false, // Default: don't create global symlinks (requires sudo)
		UseVersionsSubdirectory: false, // Default: use legacy directory structure for backward compatibility
		AssetMatchingStrategy:   "flexible", // Default: use flexible matching
		IsDirectBinary:          false, // Default: assume archived binaries
	}
}

// GetVersionedDirectoryPath returns the path to the versioned directory based on configuration
func GetVersionedDirectoryPath(config FileConfig, version string) string {
	if config.UseVersionsSubdirectory {
		// New pattern: BaseBinaryDirectory/versions/{ProjectName}/{version}/
		projectName := config.ProjectName
		if projectName == "" {
			// Fallback to BinaryName if ProjectName is not set
			projectName = config.BinaryName
		}
		return filepath.Join(config.BaseBinaryDirectory, "versions", projectName, version)
	} else {
		// Legacy pattern: BaseBinaryDirectory/{VersionedDirectoryName}/{version}/
		return filepath.Join(config.BaseBinaryDirectory, config.VersionedDirectoryName, version)
	}
}

// GetVersionedBinaryPath returns the full path to the binary in the versioned directory
func GetVersionedBinaryPath(config FileConfig, version string) string {
	versionDir := GetVersionedDirectoryPath(config, version)
	return filepath.Join(versionDir, config.BinaryName)
}

// GetSymlinkTargetPath returns the relative path from symlink to target for proper symlink creation
func GetSymlinkTargetPath(config FileConfig, version string) string {
	if config.UseVersionsSubdirectory {
		// New pattern: versions/{ProjectName}/{version}/{binary}
		projectName := config.ProjectName
		if projectName == "" {
			projectName = config.BinaryName
		}
		return filepath.Join("versions", projectName, version, config.BinaryName)
	} else {
		// Legacy pattern: {VersionedDirectoryName}/{version}/{binary}
		return filepath.Join(config.VersionedDirectoryName, version, config.BinaryName)
	}
}

// GetInstalledBinaryPath returns the preferred path to the installed binary
// Prefers symlink path when available, falls back to versioned directory path
func GetInstalledBinaryPath(config FileConfig, version string) (string, error) {
	localSymlinkPath := filepath.Join(config.BaseBinaryDirectory, config.BinaryName)
	versionedPath := GetVersionedBinaryPath(config, version)

	// Prefer local symlink if it exists and points to the correct version
	if config.CreateLocalSymlink && FileExists(localSymlinkPath) {
		if resolvedPath, err := os.Readlink(localSymlinkPath); err == nil {
			// For new pattern, resolve relative symlinks
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(config.BaseBinaryDirectory, resolvedPath)
			}
			if resolvedPath == versionedPath {
				return localSymlinkPath, nil
			}
		}
	}

	// Fall back to versioned path
	if FileExists(versionedPath) {
		return versionedPath, nil
	}

	return "", fmt.Errorf("binary not found at expected locations: %s or %s", localSymlinkPath, versionedPath)
}

// GetInstallationInfo returns comprehensive information about an installed binary
func GetInstallationInfo(config FileConfig, version string) (*InstallationInfo, error) {
	localSymlinkPath := filepath.Join(config.BaseBinaryDirectory, config.BinaryName)
	globalSymlinkPath := filepath.Join("/usr/local/bin", config.BinaryName)
	versionedPath := GetVersionedBinaryPath(config, version)

	info := &InstallationInfo{
		Version:             version,
		LocalSymlinkPath:    localSymlinkPath,
		GlobalSymlinkPath:   globalSymlinkPath,
		VersionedPath:       versionedPath,
		GlobalSymlinkNeeded: config.CreateGlobalSymlink,
	}

	// Determine installation type
	if config.IsDirectBinary {
		info.InstallationType = "direct_binary"
	} else {
		info.InstallationType = "extracted_archive"
	}

	// Check local symlink status
	if config.CreateLocalSymlink {
		if FileExists(localSymlinkPath) {
			if resolvedPath, err := os.Readlink(localSymlinkPath); err == nil {
				// For new pattern, resolve relative symlinks
				if !filepath.IsAbs(resolvedPath) {
					resolvedPath = filepath.Join(config.BaseBinaryDirectory, resolvedPath)
				}
				if resolvedPath == versionedPath {
					info.LocalSymlinkCreated = true
					info.SymlinkStatus = "created"
					info.BinaryPath = localSymlinkPath
				} else {
					info.SymlinkStatus = "failed"
					info.BinaryPath = versionedPath
				}
			} else {
				info.SymlinkStatus = "failed"
				info.BinaryPath = versionedPath
			}
		} else {
			info.SymlinkStatus = "failed"
			info.BinaryPath = versionedPath
		}
	} else {
		info.SymlinkStatus = "disabled"
		info.BinaryPath = versionedPath
	}

	// Verify binary exists
	if !FileExists(info.BinaryPath) {
		return nil, fmt.Errorf("binary not found at expected path: %s", info.BinaryPath)
	}

	return info, nil
}

// FindBinary searches for a specific binary file in a given directory and its subdirectories.
// Returns the absolute path to the binary if found, otherwise an error if the binary is not found or an issue occurs.
func FindBinary(directory, binaryName string) (string, error) {
	var binaryPath string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Match the binary name
		if info.Mode().IsRegular() && info.Name() == binaryName {
			binaryPath = path
			return filepath.SkipDir // Stop searching once found
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if binaryPath == "" {
		return "", fmt.Errorf("binary %s not found in extracted files", binaryName)
	}
	return binaryPath, nil
}

// UpdateSymlink updates the symlink to point to the latest target.
// - `target` is the file for the symlink to point to (can be relative or absolute).
// - `symlinkPath` is the path where the symlink should be created.
func UpdateSymlink(target, symlinkPath string) error {
	// For relative targets, verify the target exists relative to the symlink directory
	var targetToCheck string
	if filepath.IsAbs(target) {
		targetToCheck = target
	} else {
		// For relative paths, resolve relative to the symlink directory
		symlinkDir := filepath.Dir(symlinkPath)
		targetToCheck = filepath.Join(symlinkDir, target)
	}

	// Verify target exists
	if !FileExists(targetToCheck) {
		return fmt.Errorf("target file does not exist: %s", targetToCheck)
	}

	// Remove the symlink if it already exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %v", err)
		}
	}

	// Create the new symlink
	if err := os.Symlink(target, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}

	// Verify the symlink
	resolvedPath, err := os.Readlink(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to verify symlink: %v", err)
	}
	if resolvedPath != target {
		return fmt.Errorf("symlink was not set correctly: expected %s, got %s", target, resolvedPath)
	}

	return nil
}

// TryUpdateSymlink attempts to update a symlink with graceful fallback
// Returns true if symlink was created successfully, false if it failed
// Logs warnings for failures but doesn't return errors (graceful fallback)
func TryUpdateSymlink(target, symlinkPath string) bool {
	if err := UpdateSymlink(target, symlinkPath); err != nil {
		fmt.Printf("Warning: Failed to create symlink %s -> %s: %v\n", symlinkPath, target, err)
		fmt.Printf("Binary is still available at: %s\n", target)
		return false
	}
	return true
}

// DownloadFile downloads a file from the given URL to the specified path
func DownloadFile(link string, destination string) error {
	resp, err := http.Get(link)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	out, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// InstallBinary extracts an archive and installs the binary into a versioned folder with a symlink.
// If IsDirectBinary is true, it handles direct binary files instead of archives.
func InstallBinary(fileConfig FileConfig, version string) error {
	if fileConfig.IsDirectBinary {
		return InstallDirectBinary(fileConfig, version)
	}
	return InstallArchivedBinary(fileConfig, version)
}

// InstallDirectBinary installs a direct binary file (not archived) into a versioned folder with enhanced symlink control.
func InstallDirectBinary(fileConfig FileConfig, version string) error {
	// Apply defaults for backward compatibility
	config := fileConfig
	if config.CreateLocalSymlink == false && config.CreateGlobalSymlink == false {
		// If both are false, assume this is an old config and enable local symlinks by default
		config.CreateLocalSymlink = true
	}

	versionDir := GetVersionedDirectoryPath(config, version)
	localSymlinkPath := filepath.Join(config.BaseBinaryDirectory, config.BinaryName)
	globalSymlinkPath := filepath.Join("/usr/local/bin", config.BinaryName)

	// Step 1: Create version directory
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %v", err)
	}

	// Step 2: Install the binary to the versioned folder
	fmt.Println("Installing the binary...")
	finalBinaryPath := filepath.Join(versionDir, config.BinaryName)

	// Validate that we're not trying to extract a direct binary
	if !config.IsDirectBinary {
		return fmt.Errorf("InstallDirectBinary called but IsDirectBinary is false - this indicates a configuration error")
	}

	// Copy the downloaded binary to the final location
	if err := copyFile(config.SourceArchivePath, finalBinaryPath); err != nil {
		return fmt.Errorf("failed to copy binary to versioned directory: %v", err)
	}

	// Make the binary executable
	if err := os.Chmod(finalBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Step 3: Create/update local symlink (with graceful fallback)
	localSymlinkCreated := false
	if config.CreateLocalSymlink {
		fmt.Println("Creating local symlink...")
		symlinkTarget := GetSymlinkTargetPath(config, version)
		localSymlinkCreated = TryUpdateSymlink(symlinkTarget, localSymlinkPath)
		if localSymlinkCreated {
			fmt.Printf("Local symlink created: %s -> %s\n", localSymlinkPath, symlinkTarget)
		}
	} else {
		fmt.Println("Local symlink creation disabled")
	}

	// Step 4: Handle global symlink (provide instructions)
	if config.CreateGlobalSymlink {
		fmt.Println("Global symlink requested...")
		if localSymlinkCreated {
			fmt.Println("To create global symlink, run:")
			fmt.Printf("sudo ln -s %s %s\n", localSymlinkPath, globalSymlinkPath)
		} else {
			fmt.Println("To create global symlink, run:")
			fmt.Printf("sudo ln -s %s %s\n", finalBinaryPath, globalSymlinkPath)
		}
	}

	fmt.Println("Installation successful!")
	fmt.Printf("Binary installed at: %s\n", finalBinaryPath)
	if localSymlinkCreated {
		fmt.Printf("Available via symlink: %s\n", localSymlinkPath)
	}

	return nil
}

// InstallArchivedBinary extracts an archive and installs the binary into a versioned folder with enhanced symlink control.
func InstallArchivedBinary(fileConfig FileConfig, version string) error {
	return InstallArchivedBinaryWithConfig(fileConfig, version, nil)
}

// InstallArchivedBinaryWithConfig extracts an archive with enhanced configuration and installs the binary
func InstallArchivedBinaryWithConfig(fileConfig FileConfig, version string, extractionConfig *ExtractionConfig) error {
	// Apply defaults for backward compatibility
	config := fileConfig
	if config.CreateLocalSymlink == false && config.CreateGlobalSymlink == false {
		// If both are false, assume this is an old config and enable local symlinks by default
		config.CreateLocalSymlink = true
	}

	versionDir := GetVersionedDirectoryPath(config, version)
	localSymlinkPath := filepath.Join(config.BaseBinaryDirectory, config.BinaryName)
	globalSymlinkPath := filepath.Join("/usr/local/bin", config.BinaryName)

	// Validate that we're trying to extract an archive
	if config.IsDirectBinary {
		return fmt.Errorf("InstallArchivedBinary called but IsDirectBinary is true - this indicates a configuration error")
	}

	// Step 1: Extract the archive with enhanced configuration
	handler := archiver.NewArchiveHandler()
	fmt.Printf("Extracting %s...\n", config.SourceArchivePath)

	// Convert our ExtractionConfig to archiver.ExtractionConfig if needed
	var archiverConfig *archiver.ExtractionConfig
	if extractionConfig != nil {
		archiverConfig = &archiver.ExtractionConfig{
			StripComponents: extractionConfig.StripComponents,
			BinaryPath:      extractionConfig.BinaryPath,
		}
	}

	if err := handler.ExtractArchiveWithConfig(config.SourceArchivePath, versionDir, archiverConfig); err != nil {
		return fmt.Errorf("failed to extract archive: %v", err)
	}

	// Step 2: Locate the binary file (with enhanced path handling)
	fmt.Println("Locating the binary...")
	var binaryPath string
	var err error

	if extractionConfig != nil && extractionConfig.BinaryPath != "" {
		// Use specific binary path from extraction config
		specificPath := extractionConfig.BinaryPath
		// Replace placeholders in binary path
		specificPath = strings.ReplaceAll(specificPath, "{os}", runtime.GOOS)
		// Note: MapArch is in release package, we'll need to handle this differently
		// For now, use runtime.GOARCH directly
		specificPath = strings.ReplaceAll(specificPath, "{arch}", runtime.GOARCH)

		binaryPath = filepath.Join(versionDir, specificPath)
		if !FileExists(binaryPath) {
			return fmt.Errorf("binary not found at specified path: %s", binaryPath)
		}
	} else {
		// Use standard binary finding logic
		binaryPath, err = FindBinary(versionDir, config.SourceBinaryName)
		if err != nil {
			return fmt.Errorf("failed to locate binary %s: %v", config.SourceBinaryName, err)
		}
	}

	// Step 3: Move the binary to the expected location
	fmt.Println("Installing the binary...")
	finalBinaryPath := filepath.Join(versionDir, config.BinaryName)
	if binaryPath != finalBinaryPath {
		if err := os.Rename(binaryPath, finalBinaryPath); err != nil {
			return fmt.Errorf("failed to move binary to versioned directory: %v", err)
		}
	}

	// Make the binary executable
	if err := os.Chmod(finalBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Step 4: Create/update local symlink (with graceful fallback)
	localSymlinkCreated := false
	if config.CreateLocalSymlink {
		fmt.Println("Creating local symlink...")
		symlinkTarget := GetSymlinkTargetPath(config, version)
		localSymlinkCreated = TryUpdateSymlink(symlinkTarget, localSymlinkPath)
		if localSymlinkCreated {
			fmt.Printf("Local symlink created: %s -> %s\n", localSymlinkPath, symlinkTarget)
		}
	} else {
		fmt.Println("Local symlink creation disabled")
	}

	// Step 5: Handle global symlink (provide instructions)
	if config.CreateGlobalSymlink {
		fmt.Println("Global symlink requested...")
		if localSymlinkCreated {
			fmt.Println("To create global symlink, run:")
			fmt.Printf("sudo ln -s %s %s\n", localSymlinkPath, globalSymlinkPath)
		} else {
			fmt.Println("To create global symlink, run:")
			fmt.Printf("sudo ln -s %s %s\n", finalBinaryPath, globalSymlinkPath)
		}
	}

	fmt.Println("Installation successful!")
	fmt.Printf("Binary installed at: %s\n", finalBinaryPath)
	if localSymlinkCreated {
		fmt.Printf("Available via symlink: %s\n", localSymlinkPath)
	}

	return nil
}

// FileExists checks if the given file exists and is not a directory
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	// Ensure it's a file, not a directory or other type
	return err == nil && !info.IsDir()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}
