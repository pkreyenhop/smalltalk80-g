package objmemory

import (
	"encoding/binary"
	"fmt"
	"runtime/debug"

	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/hal"
	"smalltalk80/pkg/oops"
	"smalltalk80/pkg/realwordmemory"
)

type GCNotification interface {
	PrepareForCollection()
	CollectionCompleted()
}

const (
	ObjectTableSegment = realwordmemory.SegmentCount - 1
	ObjectTableStart   = 0
	ObjectTableSize    = realwordmemory.SegmentSize - 2
	HugeSize           = 256
	FreePointerList    = ObjectTableStart + ObjectTableSize
	BigSize            = 20
	FirstFreeChunkListSize = BigSize + 1

	HeapSegmentCount = realwordmemory.SegmentCount - 1
	FirstHeapSegment = 0
	LastHeapSegment  = FirstHeapSegment + HeapSegmentCount - 1

	HeapSpaceStop = realwordmemory.SegmentSize - FirstFreeChunkListSize - 1
	HeaderSize    = 2

	FirstFreeChunkList = HeapSpaceStop + 1
	NonPointer         = 65535
	LastSpecialOop     = 52

	ObjectSpaceBaseInImage = 512
)

var DbgWatch int

type ObjectMemory struct {
	wordMemory     *realwordmemory.RealWordMemory
	currentSegment int
	freeWords      uint32
	freeOops       int

	gcNotification GCNotification
	hal            hal.HAL
}

func New(halInterface hal.HAL, notification GCNotification) *ObjectMemory {
	return &ObjectMemory{
		wordMemory:     realwordmemory.New(),
		currentSegment: -1,
		freeWords:      0,
		freeOops:       0,
		gcNotification: notification,
		hal:            halInterface,
	}
}

func (om *ObjectMemory) WordMemory() *realwordmemory.RealWordMemory {
	return om.wordMemory
}

func (om *ObjectMemory) OopsLeft() int {
	return om.freeOops
}

func (om *ObjectMemory) CoreLeft() uint32 {
	return om.freeWords
}

func (om *ObjectMemory) runtimeCheck(cond bool, msg string) {
	if !cond {
		debug.PrintStack()
		if om.hal != nil {
			om.hal.Error(fmt.Sprintf("RUNTIME ERROR: %s", msg))
		} else {
			panic(fmt.Sprintf("RUNTIME ERROR: %s", msg))
		}
	}
}

func (om *ObjectMemory) CantBeIntegerObject(objectPointer int) {
	om.runtimeCheck(!om.IsIntegerObject(objectPointer), "Object pointer is an integer")
}

func (om *ObjectMemory) IsIntegerObject(objectPointer int) bool {
	return (objectPointer & 1) == 1
}

func (om *ObjectMemory) IsIntegerValue(valueWord int) bool {
	return valueWord >= -16384 && valueWord <= 16383
}

func (om *ObjectMemory) IntegerValueOf(objectPointer int) int {
	return int(int16(objectPointer&0xfffe)) / 2
}

func (om *ObjectMemory) IntegerObjectOf(value int) int {
	return int(uint16((value << 1) | 1))
}

func (om *ObjectMemory) ot_bits_to(objectPointer, firstBitIndex, lastBitIndex int) int {
	om.CantBeIntegerObject(objectPointer)
	return int(om.wordMemory.SegmentWordBitsTo(ObjectTableSegment, ObjectTableStart+objectPointer, firstBitIndex, lastBitIndex))
}

func (om *ObjectMemory) ot_bits_to_put(objectPointer, firstBitIndex, lastBitIndex, value int) int {
	om.CantBeIntegerObject(objectPointer)
	om.wordMemory.SegmentWordBitsToPut(ObjectTableSegment, ObjectTableStart+objectPointer, firstBitIndex, lastBitIndex, uint16(value))
	return value
}

func (om *ObjectMemory) ot(objectPointer int) int {
	om.CantBeIntegerObject(objectPointer)
	return int(om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+objectPointer))
}

func (om *ObjectMemory) ot_put(objectPointer, value int) int {
	om.CantBeIntegerObject(objectPointer)
	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+objectPointer, uint16(value))
	return value
}

func (om *ObjectMemory) countBitsOf(objectPointer int) int {
	return om.ot_bits_to(objectPointer, 0, 7)
}

func (om *ObjectMemory) countBitsOf_put(objectPointer, value int) int {
	return om.ot_bits_to_put(objectPointer, 0, 7, value)
}

func (om *ObjectMemory) oddBitOf(objectPointer int) int {
	return om.ot_bits_to(objectPointer, 8, 8)
}

func (om *ObjectMemory) oddBitOf_put(objectPointer, value int) int {
	return om.ot_bits_to_put(objectPointer, 8, 8, value)
}

func (om *ObjectMemory) pointerBitOf(objectPointer int) int {
	return om.ot_bits_to(objectPointer, 9, 9)
}

func (om *ObjectMemory) pointerBitOf_put(objectPointer, value int) int {
	return om.ot_bits_to_put(objectPointer, 9, 9, value)
}

