package fileUtils

import (
	"errors"
	"fmt"
	"gitlab.com/locke-codes/go-binary-updater/pkg/archiver"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FileConfig struct {
	VersionedDirectoryName string `json:"versioned_directory_name"`
	SourceBinaryName       string `json:"source_binary_name"`
	BinaryName             string `json:"binary_name"`
	CreateGlobalSymlink    bool   `json:"create_global_symlink"`
	BaseBinaryDirectory    string `json:"base_binary_directory"`
	SourceArchivePath      string `json:"source_archive_path"`
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
// - `target` is the file for the symlink to point to.
// - `symlinkPath` is the path where the symlink should be created.
func UpdateSymlink(target, symlinkPath string) error {
	// Verify target exists
	if !FileExists(target) {
		fmt.Printf("DEBUG: FileExists returned false for target: %s\n", target)
		return errors.New(fmt.Sprintf("target file does not exist: %s", target))
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
func InstallBinary(fileConfig FileConfig, version string) error {
	versionDir := filepath.Join(fileConfig.BaseBinaryDirectory, fileConfig.VersionedDirectoryName, version)
	localSymlinkPath := filepath.Join(fileConfig.BaseBinaryDirectory, fileConfig.BinaryName)
	globalSymlinkPath := filepath.Join("/usr/local/bin", fileConfig.BinaryName)

	// Step 1: Extract the archive
	handler := archiver.NewArchiveHandler()
	fmt.Printf("Extracting %s...\n", fileConfig.SourceArchivePath)
	if err := handler.ExtractArchive(fileConfig.SourceArchivePath, versionDir); err != nil {
		return fmt.Errorf("failed to extract archive: %v", err)
	}

	// Step 2: Locate the binary file
	fmt.Println("Locating the binary...")
	binaryPath, err := FindBinary(versionDir, fileConfig.SourceBinaryName)
	if err != nil {
		return fmt.Errorf("failed to locate binary %s: %v", fileConfig.SourceBinaryName, err)
	}

	// Step 3: Move the binary to the versioned folder
	fmt.Println("Installing the binary...")
	finalBinaryPath := filepath.Join(versionDir, fileConfig.SourceBinaryName)
	if err := os.Rename(binaryPath, finalBinaryPath); err != nil {
		return fmt.Errorf("failed to move binary to versioned directory: %v", err)
	}

	// Make the binary executable
	if err := os.Chmod(finalBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Step 4: Create/update the symlink in ~/.local/bin
	fmt.Println("Updating local symlink...")
	if err := UpdateSymlink(finalBinaryPath, localSymlinkPath); err != nil {
		return fmt.Errorf("failed to update local symlink: %v", err)
	}

	if fileConfig.CreateGlobalSymlink {
		// Step 5: Create/update the global symlink in /usr/local/bin
		fmt.Println("Updating global symlink...")
		// For now just output the command for the symlink. If the user already has
		// ~/.local/bin in path then it should already work
		fmt.Println("You must either ensure that ~/.local/bin is in your path or run the following command:")
		fmt.Printf("sudo ln -s %s %s\n", localSymlinkPath, globalSymlinkPath)
		//if err := updateSymlink(localSymlinkPath, globalSymlinkPath); err != nil {
		//	return fmt.Errorf("failed to update global symlink: %v", err)
		//}
	}

	fmt.Println("Installation successful!")
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
