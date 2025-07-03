package fileUtils

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVersionsSubdirectoryPathConstruction tests the new path construction functions
func TestVersionsSubdirectoryPathConstruction(t *testing.T) {
	tests := []struct {
		name                    string
		config                  FileConfig
		version                 string
		expectedVersionedDir    string
		expectedVersionedBinary string
		expectedSymlinkTarget   string
	}{
		{
			name: "Legacy pattern (UseVersionsSubdirectory=false)",
			config: FileConfig{
				BaseBinaryDirectory:     "/home/user/.local/bin",
				VersionedDirectoryName:  "versions",
				BinaryName:              "helm",
				ProjectName:             "helm",
				UseVersionsSubdirectory: false,
			},
			version:                 "v3.18.3",
			expectedVersionedDir:    "/home/user/.local/bin/versions/v3.18.3",
			expectedVersionedBinary: "/home/user/.local/bin/versions/v3.18.3/helm",
			expectedSymlinkTarget:   "versions/v3.18.3/helm",
		},
		{
			name: "New pattern (UseVersionsSubdirectory=true)",
			config: FileConfig{
				BaseBinaryDirectory:     "/home/user/.local/bin",
				VersionedDirectoryName:  "versions",
				BinaryName:              "helm",
				ProjectName:             "helm",
				UseVersionsSubdirectory: true,
			},
			version:                 "v3.18.3",
			expectedVersionedDir:    "/home/user/.local/bin/versions/helm/v3.18.3",
			expectedVersionedBinary: "/home/user/.local/bin/versions/helm/v3.18.3/helm",
			expectedSymlinkTarget:   "versions/helm/v3.18.3/helm",
		},
		{
			name: "New pattern with different ProjectName",
			config: FileConfig{
				BaseBinaryDirectory:     "/home/user/.local/bin",
				VersionedDirectoryName:  "versions",
				BinaryName:              "kubectl",
				ProjectName:             "kubernetes",
				UseVersionsSubdirectory: true,
			},
			version:                 "v1.31.3",
			expectedVersionedDir:    "/home/user/.local/bin/versions/kubernetes/v1.31.3",
			expectedVersionedBinary: "/home/user/.local/bin/versions/kubernetes/v1.31.3/kubectl",
			expectedSymlinkTarget:   "versions/kubernetes/v1.31.3/kubectl",
		},
		{
			name: "New pattern without ProjectName (fallback to BinaryName)",
			config: FileConfig{
				BaseBinaryDirectory:     "/home/user/.local/bin",
				VersionedDirectoryName:  "versions",
				BinaryName:              "k0s",
				ProjectName:             "", // Empty ProjectName
				UseVersionsSubdirectory: true,
			},
			version:                 "v1.33.2+k0s.0",
			expectedVersionedDir:    "/home/user/.local/bin/versions/k0s/v1.33.2+k0s.0",
			expectedVersionedBinary: "/home/user/.local/bin/versions/k0s/v1.33.2+k0s.0/k0s",
			expectedSymlinkTarget:   "versions/k0s/v1.33.2+k0s.0/k0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetVersionedDirectoryPath
			versionedDir := GetVersionedDirectoryPath(tt.config, tt.version)
			if versionedDir != tt.expectedVersionedDir {
				t.Errorf("GetVersionedDirectoryPath() = %v, want %v", versionedDir, tt.expectedVersionedDir)
			}

			// Test GetVersionedBinaryPath
			versionedBinary := GetVersionedBinaryPath(tt.config, tt.version)
			if versionedBinary != tt.expectedVersionedBinary {
				t.Errorf("GetVersionedBinaryPath() = %v, want %v", versionedBinary, tt.expectedVersionedBinary)
			}

			// Test GetSymlinkTargetPath
			symlinkTarget := GetSymlinkTargetPath(tt.config, tt.version)
			if symlinkTarget != tt.expectedSymlinkTarget {
				t.Errorf("GetSymlinkTargetPath() = %v, want %v", symlinkTarget, tt.expectedSymlinkTarget)
			}
		})
	}
}