func (om *ObjectMemory) freeBitOf(objectPointer int) int {
	return om.ot_bits_to(objectPointer, 10, 10)
}

func (om *ObjectMemory) freeBitOf_put(objectPointer, value int) int {
	return om.ot_bits_to_put(objectPointer, 10, 10, value)
}

func (om *ObjectMemory) segmentBitsOf(objectPointer int) int {
	return om.ot_bits_to(objectPointer, 12, 15)
}

func (om *ObjectMemory) segmentBitsOf_put(objectPointer, value int) int {
	return om.ot_bits_to_put(objectPointer, 12, 15, value)
}

func (om *ObjectMemory) locationBitsOf(objectPointer int) int {
	om.CantBeIntegerObject(objectPointer)
	return int(om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+objectPointer+1))
}

func (om *ObjectMemory) locationBitsOf_put(objectPointer, value int) int {
	om.CantBeIntegerObject(objectPointer)
	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+objectPointer+1, uint16(value))
	return value
}

func (om *ObjectMemory) heapChunkOf_word(objectPointer, offset int) int {
	return int(om.wordMemory.SegmentWord(om.segmentBitsOf(objectPointer), om.locationBitsOf(objectPointer)+offset))
}

func (om *ObjectMemory) heapChunkOf_word_put(objectPointer, offset, value int) int {
	om.wordMemory.SegmentWordPut(om.segmentBitsOf(objectPointer), om.locationBitsOf(objectPointer)+offset, uint16(value))
	return value
}

func (om *ObjectMemory) heapChunkOf_byte(objectPointer, offset int) int {
	return int(om.wordMemory.SegmentWordByte(om.segmentBitsOf(objectPointer), om.locationBitsOf(objectPointer)+offset/2, offset%2))
}

func (om *ObjectMemory) heapChunkOf_byte_put(objectPointer, offset, value int) int {
	om.wordMemory.SegmentWordBytePut(om.segmentBitsOf(objectPointer), om.locationBitsOf(objectPointer)+offset/2, offset%2, uint8(value))
	return value
}

func (om *ObjectMemory) sizeBitsOf(objectPointer int) int {
	return om.heapChunkOf_word(objectPointer, 0)
}

func (om *ObjectMemory) sizeBitsOf_put(objectPointer, value int) int {
	return om.heapChunkOf_word_put(objectPointer, 0, value)
}

func (om *ObjectMemory) classBitsOf(objectPointer int) int {
	return om.heapChunkOf_word(objectPointer, 1)
}

func (om *ObjectMemory) classBitsOf_put(objectPointer, value int) int {
	return om.heapChunkOf_word_put(objectPointer, 1, value)
}

func (om *ObjectMemory) FetchWordLengthOf(objectPointer int) int {
	return om.sizeBitsOf(objectPointer) - HeaderSize
}

func (om *ObjectMemory) FetchByteLengthOf(objectPointer int) int {
	return om.FetchWordLengthOf(objectPointer)*2 - om.oddBitOf(objectPointer)
}

func (om *ObjectMemory) FetchWord_ofObject(wordIndex, objectPointer int) int {
	om.runtimeCheck(wordIndex >= 0 && wordIndex < om.FetchWordLengthOf(objectPointer), "fetchWord_ofObject out of range")
	return om.heapChunkOf_word(objectPointer, HeaderSize+wordIndex)
}

func (om *ObjectMemory) FetchByte_ofObject(byteIndex, objectPointer int) int {
	return om.heapChunkOf_byte(objectPointer, HeaderSize*2+byteIndex)
}

func (om *ObjectMemory) FetchPointer_ofObject(fieldIndex, objectPointer int) int {
	om.runtimeCheck(fieldIndex >= 0 && fieldIndex < om.FetchWordLengthOf(objectPointer), "fetchPointer_ofObject out of range")
	return om.heapChunkOf_word(objectPointer, HeaderSize+fieldIndex)
}

func (om *ObjectMemory) StoreWord_ofObject_withValue(wordIndex, objectPointer, valueWord int) int {
	return om.heapChunkOf_word_put(objectPointer, HeaderSize+wordIndex, valueWord)
}

func (om *ObjectMemory) StoreByte_ofObject_withValue(byteIndex, objectPointer, valueByte int) int {
	return om.heapChunkOf_byte_put(objectPointer, HeaderSize*2+byteIndex, valueByte)
}

func (om *ObjectMemory) StorePointer_ofObject_withValue(fieldIndex, objectPointer, valuePointer int) int {
	om.countUp(valuePointer)
	om.countDown(om.fetchPointer_ofObject_internal(fieldIndex, objectPointer))
	return om.heapChunkOf_word_put(objectPointer, HeaderSize+fieldIndex, valuePointer)
}

func (om *ObjectMemory) fetchPointer_ofObject_internal(fieldIndex, objectPointer int) int {
	return om.heapChunkOf_word(objectPointer, HeaderSize+fieldIndex)
}

