//go:build windows

package infra

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestReplaceProjectConfigFileWindowsReplacesExistingFile(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "replacement.toml")
	destination := filepath.Join(directory, projectConfigFileName)
	mustWrite(t, source, "new project config\n")
	mustWrite(t, destination, "old project config\n")

	if err := replaceProjectConfigFile(source, destination); err != nil {
		t.Fatalf("replaceProjectConfigFile(%s, %s): %v", source, destination, err)
	}
	assertFileBytes(t, destination, []byte("new project config\n"))
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("replacement source still exists after success: %v", err)
	}
}

func TestReplaceProjectConfigFileWindowsFailurePreservesBothFiles(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "replacement.toml")
	destination := filepath.Join(directory, projectConfigFileName)
	sourceBytes := []byte("new project config\n")
	destinationBytes := []byte("old project config\n")
	if err := os.WriteFile(source, sourceBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", source, err)
	}
	if err := os.WriteFile(destination, destinationBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", destination, err)
	}

	// syscall.Open omits FILE_SHARE_DELETE, so Windows must reject replacement
	// while this handle is open instead of exposing a partial destination.
	handle, err := syscall.Open(destination, syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("syscall.Open(%s): %v", destination, err)
	}
	defer syscall.CloseHandle(handle)

	if err := replaceProjectConfigFile(source, destination); err == nil {
		t.Fatal("replaceProjectConfigFile unexpectedly replaced a delete-locked destination")
	}
	assertFileBytes(t, source, sourceBytes)
	assertFileBytes(t, destination, destinationBytes)
}
