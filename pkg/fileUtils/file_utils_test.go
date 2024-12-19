package fileUtils

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestFindBinary(t *testing.T) {
	tmpDir := t.TempDir()

	dummyFileNames := []string{"file1.txt", "file2.txt", "file3.go", "someBinary"}
	for _, f := range dummyFileNames {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("Dummy content"), 0666)
		if err != nil {
			t.Fatalf("Failed to create dummy file for testing: %s", err)
		}
	}

	tests := []struct {
		name        string
		dir         string
		binaryName  string
		expected    string
		expectError bool
	}{
		{
			name:        "FileExists",
			dir:         tmpDir,
			binaryName:  "someBinary",
			expected:    filepath.Join(tmpDir, "someBinary"),
			expectError: false,
		},
		{
			name:        "FileDoesNotExist",
			dir:         tmpDir,
			binaryName:  "nonExistingBinary",
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FindBinary(tc.dir, tc.binaryName)
			if (err != nil) != tc.expectError {
				t.Errorf("FindBinary() error = %v, expectError %v", err, tc.expectError)
				return
			}
			if result != tc.expected {
				t.Errorf("FindBinary() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestUpdateSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		target      string
		symlinkPath string
		expected    string
		expectError bool
	}{
		{
			name:        "SymlinkCreatedSuccessfully",
			target:      filepath.Join(tmpDir, "target.txt"),
			symlinkPath: filepath.Join(tmpDir, "sym.txt"),
			expected:    filepath.Join(tmpDir, "target.txt"),
			expectError: false,
		},
		{
			name:        "SymlinkUpdateSuccessfully",
			target:      filepath.Join(tmpDir, "newTarget.txt"),
			symlinkPath: filepath.Join(tmpDir, "sym.txt"),
			expected:    filepath.Join(tmpDir, "newTarget.txt"),
			expectError: false,
		},
		{
			name:        "SymlinkFailsOnInvalidPath",
			target:      "invalidpath",
			symlinkPath: filepath.Join(tmpDir, "sym.txt"),
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if !tc.expectError {
				err := os.WriteFile(tc.target, []byte("this is some content"), 0666)
				if err != nil {
					t.Fatalf("Failed to create target file: %s", err)
				}
			}
			err = UpdateSymlink(tc.target, tc.symlinkPath)
			if (err != nil) != tc.expectError {
				t.Errorf("UpdateSymlink() error = %v, expectError %v", err, tc.expectError)
				return
			}
			if !tc.expectError {
				resolvedPath, _ := os.Readlink(tc.symlinkPath)
				if resolvedPath != tc.expected {
					t.Errorf("UpdateSymlink() = %v, want %v", resolvedPath, tc.expected)
				}
			}
		})
	}
}
func TestCheckFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only existing file
	err := os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("Some content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create dummy file for testing: %s", err)
	}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "FileExists",
			filePath: filepath.Join(tmpDir, "exists.txt"),
			expected: true,
		},
		{
			name:     "FileDoesNotExist",
			filePath: filepath.Join(tmpDir, "doesnotexist.txt"),
			expected: false,
		},
		{
			name:     "EmptyFilePath",
			filePath: "",
			expected: false,
		},
		{
			name:     "InvalidPath",
			filePath: "/invalid/path/doesnotexist.txt",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := FileExists(tc.filePath)
			if res != tc.expected {
				t.Errorf("FileExists(%q) = %v; expected %v", tc.filePath, res, tc.expected)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name        string
		link        string
		destination string
		expectError bool
	}{
		{
			name:        "SuccessfulDownload",
			link:        "https://google.com",
			destination: "test.txt",
			expectError: false,
		},
		{
			name:        "UnavailableLink",
			link:        "http://localhost/test.txt",
			destination: "test.txt",
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := DownloadFile(tc.link, tc.destination)
			if (err != nil) != tc.expectError {
				t.Errorf("DownloadFile() error = %v, expectError %v", err, tc.expectError)
			}
		})
	}
	_ = os.Remove("test.txt")
}

func createTestArchive(filePath, binaryName string) error {
	// Create the .tar.gz file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Create a test binary file in the archive
	binaryContent := []byte("#!/bin/bash\necho 'Hello World'\n")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0755,
		Size: int64(len(binaryContent)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(binaryContent); err != nil {
		return err
	}

	return nil
}

func TestInstallBinary(t *testing.T) {
	tempDir := t.TempDir()
	println(tempDir)

	// Create the test source.tar.gz file
	sourceArchivePath := path.Join(tempDir, "source.tar.gz")
	if err := createTestArchive(sourceArchivePath, "binary"); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tests := []struct {
		name        string
		version     string
		fileConfig  FileConfig
		expectError bool
	}{
		{
			name:    "SuccessfulInstall",
			version: "1.0.0",
			fileConfig: FileConfig{
				SourceArchivePath:      path.Join(tempDir, "source.tar.gz"),
				BaseBinaryDirectory:    tempDir,
				VersionedDirectoryName: "test",
				SourceBinaryName:       "binary",
				BinaryName:             "binary",
				CreateGlobalSymlink:    false,
			},
			expectError: false,
		},
		{
			name:    "NonExistentArchive",
			version: "1.0.0",
			fileConfig: FileConfig{
				SourceArchivePath:      path.Join(tempDir, "nonexistent.tar.gz"),
				BaseBinaryDirectory:    tempDir,
				VersionedDirectoryName: "test",
				SourceBinaryName:       "binary",
				BinaryName:             "binary",
				CreateGlobalSymlink:    false,
			},
			expectError: true,
		},
		{
			name:    "NonExistentBinary",
			version: "1.0.0",
			fileConfig: FileConfig{
				SourceArchivePath:      path.Join(tempDir, "source.tar.gz"),
				BaseBinaryDirectory:    tempDir,
				VersionedDirectoryName: "test",
				SourceBinaryName:       "nonexistent",
				BinaryName:             "binary",
				CreateGlobalSymlink:    false,
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := InstallBinary(tc.fileConfig, tc.version)
			if (err != nil) != tc.expectError {
				t.Errorf("InstallBinary() error = %v, expectError %v", err, tc.expectError)
			}
		})
	}
}