func (om *ObjectMemory) FetchClassOf(objectPointer int) int {
	if om.IsIntegerObject(objectPointer) {
		return oops.ClassSmallInteger
	}
	cls := om.classBitsOf(objectPointer)
	if (cls == NonPointer || cls == 0) && objectPointer == oops.NilPointer {
		return oops.ClassUndefinedObject
	}
	return cls
}

func (om *ObjectMemory) HasObject(objectPointer int) bool {
	if om.IsIntegerObject(objectPointer) {
		return false
	}
	if objectPointer < 0 || objectPointer >= ObjectTableSize {
		return false
	}
	return om.freeBitOf(objectPointer) == 0 && om.countBitsOf(objectPointer) > 0
}

func (om *ObjectMemory) IncreaseReferencesTo(objectPointer int) {
	om.countUp(objectPointer)
}

func (om *ObjectMemory) DecreaseReferencesTo(objectPointer int) {
	om.countDown(objectPointer)
}

func (om *ObjectMemory) countUp(objectPointer int) int {
	if om.IsIntegerObject(objectPointer) {
		return objectPointer
	}
	count := om.countBitsOf(objectPointer)
	if count < 128 {
		om.countBitsOf_put(objectPointer, count+1)
	}
	return objectPointer
}

// countDown decrements an object's reference count but does NOT eagerly free it
// when the count reaches zero. Eager, cascading free-on-zero is extremely
// sensitive to any single missed countUp elsewhere: one under-count frees a
// still-referenced chunk, leaving a dangling pointer that later corrupts the
// mark-sweep collector. Instead we let unreachable objects accumulate and be
// reclaimed by reclaimInaccessibleObjects (the Bluebook's mark-sweep fallback),
// which recomputes all counts from reachability and is immune to count drift.
// Counts >= 128 are sticky (never decremented).
func (om *ObjectMemory) countDown(rootObjectPointer int) int {
	if om.IsIntegerObject(rootObjectPointer) {
		return rootObjectPointer
	}
	if om.countBitsOf(rootObjectPointer) == 0 {
		return rootObjectPointer
	}
	return om.forAllObjectsAccessibleFrom_suchThat_do(
		rootObjectPointer,
		func(objectPointer int) bool {
			count := om.countBitsOf(objectPointer) - 1
			if count < 127 {
				om.countBitsOf_put(objectPointer, count)
			}
			return count == 0
		},
		func(objectPointer int) {
			om.countBitsOf_put(objectPointer, 0)
			om.freeWords += uint32(om.spaceOccupiedBy(objectPointer))
			om.freeOops++
			om.deallocate(objectPointer)
		},
	)
}

func (om *ObjectMemory) deallocate(objectPointer int) {
	space := om.spaceOccupiedBy(objectPointer)
	om.sizeBitsOf_put(objectPointer, space)
	om.toFreeChunkList_add(minInt(space, BigSize), objectPointer)
}

func (om *ObjectMemory) forAllOtherObjectsAccessibleFrom_suchThat_do(
	objectPointer int,
	predicate func(int) bool,
	action func(int),
) int {
	prior := NonPointer
	current := objectPointer
	offset := om.lastPointerOf(objectPointer)
	size := offset

	for {
		offset--
		if offset > 0 {
			next := om.heapChunkOf_word(current, offset)
			if !om.IsIntegerObject(next) && predicate(next) {
				om.heapChunkOf_word_put(current, offset, prior)
				if size < HugeSize {
					om.countBitsOf_put(current, offset)
				} else {
					om.heapChunkOf_word_put(current, size, offset)
				}
				prior = current
				current = next
				offset = om.lastPointerOf(current)
				size = offset
			}
		} else {
			if prior == NonPointer {
				action(current)
				return objectPointer
			}
			next := current
			current = prior
			size = om.lastPointerOf(current)
			if size < HugeSize {
				offset = om.countBitsOf(current)
			} else {
				offset = om.heapChunkOf_word(current, size)
			}
			prior = om.heapChunkOf_word(current, offset)
			om.heapChunkOf_word_put(current, offset, next)
			action(next)
		}
	}
}

func (om *ObjectMemory) forAllObjectsAccessibleFrom_suchThat_do(
	objectPointer int,
	predicate func(int) bool,
	action func(int),
) int {
	if om.IsIntegerObject(objectPointer) {
		return objectPointer
	}
	if predicate(objectPointer) {
		return om.forAllOtherObjectsAccessibleFrom_suchThat_do(objectPointer, predicate, action)
	}
	return objectPointer
}

func (om *ObjectMemory) spaceOccupiedBy(objectPointer int) int {
	size := om.sizeBitsOf(objectPointer)
	if size >= HugeSize && om.pointerBitOf(objectPointer) == 1 {
		return size + 1
	}
	return size
}

// Free pointer / chunk list helpers

func (om *ObjectMemory) headOfFreePointerList() int {
	return int(om.wordMemory.SegmentWord(ObjectTableSegment, FreePointerList))
}

func (om *ObjectMemory) headOfFreePointerListPut(objectPointer int) int {
	om.wordMemory.SegmentWordPut(ObjectTableSegment, FreePointerList, uint16(objectPointer))
	return objectPointer
}

