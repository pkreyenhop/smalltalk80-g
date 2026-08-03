package objmemory_test

import (
	"testing"

	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/objmemory"
	"smalltalk80/pkg/oops"
)

func TestObjectMemoryAllocation(t *testing.T) {
	om := objmemory.New(nil, nil)

	// Test Integer object checks
	if !om.IsIntegerObject(1) || om.IsIntegerObject(2) {
		t.Fatalf("IsIntegerObject failed")
	}

	val := 42
	obj := om.IntegerObjectOf(val)
	if om.IntegerValueOf(obj) != val {
		t.Fatalf("expected integer value %d, got %d", val, om.IntegerValueOf(obj))
	}

	// Test snapshot loading from included files/snapshot.im
	fs := filesystem.NewPosixST80FileSystem("../../files")
	if !om.LoadSnapshot(fs, "snapshot.im") {
		t.Fatalf("failed to load snapshot.im from files")
	}

	if !om.HasObject(oops.NilPointer) {
		t.Fatalf("expected NilPointer object to exist in image")
	}

	if !om.HasObject(oops.TruePointer) {
		t.Fatalf("expected TruePointer object to exist in image")
	}
}
