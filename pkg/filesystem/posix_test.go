package filesystem_test

import (
	"os"
	"testing"

	"smalltalk80/pkg/filesystem"
)

func TestPosixST80FileSystem(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "st80_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs := filesystem.NewPosixST80FileSystem(tempDir)

	h := fs.CreateFile("test.txt")
	if h < 0 {
		t.Fatalf("CreateFile failed")
	}

	data := []byte("Hello Smalltalk!")
	if n := fs.Write(h, data); n != len(data) {
		t.Fatalf("Write expected %d, got %d", len(data), n)
	}

	if sz := fs.FileSize(h); sz != len(data) {
		t.Fatalf("FileSize expected %d, got %d", len(data), sz)
	}

	fs.SeekTo(h, 0)
	buf := make([]byte, len(data))
	if n := fs.Read(h, buf); n != len(data) || string(buf) != string(data) {
		t.Fatalf("Read expected %s, got %s (n=%d)", data, buf, n)
	}

	fs.CloseFile(h)

	var files []string
	fs.EnumerateFiles(func(fname string) {
		files = append(files, fname)
	})
	if len(files) != 1 || files[0] != "test.txt" {
		t.Fatalf("EnumerateFiles expected [test.txt], got %v", files)
	}

	if !fs.RenameFile("test.txt", "renamed.txt") {
		t.Fatalf("RenameFile failed")
	}

	if !fs.DeleteFile("renamed.txt") {
		t.Fatalf("DeleteFile failed")
	}
}