func (om *ObjectMemory) toFreePointerListAdd(objectPointer int) {
	om.locationBitsOf_put(objectPointer, om.headOfFreePointerList())
	om.headOfFreePointerListPut(objectPointer)
	om.freeBitOf_put(objectPointer, 1)
}

func (om *ObjectMemory) removeFromFreePointerList() int {
	if om.headOfFreePointerList() == NonPointer {
		return oops.NilPointer
	}
	objectPointer := om.headOfFreePointerList()
	om.headOfFreePointerListPut(om.locationBitsOf(objectPointer))
	om.freeBitOf_put(objectPointer, 0)
	return objectPointer
}

func (om *ObjectMemory) headOfFreeChunkList_inSegment(size, segment int) int {
	return int(om.wordMemory.SegmentWord(segment, FirstFreeChunkList+size))
}

func (om *ObjectMemory) headOfFreeChunkList_inSegment_put(size, segment, objectPointer int) int {
	om.wordMemory.SegmentWordPut(segment, FirstFreeChunkList+size, uint16(objectPointer))
	return objectPointer
}

func (om *ObjectMemory) resetFreeChunkList_inSegment(size, segment int) {
	om.headOfFreeChunkList_inSegment_put(size, segment, NonPointer)
}

func (om *ObjectMemory) toFreeChunkList_add(size, objectPointer int) {
	seg := om.segmentBitsOf(objectPointer)
	om.classBitsOf_put(objectPointer, om.headOfFreeChunkList_inSegment(size, seg))
	om.headOfFreeChunkList_inSegment_put(size, seg, objectPointer)
	om.countBitsOf_put(objectPointer, 0)
	om.freeWords += uint32(size)
}

func (om *ObjectMemory) removeFromFreeChunkList(size int) int {
	for seg := FirstHeapSegment; seg <= LastHeapSegment; seg++ {
		if om.headOfFreeChunkList_inSegment(size, seg) != NonPointer {
			objectPointer := om.headOfFreeChunkList_inSegment(size, seg)
			om.headOfFreeChunkList_inSegment_put(size, seg, om.classBitsOf(objectPointer))
			om.freeWords -= uint32(size)
			return objectPointer
		}
	}
	// Empty list: return NilPointer (not NonPointer) so callers fall through to
	// splitting a larger chunk instead of treating NonPointer as a valid chunk.
	return oops.NilPointer
}

func (om *ObjectMemory) auditFreeOops() int {
	count := 0
	for objectPointer := 2; objectPointer < ObjectTableSize; objectPointer += 2 {
		if om.freeBitOf(objectPointer) == 1 || om.countBitsOf(objectPointer) == 0 {
			count++
		}
	}
	return count
}

func (om *ObjectMemory) obtainPointer_location(size, location int) int {
	objectPointer := om.removeFromFreePointerList()
	if objectPointer == oops.NilPointer {
		return oops.NilPointer
	}
	om.segmentBitsOf_put(objectPointer, om.currentSegment)
	om.locationBitsOf_put(objectPointer, location)
	om.sizeBitsOf_put(objectPointer, size)
	om.freeOops--
	return objectPointer
}

func (om *ObjectMemory) allocate_odd_pointer_extra_class(size, oddBit, pointerBit, extraWord, classPointer int) int {
	chunkSize := size + HeaderSize + extraWord
	objectPointer := om.allocateChunk(chunkSize)
	if objectPointer == oops.NilPointer || objectPointer == NonPointer {
		return oops.NilPointer
	}
	// A freshly allocated object starts with a reference count of 0; it is
	// counted up when first referenced (stored, pushed, or made active). Setting
	// it to 1 here would double-count every object and leak it on release.
	om.countBitsOf_put(objectPointer, 0)
	om.oddBitOf_put(objectPointer, oddBit)
	om.pointerBitOf_put(objectPointer, pointerBit)
	om.classBitsOf_put(objectPointer, classPointer)
	// Initialize the fields with raw (non-reference-counted) stores. The chunk
	// may be recycled and hold stale pointers; reference-counting those would be
	// incorrect. Pointer objects default to NilPointer, word/byte objects to 0.
	defaultValue := 0
	if pointerBit != 0 {
		defaultValue = oops.NilPointer
	}
	for i := HeaderSize; i < HeaderSize+size; i++ {
		om.heapChunkOf_word_put(objectPointer, i, defaultValue)
	}
	om.sizeBitsOf_put(objectPointer, size+HeaderSize)
	if extraWord == 1 {
		om.heapChunkOf_word_put(objectPointer, size+HeaderSize, size+HeaderSize)
	}
	return objectPointer
}

func (om *ObjectMemory) InstantiateClass_withBytes(classPointer, length int) int {
	om.countUp(classPointer)
	var extraWord int
	if (length+1)/2 >= HugeSize {
		extraWord = 1
	}
	return om.allocate_odd_pointer_extra_class((length+1)/2, length%2, 0, extraWord, classPointer)
}

