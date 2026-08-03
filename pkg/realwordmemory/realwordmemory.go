package realwordmemory

const (
	SegmentCount = 16
	SegmentSize  = 65536 // in words
)

// RealWordMemory represents the segmented memory model (G&R pg. 656)
type RealWordMemory struct {
	memory [SegmentCount][SegmentSize]uint16
}

func New() *RealWordMemory {
	return &RealWordMemory{}
}

func (r *RealWordMemory) SegmentWord(s, w int) uint16 {
	return r.memory[s][w]
}

func (r *RealWordMemory) SegmentWordPut(s, w int, value uint16) uint16 {
	r.memory[s][w] = value
	return value
}

func (r *RealWordMemory) SegmentWordByte(s, w, byteNumber int) uint8 {
	if byteNumber == 0 {
		return uint8(r.memory[s][w] & 0xff)
	}
	return uint8((r.memory[s][w] >> 8) & 0xff)
}

func (r *RealWordMemory) SegmentWordBytePut(s, w, byteNumber int, value uint8) uint8 {
	if byteNumber == 0 {
		r.memory[s][w] = (r.memory[s][w] & 0xff00) | uint16(value)
	} else {
		r.memory[s][w] = (r.memory[s][w] & 0x00ff) | (uint16(value) << 8)
	}
	return value
}

// SegmentWordBitsTo extracts bits from firstBitIndex to lastBitIndex.
// Bit 0 is MSB (bit 15 in standard uint16), Bit 15 is LSB (bit 0 in standard uint16).
func (r *RealWordMemory) SegmentWordBitsTo(s, w, firstBitIndex, lastBitIndex int) uint16 {
	shift := r.memory[s][w] >> (15 - lastBitIndex)
	mask := uint16((1 << (lastBitIndex - firstBitIndex + 1)) - 1)
	return shift & mask
}

func (r *RealWordMemory) SegmentWordBitsToPut(s, w, firstBitIndex, lastBitIndex int, value uint16) uint16 {
	mask := uint16((1 << (lastBitIndex - firstBitIndex + 1)) - 1)
	valShifted := (value & mask) << (15 - lastBitIndex)
	clearMask := ^(mask << (15 - lastBitIndex))
	r.memory[s][w] = (r.memory[s][w] & clearMask) | valShifted
	return value
}
