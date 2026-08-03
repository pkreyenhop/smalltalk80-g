package bitblt

import (
	"smalltalk80/pkg/objmemory"
	"smalltalk80/pkg/oops"
)

var rightMasks = [17]uint16{
	0, 0x1, 0x3, 0x7, 0xF, 0x1F, 0x3F, 0x7F, 0xFF,
	0x1FF, 0x3FF, 0x7FF, 0xFFF, 0x1FFF, 0x3FFF, 0x7FFF, 0xFFFF,
}

const AllOnes uint16 = 0xFFFF

type BitBlt struct {
	memory          *objmemory.ObjectMemory
	destForm        int
	sourceForm      int
	halftoneForm    int
	combinationRule int
	destX           int
	destY           int
	width           int
	height          int
	sourceX         int
	sourceY         int
	clipX           int
	clipY           int
	clipWidth       int
	clipHeight      int

	sourceFormWidth  int
	sourceFormHeight int
	destFormWidth    int
	destFormHeight   int

	sourceBits           int
	sourceRaster         int
	destBits             int
	destRaster           int
	halftoneBits         int
	skew                 int
	preload              bool
	nWords               int
	hDir                 int
	vDir                 int
	sourceIndex          int
	sourceDelta          int
	destIndex            int
	destDelta            int
	sx, sy, dx, dy, w, h int

	sourceBitsWordLength int
	destBitsWordLength   int

	mask1, mask2, skewMask uint16

	updatedX, updatedY, updatedWidth, updatedHeight int
}

func NewBitBlt(
	memory *objmemory.ObjectMemory,
	destForm, sourceForm, halftoneForm, combinationRule,
	destX, destY, width, height,
	sourceX, sourceY, clipX, clipY, clipWidth, clipHeight int,
) *BitBlt {
	b := &BitBlt{
		memory:          memory,
		destForm:        destForm,
		sourceForm:      sourceForm,
		halftoneForm:    halftoneForm,
		combinationRule: combinationRule,
		destX:           destX,
		destY:           destY,
		width:           width,
		height:          height,
		sourceX:         sourceX,
		sourceY:         sourceY,
		clipX:           clipX,
		clipY:           clipY,
		clipWidth:       clipWidth,
		clipHeight:      clipHeight,
	}

	const widthInForm = 1
	const heightInForm = 2

	if sourceForm != oops.NilPointer {
		b.sourceFormWidth = memory.IntegerValueOf(memory.FetchWord_ofObject(widthInForm, sourceForm))
		b.sourceFormHeight = memory.IntegerValueOf(memory.FetchWord_ofObject(heightInForm, sourceForm))
	} else {
		b.sourceX = 0
		b.sourceY = 0
		b.sourceFormWidth = 0
		b.sourceFormHeight = 0
	}
	b.destFormWidth = memory.IntegerValueOf(memory.FetchWord_ofObject(widthInForm, destForm))
	b.destFormHeight = memory.IntegerValueOf(memory.FetchWord_ofObject(heightInForm, destForm))
	return b
}

func (b *BitBlt) GetUpdatedBounds() (x, y, w, h int) {
	return b.updatedX, b.updatedY, b.updatedWidth, b.updatedHeight
}

func (b *BitBlt) CopyBits() bool {
	b.clipRange()
	if b.w > 0 && b.h > 0 {
		b.updatedX = b.dx
		b.updatedY = b.dy
		b.updatedWidth = b.w
		b.updatedHeight = b.h
		b.computeMasks()

		if b.sourceForm != oops.NilPointer && b.formWordCount(b.sourceFormWidth, b.sourceFormHeight) != b.sourceBitsWordLength {
			return false
		}
		if b.formWordCount(b.destFormWidth, b.destFormHeight) != b.destBitsWordLength {
			return false
		}

		b.checkOverlap()
		b.calculateOffsets()
		b.copyLoop()
	} else {
		b.updatedX = 0
		b.updatedY = 0
		b.updatedWidth = 0
		b.updatedHeight = 0
	}
	return true
}