func (om *ObjectMemory) InstantiateClass_withWords(classPointer, length int) int {
	om.countUp(classPointer)
	var extraWord int
	if length >= HugeSize {
		extraWord = 1
	}
	return om.allocate_odd_pointer_extra_class(length, 0, 0, extraWord, classPointer)
}

func (om *ObjectMemory) InstantiateClass_withPointers(classPointer, length int) int {
	om.countUp(classPointer)
	var extraWord int
	if length >= HugeSize {
		extraWord = 1
	}
	return om.allocate_odd_pointer_extra_class(length, 0, 1, extraWord, classPointer)
}

func (om *ObjectMemory) SwapPointersOf_and(firstPointer, secondPointer int) {
	om.CantBeIntegerObject(firstPointer)
	om.CantBeIntegerObject(secondPointer)

	firstOT := om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+firstPointer)
	firstLoc := om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+firstPointer+1)

	secondOT := om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+secondPointer)
	secondLoc := om.wordMemory.SegmentWord(ObjectTableSegment, ObjectTableStart+secondPointer+1)

	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+firstPointer, secondOT)
	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+firstPointer+1, secondLoc)

	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+secondPointer, firstOT)
	om.wordMemory.SegmentWordPut(ObjectTableSegment, ObjectTableStart+secondPointer+1, firstLoc)
}

func (om *ObjectMemory) InitialInstanceOf(classPointer int) int {
	for objectPointer := 2; objectPointer < ObjectTableSize; objectPointer += 2 {
		if om.HasObject(objectPointer) && om.FetchClassOf(objectPointer) == classPointer {
			return objectPointer
		}
	}
	return oops.NilPointer
}

func (om *ObjectMemory) InstanceAfter(objectPointer int) int {
	targetClass := om.FetchClassOf(objectPointer)
	for next := objectPointer + 2; next < ObjectTableSize; next += 2 {
		if om.HasObject(next) && om.FetchClassOf(next) == targetClass {
			return next
		}
	}
	return oops.NilPointer
}

// Chunk Allocation & Garbage Collection

func (om *ObjectMemory) allocateChunk(size int) int {
	objectPointer := om.attemptToAllocateChunk(size)
	if objectPointer == oops.NilPointer {
		om.reclaimInaccessibleObjects()
		objectPointer = om.attemptToAllocateChunk(size)
	}
	if objectPointer != oops.NilPointer {
		if om.freeWords >= uint32(size) {
			om.freeWords -= uint32(size)
		}
		return objectPointer
	}
	om.outOfMemoryError(size)
	return oops.NilPointer
}

func (om *ObjectMemory) attemptToAllocateChunk(size int) int {
	objectPointer := om.attemptToAllocateChunkInCurrentSegment(size)
	if objectPointer != oops.NilPointer {
		return objectPointer
	}
	for i := 1; i <= HeapSegmentCount; i++ {
		om.currentSegment++
		if om.currentSegment > LastHeapSegment {
			om.currentSegment = FirstHeapSegment
		}
		om.compactCurrentSegment()
		objectPointer = om.attemptToAllocateChunkInCurrentSegment(size)
		if objectPointer != oops.NilPointer {
			return objectPointer
		}
	}
	return oops.NilPointer
}

func (om *ObjectMemory) attemptToAllocateChunkInCurrentSegment(size int) int {
	var objectPointer int = oops.NilPointer
	if size < BigSize {
		objectPointer = om.removeFromFreeChunkList(size)
	}
	if objectPointer != oops.NilPointer {
		return objectPointer
	}

	predecessor := NonPointer
	objectPointer = om.headOfFreeChunkList_inSegment(BigSize, om.currentSegment)

	for objectPointer != NonPointer {
		availableSize := om.sizeBitsOf(objectPointer)
		if availableSize == size {
			next := om.classBitsOf(objectPointer)
			if predecessor == NonPointer {
				om.headOfFreeChunkList_inSegment_put(BigSize, om.currentSegment, next)
			} else {
				om.classBitsOf_put(predecessor, next)
			}
			return objectPointer
		}

		excessSize := availableSize - size
		if excessSize >= HeaderSize {
			newPointer := om.obtainPointer_location(size, om.locationBitsOf(objectPointer)+excessSize)
			if newPointer == oops.NilPointer {
				return oops.NilPointer
			}
			om.sizeBitsOf_put(objectPointer, excessSize)
			return newPointer
		} else {
			predecessor = objectPointer
			objectPointer = om.classBitsOf(objectPointer)
		}
	}

	return oops.NilPointer
}

func (om *ObjectMemory) GarbageCollect() {
	om.reclaimInaccessibleObjects()
}

func (om *ObjectMemory) reclaimInaccessibleObjects() {
	om.zeroReferenceCounts()
	om.markAccessibleObjects()
	om.rectifyCountsAndDeallocateGarbage()
	for segment := FirstHeapSegment; segment <= LastHeapSegment; segment++ {
		om.currentSegment = segment
		om.compactCurrentSegment()
	}
	om.countBitsOf_put(oops.NilPointer, 128)
	om.freeOops = om.auditFreeOops()
	if om.gcNotification != nil {
		om.gcNotification.CollectionCompleted()
	}
}

