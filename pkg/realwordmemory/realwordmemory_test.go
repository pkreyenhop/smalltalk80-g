package realwordmemory_test

import (
	"testing"

	"smalltalk80/pkg/realwordmemory"
)

func TestRealWordMemory(t *testing.T) {
	mem := realwordmemory.New()

	mem.SegmentWordPut(0, 10, 0x1234)
	if val := mem.SegmentWord(0, 10); val != 0x1234 {
		t.Fatalf("expected 0x1234, got 0x%x", val)
	}

	b0 := mem.SegmentWordByte(0, 10, 0)
	b1 := mem.SegmentWordByte(0, 10, 1)
	if b0 != 0x34 || b1 != 0x12 {
		t.Fatalf("expected b0=0x34, b1=0x12, got b0=0x%x, b1=0x%x", b0, b1)
	}

	mem.SegmentWordBytePut(0, 10, 0, 0x78)
	if val := mem.SegmentWord(0, 10); val != 0x1278 {
		t.Fatalf("expected 0x1278, got 0x%x", val)
	}

	// Test MSB bit extraction
	// 0x8000 = bit 0 set
	mem.SegmentWordPut(1, 5, 0x8000)
	if bits := mem.SegmentWordBitsTo(1, 5, 0, 0); bits != 1 {
		t.Fatalf("expected bits 0..0 to be 1, got %d", bits)
	}
	if bits := mem.SegmentWordBitsTo(1, 5, 1, 15); bits != 0 {
		t.Fatalf("expected bits 1..15 to be 0, got %d", bits)
	}

	mem.SegmentWordBitsToPut(1, 5, 0, 3, 0xF)
	if val := mem.SegmentWord(1, 5); val != 0xF000 {
		t.Fatalf("expected 0xF000, got 0x%x", val)
	}
}