func (b *BitBlt) formWordCount(width, height int) int {
	return (width + 15) / 16 * height
}

func (b *BitBlt) clipRange() {
	if b.clipX < 0 {
		b.clipWidth += b.clipX
		b.clipX = 0
	}
	if b.clipY < 0 {
		b.clipHeight += b.clipY
		b.clipY = 0
	}
	if (b.clipX + b.clipWidth) > b.destFormWidth {
		b.clipWidth = b.destFormWidth - b.clipX
	}
	if (b.clipY + b.clipHeight) > b.destFormHeight {
		b.clipHeight = b.destFormHeight - b.clipY
	}

	if b.destX >= b.clipX {
		b.sx = b.sourceX
		b.dx = b.destX
		b.w = b.width
	} else {
		b.sx = b.sourceX + (b.clipX - b.destX)
		b.w = b.width - (b.clipX - b.destX)
		b.dx = b.clipX
	}
	if (b.dx + b.w) > (b.clipX + b.clipWidth) {
		b.w = b.w - ((b.dx + b.w) - (b.clipX + b.clipWidth))
	}

	if b.destY >= b.clipY {
		b.sy = b.sourceY
		b.dy = b.destY
		b.h = b.height
	} else {
		b.sy = b.sourceY + (b.clipY - b.destY)
		b.h = b.height - (b.clipY - b.destY)
		b.dy = b.clipY
	}
	if (b.dy + b.h) > (b.clipY + b.clipHeight) {
		b.h = b.h - ((b.dy + b.h) - (b.clipY + b.clipHeight))
	}

	if b.sourceForm == oops.NilPointer {
		return
	}

	if b.sx < 0 {
		b.dx = b.dx - b.sx
		b.w = b.w + b.sx
		b.sx = 0
	}
	if b.sx+b.w > b.sourceFormWidth {
		b.w = b.w - (b.sx + b.w - b.sourceFormWidth)
	}
	if b.sy < 0 {
		b.dy = b.dy - b.sy
		b.h = b.h + b.sy
		b.sy = 0
	}
	if b.sy+b.h > b.sourceFormHeight {
		b.h = b.h - (b.sy + b.h - b.sourceFormHeight)
	}
}

func (b *BitBlt) computeMasks() {
	const bitsInForm = 0

	b.destBits = b.memory.FetchPointer_ofObject(bitsInForm, b.destForm)
	b.destBitsWordLength = b.memory.FetchWordLengthOf(b.destBits)
	b.destRaster = (b.destFormWidth-1)/16 + 1

	if b.sourceForm != oops.NilPointer {
		b.sourceBits = b.memory.FetchPointer_ofObject(bitsInForm, b.sourceForm)
		b.sourceBitsWordLength = b.memory.FetchWordLengthOf(b.sourceBits)
		b.sourceRaster = (b.sourceFormWidth-1)/16 + 1
	} else {
		b.sourceBitsWordLength = 0
	}

	if b.halftoneForm != oops.NilPointer {
		b.halftoneBits = b.memory.FetchPointer_ofObject(bitsInForm, b.halftoneForm)
	}

	b.skew = (b.sx - b.dx) & 15
	startBits := 16 - (b.dx & 15)
	b.mask1 = rightMasks[startBits]

	endBits := 15 - ((b.dx + b.w - 1) & 15)
	b.mask2 = ^rightMasks[endBits]

	if b.skew == 0 {
		b.skewMask = 0
	} else {
		b.skewMask = rightMasks[16-b.skew]
	}

	if b.w < startBits {
		b.mask1 = b.mask1 & b.mask2
		b.mask2 = 0
		b.nWords = 1
	} else {
		b.nWords = (b.w - startBits + 15)/16 + 1
	}
}

func (b *BitBlt) checkOverlap() {
	b.hDir = 1
	b.vDir = 1

	if b.sourceForm == b.destForm && (b.dy >= b.sy) {
		if b.dy > b.sy {
			b.vDir = -1
			b.sy = b.sy + b.h - 1
			b.dy = b.dy + b.h - 1
		} else {
			if b.dx > b.sx {
				b.hDir = -1
				b.sx = b.sx + b.w - 1
				b.dx = b.dx + b.w - 1
				b.skewMask = ^b.skewMask
				t := b.mask1
				b.mask1 = b.mask2
				b.mask2 = t
			}
		}
	}
}