func (om *ObjectMemory) AddRoot(rootObjectPointer int) {
	om.markObjectsAccessibleFrom(rootObjectPointer)
}

func (om *ObjectMemory) markAccessibleObjects() {
	for i := 0; i <= LastSpecialOop; i += 2 {
		om.AddRoot(i)
	}
	if om.gcNotification != nil {
		om.gcNotification.PrepareForCollection()
	}
}

func (om *ObjectMemory) markObjectsAccessibleFrom(rootObjectPointer int) int {
	if om.IsIntegerObject(rootObjectPointer) {
		return rootObjectPointer
	}
	return om.forAllObjectsAccessibleFrom_suchThat_do(rootObjectPointer,
		func(objectPointer int) bool {
			unmarked := om.countBitsOf(objectPointer) == 0
			if unmarked {
				om.countBitsOf_put(objectPointer, 1)
			}
			return unmarked
		},
		func(objectPointer int) {
			om.countBitsOf_put(objectPointer, 1)
		},
	)
}

func (om *ObjectMemory) zeroReferenceCounts() {
	for objectPointer := 2; objectPointer <= ObjectTableSize-2; objectPointer += 2 {
		om.countBitsOf_put(objectPointer, 0)
	}
}

func (om *ObjectMemory) rectifyCountsAndDeallocateGarbage() {
	for segment := FirstHeapSegment; segment <= LastHeapSegment; segment++ {
		for size := HeaderSize; size <= BigSize; size++ {
			om.resetFreeChunkList_inSegment(size, segment)
		}
	}

	for objectPointer := 2; objectPointer <= ObjectTableSize-2; objectPointer += 2 {
		if om.freeBitOf(objectPointer) == 0 {
			count := om.countBitsOf(objectPointer)
			if count == 0 {
				om.freeWords += uint32(om.spaceOccupiedBy(objectPointer))
				om.deallocate(objectPointer)
			} else {
				if count < 128 {
					om.countBitsOf_put(objectPointer, count-1)
				}
				limit := om.lastPointerOf(objectPointer) - 1
				for offset := 1; offset <= limit; offset++ {
					om.countUp(om.heapChunkOf_word(objectPointer, offset))
				}
			}
		}
	}

	om.countBitsOf_put(oops.NilPointer, 128)
	om.freeOops = om.auditFreeOops()
	if om.gcNotification != nil {
		om.gcNotification.CollectionCompleted()
	}
}

func (om *ObjectMemory) lastPointerOf(objectPointer int) int {
	if om.pointerBitOf(objectPointer) == 0 {
		if om.classBitsOf(objectPointer) == oops.ClassCompiledMethod {
			methodHeader := om.heapChunkOf_word(objectPointer, HeaderSize)
			return HeaderSize + 1 + ((methodHeader & 126) >> 1)
		}
		return HeaderSize
	}
	return om.sizeBitsOf(objectPointer)
}

func (om *ObjectMemory) compactCurrentSegment() {
	lowWaterMark := om.abandonFreeChunksInSegment(om.currentSegment)
	if lowWaterMark < HeapSpaceStop {
		om.reverseHeapPointersAbove(lowWaterMark)
		bigSpace := om.sweepCurrentSegmentFrom(lowWaterMark)
		newPtr := om.obtainPointer_location(HeapSpaceStop+1-bigSpace, bigSpace)
		if newPtr != oops.NilPointer {
			om.deallocate(newPtr)
		}
	}
}

func (om *ObjectMemory) abandonFreeChunksInSegment(segment int) int {
	lowWaterMark := HeapSpaceStop
	for size := HeaderSize; size <= BigSize; size++ {
		objectPointer := om.headOfFreeChunkList_inSegment(size, segment)
		for objectPointer != NonPointer {
			nextPointer := om.classBitsOf(objectPointer)
			if om.freeBitOf(objectPointer) == 0 {
				lowWaterMark = minInt(lowWaterMark, om.locationBitsOf(objectPointer))
				om.classBitsOf_put(objectPointer, NonPointer)
				om.releasePointer(objectPointer)
			}
			objectPointer = nextPointer
		}
		om.resetFreeChunkList_inSegment(size, segment)
	}
	return lowWaterMark
}

func (om *ObjectMemory) releasePointer(objectPointer int) {
	om.freeBitOf_put(objectPointer, 1)
	om.toFreePointerListAdd(objectPointer)
}

func (om *ObjectMemory) reverseHeapPointersAbove(lowWaterMark int) {
	for objectPointer := 2; objectPointer <= ObjectTableSize-2; objectPointer += 2 {
		if om.freeBitOf(objectPointer) == 0 {
			if om.segmentBitsOf(objectPointer) == om.currentSegment {
				if om.locationBitsOf(objectPointer) >= lowWaterMark {
					size := om.sizeBitsOf(objectPointer)
					om.sizeBitsOf_put(objectPointer, objectPointer)
					om.locationBitsOf_put(objectPointer, size)
				}
			}
		}
	}
}

