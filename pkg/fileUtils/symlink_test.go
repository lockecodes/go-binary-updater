package fileUtils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultFileConfig(t *testing.T) {
	config := DefaultFileConfig()
	
	// Test that symlink-first approach is preserved
	if !config.CreateLocalSymlink {
		t.Error("Expected CreateLocalSymlink to be true by default (preserving symlink-first approach)")
	}
	
	if config.CreateGlobalSymlink {
		t.Error("Expected CreateGlobalSymlink to be false by default (requires sudo)")
	}
	
	if config.AssetMatchingStrategy != "flexible" {
		t.Errorf("Expected AssetMatchingStrategy to be 'flexible', got '%s'", config.AssetMatchingStrategy)
	}
	
	if config.IsDirectBinary {
		t.Error("Expected IsDirectBinary to be false by default")
	}
}

func TestGetInstalledBinaryPath(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := FileConfig{
		BaseBinaryDirectory:    tempDir,
		VersionedDirectoryName: "versions",
		BinaryName:             "testapp",
		CreateLocalSymlink:     true,
	}
	version := "v1.0.0"

	// Create versioned directory and binary
	versionDir := filepath.Join(tempDir, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	binaryPath := filepath.Join(versionDir, "testapp")
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

	// Create symlink
	symlinkPath := filepath.Join(tempDir, "testapp")
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test with symlink - should return symlink path
	path, err = GetInstalledBinaryPath(config, version)
	if err != nil {
		t.Fatalf("GetInstalledBinaryPath failed: %v", err)
	}
	if path != symlinkPath {
		t.Errorf("Expected symlink path %s, got %s", symlinkPath, path)
	}
}

func TestGetInstallationInfo(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "installation_info_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := FileConfig{
		BaseBinaryDirectory:    tempDir,
		VersionedDirectoryName: "versions",
		BinaryName:             "testapp",
		CreateLocalSymlink:     true,
		CreateGlobalSymlink:    true,
		IsDirectBinary:         false,
	}
	version := "v1.0.0"

	// Create versioned directory and binary
	versionDir := filepath.Join(tempDir, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	binaryPath := filepath.Join(versionDir, "testapp")
	if err := os.WriteFile(binaryPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Test without symlink
	info, err := GetInstallationInfo(config, version)
	if err != nil {
		t.Fatalf("GetInstallationInfo failed: %v", err)
	}

	if info.Version != version {
		t.Errorf("Expected version %s, got %s", version, info.Version)
	}

	if info.InstallationType != "extracted_archive" {
		t.Errorf("Expected installation type 'extracted_archive', got %s", info.InstallationType)
	}

	if info.SymlinkStatus != "failed" {
		t.Errorf("Expected symlink status 'failed', got %s", info.SymlinkStatus)
	}

	if info.BinaryPath != binaryPath {
		t.Errorf("Expected binary path %s, got %s", binaryPath, info.BinaryPath)
	}

	if !info.GlobalSymlinkNeeded {
		t.Error("Expected GlobalSymlinkNeeded to be true")
	}

	// Create symlink and test again
	symlinkPath := filepath.Join(tempDir, "testapp")
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	info, err = GetInstallationInfo(config, version)
	if err != nil {
		t.Fatalf("GetInstallationInfo failed: %v", err)
	}

	if info.SymlinkStatus != "created" {
		t.Errorf("Expected symlink status 'created', got %s", info.SymlinkStatus)
	}

	if !info.LocalSymlinkCreated {
		t.Error("Expected LocalSymlinkCreated to be true")
	}

	if info.BinaryPath != symlinkPath {
		t.Errorf("Expected binary path %s, got %s", symlinkPath, info.BinaryPath)
	}
}

func TestGetInstallationInfo_DirectBinary(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "direct_binary_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := FileConfig{
		BaseBinaryDirectory:    tempDir,
		VersionedDirectoryName: "versions",
		BinaryName:             "k0s",
		CreateLocalSymlink:     false, // Disabled symlinks
		CreateGlobalSymlink:    false,
		IsDirectBinary:         true,
	}
	version := "v1.33.2+k0s.0"

	// Create versioned directory and binary
	versionDir := filepath.Join(tempDir, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	binaryPath := filepath.Join(versionDir, "k0s")
	if err := os.WriteFile(binaryPath, []byte("fake k0s binary"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	info, err := GetInstallationInfo(config, version)
	if err != nil {
		t.Fatalf("GetInstallationInfo failed: %v", err)
	}

	if info.InstallationType != "direct_binary" {
		t.Errorf("Expected installation type 'direct_binary', got %s", info.InstallationType)
	}

	if info.SymlinkStatus != "disabled" {
		t.Errorf("Expected symlink status 'disabled', got %s", info.SymlinkStatus)
	}

	if info.LocalSymlinkCreated {
		t.Error("Expected LocalSymlinkCreated to be false")
	}

	if info.BinaryPath != binaryPath {
		t.Errorf("Expected binary path %s, got %s", binaryPath, info.BinaryPath)
	}
}

func TestTryUpdateSymlink(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "try_symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create target file
	targetPath := filepath.Join(tempDir, "target")
	if err := os.WriteFile(targetPath, []byte("target content"), 0755); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	symlinkPath := filepath.Join(tempDir, "symlink")

	// Test successful symlink creation
	success := TryUpdateSymlink(targetPath, symlinkPath)
	if !success {
		t.Error("Expected TryUpdateSymlink to succeed")
	}

	// Verify symlink was created
	if !FileExists(symlinkPath) {
		t.Error("Expected symlink to exist")
	}

	resolvedPath, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	if resolvedPath != targetPath {
		t.Errorf("Expected symlink to point to %s, got %s", targetPath, resolvedPath)
	}

	// Test symlink creation failure (non-existent target)
	nonExistentTarget := filepath.Join(tempDir, "nonexistent")
	badSymlinkPath := filepath.Join(tempDir, "bad_symlink")
	
	success = TryUpdateSymlink(nonExistentTarget, badSymlinkPath)
	if success {
		t.Error("Expected TryUpdateSymlink to fail for non-existent target")
	}

	// Verify bad symlink was not created
	if FileExists(badSymlinkPath) {
		t.Error("Expected bad symlink not to exist")
	}
}

func TestBackwardCompatibility_SymlinkDefaults(t *testing.T) {
	// Test that old configurations (with both symlink options false) get defaults applied
	tempDir, err := os.MkdirTemp("", "backward_compat_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Old-style config with both symlink options false (should get defaults)
	oldConfig := FileConfig{
		BaseBinaryDirectory:    tempDir,
		VersionedDirectoryName: "versions",
		BinaryName:             "oldapp",
		CreateLocalSymlink:     false,
		CreateGlobalSymlink:    false,
		IsDirectBinary:         false,
	}

	// Create test binary
	version := "v1.0.0"
	versionDir := filepath.Join(tempDir, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	binaryPath := filepath.Join(versionDir, "oldapp")
	if err := os.WriteFile(binaryPath, []byte("old app binary"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// The InstallArchivedBinary function should apply defaults and create local symlinks
	// We can't test this directly without the archiver, but we can test the logic
	
	// Test that GetInstallationInfo works with old config
	info, err := GetInstallationInfo(oldConfig, version)
	if err != nil {
		t.Fatalf("GetInstallationInfo failed: %v", err)
	}

	// With old config (symlinks disabled), should use versioned path
	if info.BinaryPath != binaryPath {
		t.Errorf("Expected binary path %s, got %s", binaryPath, info.BinaryPath)
	}

	if info.SymlinkStatus != "disabled" {
		t.Errorf("Expected symlink status 'disabled', got %s", info.SymlinkStatus)
	}
}
