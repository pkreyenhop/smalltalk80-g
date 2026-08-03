package interpreter_test

import (
	"testing"

	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/interpreter"
)

func TestInterpreterInit(t *testing.T) {
	fs := filesystem.NewPosixST80FileSystem("../../files")
	interp := interpreter.New(nil, fs)

	if !interp.Init() {
		t.Fatalf("failed to initialize interpreter with snapshot.im")
	}

	if interp.Memory().CoreLeft() == 0 {
		t.Fatalf("expected core left to be non-zero after snapshot load")
	}
}
