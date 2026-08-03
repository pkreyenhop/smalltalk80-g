package bitblt_test

import (
	"testing"

	"smalltalk80/pkg/bitblt"
	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/objmemory"
	"smalltalk80/pkg/oops"
)

func TestBitBltInitialization(t *testing.T) {
	fs := filesystem.NewPosixST80FileSystem("../../files")
	om := objmemory.New(nil, nil)
	if !om.LoadSnapshot(fs, "snapshot.im") {
		t.Skip("snapshot.im not found, skipping bitblt test")
	}
	om.GarbageCollect()

	// Create a dummy form object
	formClass := oops.ClassDisplayScreenPointer
	destForm := om.InstantiateClass_withWords(formClass, 4)
	t.Logf("destForm OOP = %d (NilPointer = %d)", destForm, oops.NilPointer)
	if destForm == oops.NilPointer {
		t.Fatalf("failed to allocate destForm")
	}

	// Create a dummy bitmap for field 0
	bits := om.InstantiateClass_withWords(oops.ClassArrayPointer, (640*480)/16)
	om.StorePointer_ofObject_withValue(0, destForm, bits)

	// Set width and height in Form
	om.StorePointer_ofObject_withValue(1, destForm, om.IntegerObjectOf(640))
	om.StorePointer_ofObject_withValue(2, destForm, om.IntegerObjectOf(480))

	bb := bitblt.NewBitBlt(
		om, destForm, oops.NilPointer, oops.NilPointer, 3,
		0, 0, 100, 100,
		0, 0, 0, 0, 640, 480,
	)

	if !bb.CopyBits() {
		t.Fatalf("CopyBits failed")
	}

	x, y, w, h := bb.GetUpdatedBounds()
	if x != 0 || y != 0 || w != 100 || h != 100 {
		t.Fatalf("expected updated bounds (0,0,100,100), got (%d,%d,%d,%d)", x, y, w, h)
	}
}