// TestVersionsSubdirectoryInstallation tests the complete installation process with new directory structure
func TestVersionsSubdirectoryInstallation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "versions_subdirectory_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test binary file
	testBinaryContent := []byte("#!/bin/bash\necho 'test binary'\n")
	sourceBinaryPath := filepath.Join(tempDir, "test-binary")
	if err := os.WriteFile(sourceBinaryPath, testBinaryContent, 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	config := FileConfig{
		BaseBinaryDirectory:     tempDir,
		VersionedDirectoryName:  "versions",
		SourceBinaryName:        "test-binary",
		BinaryName:              "testapp",
		ProjectName:             "testproject",
		SourceArchivePath:       sourceBinaryPath,
		CreateLocalSymlink:      true,
		CreateGlobalSymlink:     false,
		UseVersionsSubdirectory: true,
		IsDirectBinary:          true,
	}

	version := "v1.0.0"

	// Install the binary
	err = InstallDirectBinary(config, version)
	if err != nil {
		t.Fatalf("InstallDirectBinary failed: %v", err)
	}

	// Verify directory structure
	expectedVersionDir := filepath.Join(tempDir, "versions", "testproject", version)
	t.Logf("Checking for version directory: %s", expectedVersionDir)

	// List what was actually created
	if entries, err := os.ReadDir(tempDir); err == nil {
		t.Logf("Contents of %s:", tempDir)
		for _, entry := range entries {
			t.Logf("  %s (dir: %t)", entry.Name(), entry.IsDir())
			if entry.IsDir() && entry.Name() == "versions" {
				versionsDir := filepath.Join(tempDir, "versions")
				if subEntries, err := os.ReadDir(versionsDir); err == nil {
					t.Logf("  Contents of versions/:")
					for _, subEntry := range subEntries {
						t.Logf("    %s (dir: %t)", subEntry.Name(), subEntry.IsDir())
						if subEntry.IsDir() {
							projectDir := filepath.Join(versionsDir, subEntry.Name())
							if projectEntries, err := os.ReadDir(projectDir); err == nil {
								t.Logf("    Contents of versions/%s/:", subEntry.Name())
								for _, projectEntry := range projectEntries {
									t.Logf("      %s (dir: %t)", projectEntry.Name(), projectEntry.IsDir())
								}
							}
						}
					}
				}
			}
		}
	}

	// Check if version directory exists (use os.Stat since FileExists is for files only)
	if stat, err := os.Stat(expectedVersionDir); err != nil {
		t.Errorf("Version directory not created: %s, error: %v", expectedVersionDir, err)
	} else if !stat.IsDir() {
		t.Errorf("Expected directory but found file: %s", expectedVersionDir)
	}

	expectedBinaryPath := filepath.Join(expectedVersionDir, "testapp")
	if !FileExists(expectedBinaryPath) {
		t.Errorf("Binary not installed: %s", expectedBinaryPath)
	}

	// Verify symlink
	expectedSymlinkPath := filepath.Join(tempDir, "testapp")
	if !FileExists(expectedSymlinkPath) {
		t.Errorf("Symlink not created: %s", expectedSymlinkPath)
	}

	// Verify symlink target
	symlinkTarget, err := os.Readlink(expectedSymlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	expectedTarget := "versions/testproject/v1.0.0/testapp"
	if symlinkTarget != expectedTarget {
		t.Errorf("Symlink target = %v, want %v", symlinkTarget, expectedTarget)
	}

	// Verify symlink resolves correctly
	resolvedPath := filepath.Join(tempDir, symlinkTarget)
	if resolvedPath != expectedBinaryPath {
		t.Errorf("Resolved symlink path = %v, want %v", resolvedPath, expectedBinaryPath)
	}
}

// TestVersionsSubdirectoryBackwardCompatibility tests that legacy configurations still work
func TestVersionsSubdirectoryBackwardCompatibility(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "backward_compatibility_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test binary file
	testBinaryContent := []byte("#!/bin/bash\necho 'legacy test binary'\n")
	sourceBinaryPath := filepath.Join(tempDir, "legacy-binary")
	if err := os.WriteFile(sourceBinaryPath, testBinaryContent, 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Legacy configuration (UseVersionsSubdirectory defaults to false)
	config := FileConfig{
		BaseBinaryDirectory:    tempDir,
		VersionedDirectoryName: "versions",
		SourceBinaryName:       "legacy-binary",
		BinaryName:             "legacyapp",
		ProjectName:            "legacyproject",
		SourceArchivePath:      sourceBinaryPath,
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    false,
		IsDirectBinary:         true,
		// UseVersionsSubdirectory: false (default)
	}

	version := "v1.0.0"

	// Install the binary
	err = InstallDirectBinary(config, version)
	if err != nil {
		t.Fatalf("InstallDirectBinary failed: %v", err)
	}

	// Verify legacy directory structure
	expectedVersionDir := filepath.Join(tempDir, "versions", version)
	if stat, err := os.Stat(expectedVersionDir); err != nil {
		t.Errorf("Legacy version directory not created: %s, error: %v", expectedVersionDir, err)
	} else if !stat.IsDir() {
		t.Errorf("Expected directory but found file: %s", expectedVersionDir)
	}

	expectedBinaryPath := filepath.Join(expectedVersionDir, "legacyapp")
	if !FileExists(expectedBinaryPath) {
		t.Errorf("Legacy binary not installed: %s", expectedBinaryPath)
	}

	// Verify legacy symlink
	expectedSymlinkPath := filepath.Join(tempDir, "legacyapp")
	if !FileExists(expectedSymlinkPath) {
		t.Errorf("Legacy symlink not created: %s", expectedSymlinkPath)
	}

	// Verify legacy symlink target
	symlinkTarget, err := os.Readlink(expectedSymlinkPath)
	if err != nil {
		t.Fatalf("Failed to read legacy symlink: %v", err)
	}

	expectedTarget := "versions/v1.0.0/legacyapp"
	if symlinkTarget != expectedTarget {
		t.Errorf("Legacy symlink target = %v, want %v", symlinkTarget, expectedTarget)
	}
}

// TestGetInstalledBinaryPathWithVersionsSubdirectory tests path resolution with new directory structure
func TestGetInstalledBinaryPathWithVersionsSubdirectory(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_resolution_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := FileConfig{
		BaseBinaryDirectory:     tempDir,
		VersionedDirectoryName:  "versions",
		BinaryName:              "testapp",
		ProjectName:             "testproject",
		CreateLocalSymlink:      true,
		UseVersionsSubdirectory: true,
	}
	version := "v1.0.0"

	// Create versioned directory and binary
	versionDir := GetVersionedDirectoryPath(config, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	binaryPath := GetVersionedBinaryPath(config, version)
	if err := os.WriteFile(binaryPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Test without symlink - should return versioned path
	path, err := GetInstalledBinaryPath(config, version)
	if err != nil {
		t.Fatalf("GetInstalledBinaryPath failed: %v", err)
	}
	if path != binaryPath {
		t.Errorf("Expected versioned path %s, got %s", binaryPath, path)
	}

	// Create symlink using new pattern
	symlinkPath := filepath.Join(tempDir, "testapp")
	symlinkTarget := GetSymlinkTargetPath(config, version)
	if err := os.Symlink(symlinkTarget, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test with symlink - should return symlink path
	path, err = GetInstalledBinaryPath(config, version)
	if err != nil {
		t.Fatalf("GetInstalledBinaryPath with symlink failed: %v", err)
	}
	if path != symlinkPath {
		t.Errorf("Expected symlink path %s, got %s", symlinkPath, path)
	}
}

// TestDefaultFileConfigVersionsSubdirectory tests that the default config has the correct setting
func TestDefaultFileConfigVersionsSubdirectory(t *testing.T) {
	config := DefaultFileConfig()
	
	// Should default to false for backward compatibility
	if config.UseVersionsSubdirectory {
		t.Error("DefaultFileConfig().UseVersionsSubdirectory should be false for backward compatibility")
	}
}