func (b *BitBlt) calculateOffsets() {
	b.preload = (b.sourceForm != oops.NilPointer) && b.skew != 0 && b.skew <= (b.sx&15)
	if b.hDir < 0 {
		b.preload = !b.preload
	}

	b.sourceIndex = b.sy*b.sourceRaster + (b.sx / 16)
	b.destIndex = b.dy*b.destRaster + (b.dx / 16)

	preloadInc := 0
	if b.preload {
		preloadInc = 1
	}
	b.sourceDelta = (b.sourceRaster * b.vDir) - ((b.nWords + preloadInc) * b.hDir)
	b.destDelta = (b.destRaster * b.vDir) - (b.nWords * b.hDir)
}

func (b *BitBlt) mergeWith(sourceWord, destinationWord uint16) uint16 {
	switch b.combinationRule {
	case 0:
		return 0
	case 1:
		return sourceWord & destinationWord
	case 2:
		return sourceWord & (^destinationWord)
	case 3:
		return sourceWord
	case 4:
		return (^sourceWord) & destinationWord
	case 5:
		return destinationWord
	case 6:
		return sourceWord ^ destinationWord
	case 7:
		return sourceWord | destinationWord
	case 8:
		return (^sourceWord) & (^destinationWord)
	case 9:
		return (^sourceWord) ^ destinationWord
	case 10:
		return ^destinationWord
	case 11:
		return sourceWord | (^destinationWord)
	case 12:
		return ^sourceWord
	case 13:
		return (^sourceWord) | destinationWord
	case 14:
		return (^sourceWord) | (^destinationWord)
	case 15:
		return AllOnes
	}
	return 0
}

func (b *BitBlt) copyLoop() {
	var prevWord uint16
	var thisWord uint16
	var skewWord uint16
	var mergeMask uint16
	var halftoneWord uint16
	var mergeWord uint16

	for i := 1; i <= b.h; i++ {
		if b.halftoneForm != oops.NilPointer {
			halftoneWord = uint16(b.memory.FetchWord_ofObject(b.dy&15, b.halftoneBits))
			b.dy += b.vDir
		} else {
			halftoneWord = AllOnes
		}
		skewWord = halftoneWord

		if b.preload {
			prevWord = uint16(b.memory.FetchWord_ofObject(b.sourceIndex, b.sourceBits))
			b.sourceIndex += b.hDir
		} else {
			prevWord = 0
		}
		mergeMask = b.mask1

		for word := 1; word <= b.nWords; word++ {
			if b.sourceForm != oops.NilPointer {
				prevWord = prevWord & b.skewMask
				if word <= b.sourceRaster && b.sourceIndex >= 0 && b.sourceIndex < b.sourceBitsWordLength {
					thisWord = uint16(b.memory.FetchWord_ofObject(b.sourceIndex, b.sourceBits))
				}
				skewWord = prevWord | (thisWord & (^b.skewMask))
				prevWord = thisWord
				skewWord = (skewWord << b.skew) | (skewWord >> (16 - b.skew))
			}

			if b.destIndex >= b.destBitsWordLength {
				return
			}

			destWord := uint16(b.memory.FetchWord_ofObject(b.destIndex, b.destBits))
			mergeWord = b.mergeWith(skewWord&halftoneWord, destWord)

			b.memory.StoreWord_ofObject_withValue(b.destIndex, b.destBits, int((mergeMask&mergeWord)|((^mergeMask)&destWord)))
			b.sourceIndex += b.hDir
			b.destIndex += b.hDir

			if word == (b.nWords - 1) {
				mergeMask = b.mask2
			} else {
				mergeMask = AllOnes
			}
		}

		b.sourceIndex += b.sourceDelta
		b.destIndex += b.destDelta
	}
}