func (om *ObjectMemory) sweepCurrentSegmentFrom(lowWaterMark int) int {
	si := lowWaterMark
	di := lowWaterMark
	for si < HeapSpaceStop {
		if om.wordMemory.SegmentWord(om.currentSegment, si+1) == uint16(NonPointer) {
			size := int(om.wordMemory.SegmentWord(om.currentSegment, si))
			si += size
		} else {
			objectPointer := int(om.wordMemory.SegmentWord(om.currentSegment, si))
			size := om.locationBitsOf(objectPointer)
			om.locationBitsOf_put(objectPointer, di)
			om.sizeBitsOf_put(objectPointer, size)
			si++
			di++
			limit := om.spaceOccupiedBy(objectPointer)
			for i := 2; i <= limit; i++ {
				if si < realwordmemory.SegmentSize && di < realwordmemory.SegmentSize {
					om.wordMemory.SegmentWordPut(om.currentSegment, di, om.wordMemory.SegmentWord(om.currentSegment, si))
				}
				si++
				di++
			}
		}
	}
	return di
}

func (om *ObjectMemory) outOfMemoryError(size int) {
	fmt.Printf("OUT OF MEMORY: requested size=%d, freeOops=%d, freeWords=%d, currentSegment=%d\n", size, om.freeOops, om.freeWords, om.currentSegment)
	if om.hal != nil {
		om.hal.Error("Out of memory")
	} else {
		panic("Out of memory")
	}
}

// Snapshot Load & Save

func (om *ObjectMemory) LoadSnapshot(fileSystem filesystem.FileSystem, fileName string) bool {
	fd := fileSystem.OpenFile(fileName)
	if fd == -1 {
		return false
	}
	defer fileSystem.CloseFile(fd)

	if !om.loadObjectTable(fileSystem, fd) {
		return false
	}
	return om.loadObjects(fileSystem, fd)
}

func (om *ObjectMemory) SaveSnapshot(fileSystem filesystem.FileSystem, fileName string) bool {
	fd := fileSystem.CreateFile(fileName)
	if fd == -1 {
		return false
	}
	defer fileSystem.CloseFile(fd)

	return om.saveObjects(fileSystem, fd)
}

func (om *ObjectMemory) loadObjectTable(fileSystem filesystem.FileSystem, fd int) bool {
	var objectTableLength int32
	if fileSystem.SeekTo(fd, 4) == -1 {
		return false
	}
	buf := make([]byte, 4)
	if fileSystem.Read(fd, buf) != 4 {
		return false
	}
	objectTableLength = int32(binary.LittleEndian.Uint32(buf))
	fileSize := fileSystem.FileSize(fd)

	if fileSystem.SeekTo(fd, fileSize-int(objectTableLength)*2) == -1 {
		return false
	}

	wbuf := make([]byte, 4)
	for objectPointer := 0; objectPointer < int(objectTableLength); objectPointer += 2 {
		if fileSystem.Read(fd, wbuf) != 4 {
			return false
		}
		w0 := int(binary.LittleEndian.Uint16(wbuf[0:2]))
		w1 := int(binary.LittleEndian.Uint16(wbuf[2:4]))
		om.ot_put(objectPointer, w0)
		om.locationBitsOf_put(objectPointer, w1)
	}

	om.headOfFreePointerListPut(NonPointer)
	for objectPointer := int(objectTableLength); objectPointer < ObjectTableSize; objectPointer += 2 {
		om.ot_put(objectPointer, 0)
		om.freeBitOf_put(objectPointer, 1)
		om.locationBitsOf_put(objectPointer, 0)
	}

	for objectPointer := ObjectTableSize - 2; objectPointer >= 2; objectPointer -= 2 {
		if om.freeBitOf(objectPointer) == 1 {
			om.toFreePointerListAdd(objectPointer)
		}
	}
	om.freeOops = om.auditFreeOops()
	return true
}

func (om *ObjectMemory) loadObjects(fileSystem filesystem.FileSystem, fd int) bool {
	segmentHeapSpaceSize := HeapSpaceStop + 1
	heapSpaceRemaining := make([]int, HeapSegmentCount)
	for seg := FirstHeapSegment; seg <= LastHeapSegment; seg++ {
		heapSpaceRemaining[seg-FirstHeapSegment] = segmentHeapSpaceSize
	}

	destinationSegment := FirstHeapSegment
	destinationWord := 0

	wbuf := make([]byte, 2)
	for objectPointer := 2; objectPointer < ObjectTableSize; objectPointer += 2 {
		if om.freeBitOf(objectPointer) == 1 {
			continue
		}
		objectImageWordAddress := (om.segmentBitsOf(objectPointer) << 16) + om.locationBitsOf(objectPointer)
		fileSystem.SeekTo(fd, ObjectSpaceBaseInImage+objectImageWordAddress*2)

		if fileSystem.Read(fd, wbuf) != 2 {
			return false
		}
		objectSize := int(binary.LittleEndian.Uint16(wbuf))

		var extraSpace int
		if objectSize >= HugeSize && om.pointerBitOf(objectPointer) == 1 {
			extraSpace = 1
		}
		space := objectSize + extraSpace

		if space > heapSpaceRemaining[destinationSegment-FirstHeapSegment] {
			destinationSegment++
			if destinationSegment == HeapSegmentCount {
				return false
			}
			destinationWord = 0
		}

		om.segmentBitsOf_put(objectPointer, destinationSegment)
		om.locationBitsOf_put(objectPointer, destinationWord)

		om.sizeBitsOf_put(objectPointer, objectSize)

		if fileSystem.Read(fd, wbuf) != 2 {
			return false
		}
		classBits := int(binary.LittleEndian.Uint16(wbuf))
		om.classBitsOf_put(objectPointer, classBits)

		for wordIndex := 0; wordIndex < objectSize-HeaderSize; wordIndex++ {
			if fileSystem.Read(fd, wbuf) != 2 {
				return false
			}
			word := int(binary.LittleEndian.Uint16(wbuf))
			om.StoreWord_ofObject_withValue(wordIndex, objectPointer, word)
		}

		destinationWord += space
		heapSpaceRemaining[destinationSegment-FirstHeapSegment] -= space
	}

	for segment := FirstHeapSegment; segment <= LastHeapSegment; segment++ {
		for size := HeaderSize; size <= BigSize; size++ {
			om.resetFreeChunkList_inSegment(size, segment)
		}
	}

	om.freeWords = 0
	for segment := FirstHeapSegment; segment <= LastHeapSegment; segment++ {
		freeChunkSize := heapSpaceRemaining[segment-FirstHeapSegment]
		om.freeWords += uint32(freeChunkSize)
		if freeChunkSize >= HeaderSize {
			freeChunkLocation := segmentHeapSpaceSize - freeChunkSize
			om.currentSegment = segment
			objectPointer := om.obtainPointer_location(freeChunkSize, freeChunkLocation)
			om.toFreeChunkList_add(minInt(freeChunkSize, BigSize), objectPointer)
		}
	}

	om.currentSegment = FirstHeapSegment
	return true
}

func (om *ObjectMemory) saveObjects(fileSystem filesystem.FileSystem, fd int) bool {
	lastUsedObjectPointer := NonPointer
	for objectPointer := 2; objectPointer < ObjectTableSize; objectPointer += 2 {
		if om.HasObject(objectPointer) {
			lastUsedObjectPointer = objectPointer
		}
	}

	storedObjectTableLength := lastUsedObjectPointer + 2
	hdrBuf := make([]byte, 8)
	fileSystem.Write(fd, hdrBuf) // Placeholder

	fileSystem.SeekTo(fd, ObjectSpaceBaseInImage)
	wbuf := make([]byte, 2)

	for objectPointer := 2; objectPointer < storedObjectTableLength; objectPointer += 2 {
		if !om.HasObject(objectPointer) {
			continue
		}
		size := om.sizeBitsOf(objectPointer)
		binary.LittleEndian.PutUint16(wbuf, uint16(size))
		fileSystem.Write(fd, wbuf)

		cls := om.classBitsOf(objectPointer)
		binary.LittleEndian.PutUint16(wbuf, uint16(cls))
		fileSystem.Write(fd, wbuf)

		for wordIndex := 0; wordIndex < size-HeaderSize; wordIndex++ {
			word := om.heapChunkOf_word(objectPointer, HeaderSize+wordIndex)
			binary.LittleEndian.PutUint16(wbuf, uint16(word))
			fileSystem.Write(fd, wbuf)
		}
	}

	om.padToPage(fileSystem, fd)
	endOfObjects := fileSystem.Tell(fd)
	objectSpaceLength := (endOfObjects - ObjectSpaceBaseInImage) / 2

	for objectPointer := 0; objectPointer < storedObjectTableLength; objectPointer += 2 {
		otVal := om.ot(objectPointer)
		locVal := om.locationBitsOf(objectPointer)
		binary.LittleEndian.PutUint16(wbuf, uint16(otVal))
		fileSystem.Write(fd, wbuf)
		binary.LittleEndian.PutUint16(wbuf, uint16(locVal))
		fileSystem.Write(fd, wbuf)
	}

	fileSystem.SeekTo(fd, 0)
	binary.LittleEndian.PutUint32(hdrBuf[0:4], uint32(objectSpaceLength))
	binary.LittleEndian.PutUint32(hdrBuf[4:8], uint32(storedObjectTableLength))
	fileSystem.Write(fd, hdrBuf)

	return true
}

func (om *ObjectMemory) padToPage(fileSystem filesystem.FileSystem, fd int) bool {
	pos := fileSystem.Tell(fd)
	desired := ((pos + 512 - 1) / 512) * 512
	wbuf := make([]byte, 2)
	pad := (desired - pos) / 2
	for pad > 0 {
		if fileSystem.Write(fd, wbuf) != 2 {
			return false
		}
		pad--
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
