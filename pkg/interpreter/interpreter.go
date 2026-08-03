package interpreter

import (
	"fmt"
	"os"

	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/hal"
	"smalltalk80/pkg/objmemory"
	"smalltalk80/pkg/oops"
)

var dbgOn = os.Getenv("STDBG") != ""

// Constants for Context / Process / Method offsets (G&R)
const (
	// MethodContext
	SenderIndex             = 0
	InstructionPointerIndex = 1
	StackPointerIndex       = 2
	MethodIndex             = 3
	ReceiverIndex           = 5
	TempFrameStart          = 6

	// BlockContext
	BlockCallerIndex        = 0
	BlockIPIndex            = 1
	BlockSPIndex            = 2
	BlockArgumentCountIndex = 3
	BlockInitialIPIndex     = 4
	BlockHomeIndex          = 5

	// Process / Scheduler
	NextLinkIndex         = 0
	SuspendedContextIndex = 1
	PriorityIndex         = 2
	MyListIndex           = 3
	FirstLinkIndex        = 0
	LastLinkIndex         = 1
	ProcessListsIndex     = 0
	ActiveProcessIndex    = 1
	ValueIndex            = 1

	// Method
	HeaderIndex                = 0
	LiteralFrameStart          = 1
	SuperclassIndex            = 0
	MessageDictionaryIndex     = 1
	InstanceSpecificationIndex = 2

	// MethodDictionary
	MethodArrayIndex = 1
	SelectorStart    = 2

	// Array / Stream / Class
	StreamArrayIndex      = 0
	StreamIndexIndex      = 1
	StreamReadLimitIndex  = 2
	StreamWriteLimitIndex = 3
	MessageSelectorIndex  = 0
	MessageArgumentsIndex = 1
	MessageSize           = 2

	MethodCacheSize = 1024 // Power of 2
	NonPointer      = 65535
)

type MethodCacheEntry struct {
	selector int
	class    int
	method   int
	prim     int
}

type Interpreter struct {
	hal        hal.HAL
	fileSystem filesystem.FileSystem
	memory     *objmemory.ObjectMemory

	activeContext      int
	homeContext        int
	method             int
	receiver           int
	instructionPointer int
	stackPointer       int
	currentBytecode    int
	messageSelector    int
	argumentCount      int
	successFlag        bool

	methodCache [MethodCacheSize]MethodCacheEntry

	newProcess        int
	newProcessWaiting bool

	checkLowMemory    bool
	memoryIsLow       bool
	lowSpaceSemaphore int
	oopsLeftLimit     int
	wordsLeftLimit    int

	currentDisplay       int
	currentCursor        int
	currentDisplayWidth  int
	currentDisplayHeight int

	semaphoreIndex int
	semaphoreList  [64]int
	totalCycles    int64

}

func New(halInterface hal.HAL, fileSystemInterface filesystem.FileSystem) *Interpreter {
	interp := &Interpreter{
		hal:        halInterface,
		fileSystem: fileSystemInterface,
	}
	interp.memory = objmemory.New(halInterface, interp)
	return interp
}

func (interp *Interpreter) Memory() *objmemory.ObjectMemory {
	return interp.memory
}

func (interp *Interpreter) Init() bool {
	interp.initializeMethodCache()
	interp.semaphoreIndex = -1

	imageName := "snapshot.im"
	if interp.hal != nil && interp.hal.GetImageName() != "" {
		imageName = interp.hal.GetImageName()
	}

	if !interp.memory.LoadSnapshot(interp.fileSystem, imageName) {
		return false
	}
	if dbgOn {
		objmemory.DbgWatch = 16776
	}

	interp.activeContext = interp.firstContext()
	interp.memory.IncreaseReferencesTo(interp.activeContext)
	interp.fetchContextRegisters()

	interp.checkLowMemory = false
	interp.memoryIsLow = false
	interp.lowSpaceSemaphore = oops.NilPointer
	interp.oopsLeftLimit = 0
	interp.wordsLeftLimit = 0
	interp.currentDisplay = 0
	interp.currentCursor = 0
	interp.currentDisplayWidth = 0
	interp.currentDisplayHeight = 0
	return true
}

func (interp *Interpreter) CheckLowMemoryConditions() {
	interp.checkLowMemory = true
}

func (interp *Interpreter) PrepareForCollection() {
	interp.storeContextRegisters()
	interp.memory.AddRoot(oops.SmalltalkPointer)
	interp.memory.AddRoot(interp.activeContext)
	if interp.newProcess != oops.NilPointer {
		interp.memory.AddRoot(interp.newProcess)
	}
}

func (interp *Interpreter) CollectionCompleted() {
	interp.memory.IncreaseReferencesTo(interp.activeContext)
	interp.fetchContextRegisters()
	if interp.newProcessWaiting {
		interp.memory.IncreaseReferencesTo(interp.newProcess)
	}
}

func (interp *Interpreter) error(msg string) {
	if interp.hal != nil {
		interp.hal.Error(msg)
	} else {
		panic(msg)
	}
}

func (interp *Interpreter) initializeMethodCache() {
	for i := 0; i < MethodCacheSize; i++ {
		interp.methodCache[i].selector = oops.NilPointer
		interp.methodCache[i].class = oops.NilPointer
		interp.methodCache[i].method = oops.NilPointer
		interp.methodCache[i].prim = 0
	}
}

func (interp *Interpreter) GetDisplayBits(width, height int) int {
	displayBits := interp.fetchDisplayBits()
	if displayBits != 0 {
		interp.currentDisplayWidth = width
		interp.currentDisplayHeight = height
	}
	return displayBits
}

func (interp *Interpreter) FetchWord_ofDislayBits(wordIndex, displayBits int) uint16 {
	return uint16(interp.memory.FetchWord_ofObject(wordIndex, displayBits))
}

// fetchDisplayBits returns the backing Bitmap (field 0) of the current display
// Form, or 0 if no display has been registered (via primitiveBeDisplay) yet, or
// the backing bitmap is smaller than the reported extent (not yet ready).
func (interp *Interpreter) fetchDisplayBits() int {
	if interp.currentDisplay == 0 {
		return 0
	}
	displayBits := interp.memory.FetchPointer_ofObject(0, interp.currentDisplay)
	w := interp.fetchInteger_ofObject(1, interp.currentDisplay)
	h := interp.fetchInteger_ofObject(2, interp.currentDisplay)
	if w <= 0 || h <= 0 {
		return 0
	}
	computedSize := h * ((w + 15) / 16)
	if interp.memory.FetchWordLengthOf(displayBits) < computedSize {
		return 0
	}
	return displayBits
}

func (interp *Interpreter) GetDisplayExtent() (int, int) {
	if interp.currentDisplay == 0 {
		return 0, 0
	}
	bits := interp.fetchDisplayBits()
	if bits == 0 {
		return 0, 0
	}
	w := interp.fetchInteger_ofObject(1, interp.currentDisplay)
	h := interp.fetchInteger_ofObject(2, interp.currentDisplay)
	return w, h
}

func (interp *Interpreter) hash(objectPointer int) int {
	return objectPointer >> 1
}

func (interp *Interpreter) argumentCountOfBlock(blockPointer int) int {
	return interp.fetchInteger_ofObject(BlockArgumentCountIndex, blockPointer)
}

// Stack & Context Operations

func (interp *Interpreter) sender() int {
	return interp.memory.FetchPointer_ofObject(SenderIndex, interp.homeContext)
}

func (interp *Interpreter) caller() int {
	return interp.memory.FetchPointer_ofObject(SenderIndex, interp.activeContext)
}

func (interp *Interpreter) push(object int) {
	interp.stackPointer++
	interp.memory.StorePointer_ofObject_withValue(interp.stackPointer, interp.activeContext, object)
}

func (interp *Interpreter) pop(number int) {
	interp.stackPointer -= number
}

func (interp *Interpreter) unPop(number int) {
	interp.stackPointer += number
}

// headerOf returns the CompiledMethod header as the raw SmallInteger oop.
// The Bluebook header bit fields are defined on this raw value (extractBits),
// so IntegerValueOf must NOT be applied before extracting fields.
func (interp *Interpreter) headerOf(methodPointer int) int {
	return interp.memory.FetchPointer_ofObject(HeaderIndex, methodPointer)
}

func (interp *Interpreter) literalCountOf(methodPointer int) int {
	// extractBits 9..14 of the header: (header >> 1) & 0x3f
	return (interp.headerOf(methodPointer) >> 1) & 0x3f
}

func (interp *Interpreter) literal_ofMethod(offset, methodPointer int) int {
	return interp.memory.FetchPointer_ofObject(LiteralFrameStart+offset, methodPointer)
}

func (interp *Interpreter) temporaryCountOf(methodPointer int) int {
	// extractBits 3..7 of the header: (header >> 8) & 0x1f
	return (interp.headerOf(methodPointer) >> 8) & 0x1f
}

func (interp *Interpreter) popStack() int {
	top := interp.memory.FetchPointer_ofObject(interp.stackPointer, interp.activeContext)
	interp.stackPointer--
	return top
}

func (interp *Interpreter) stackTop() int {
	return interp.memory.FetchPointer_ofObject(interp.stackPointer, interp.activeContext)
}

func (interp *Interpreter) stackValue(offset int) int {
	return interp.memory.FetchPointer_ofObject(interp.stackPointer-offset, interp.activeContext)
}

func (interp *Interpreter) fetchInteger_ofObject(fieldIndex, objectPointer int) int {
	intObj := interp.memory.FetchPointer_ofObject(fieldIndex, objectPointer)
	return interp.memory.IntegerValueOf(intObj)
}

func (interp *Interpreter) storeInteger_ofObject_withValue(fieldIndex, objectPointer, value int) int {
	intObj := interp.memory.IntegerObjectOf(value)
	interp.memory.StorePointer_ofObject_withValue(fieldIndex, objectPointer, intObj)
	return intObj
}

// isBlockContext distinguishes a BlockContext from a MethodContext by testing
// whether field 3 holds a SmallInteger (the block argument count) rather than a
// CompiledMethod pointer (per Bluebook). This is independent of the context's
// class, which is how MethodContext and BlockContext share representation.
func (interp *Interpreter) isBlockContext(contextPointer int) bool {
	methodOrArguments := interp.memory.FetchPointer_ofObject(MethodIndex, contextPointer)
	return interp.memory.IsIntegerObject(methodOrArguments)
}

func (interp *Interpreter) instructionPointerOfContext(contextPointer int) int {
	return interp.fetchInteger_ofObject(InstructionPointerIndex, contextPointer)
}

func (interp *Interpreter) storeInstructionPointerValue_inContext(value, contextPointer int) {
	interp.storeInteger_ofObject_withValue(InstructionPointerIndex, contextPointer, value)
}

func (interp *Interpreter) stackPointerOfContext(contextPointer int) int {
	return interp.fetchInteger_ofObject(StackPointerIndex, contextPointer)
}

func (interp *Interpreter) storeStackPointerValue_inContext(value, contextPointer int) {
	interp.storeInteger_ofObject_withValue(StackPointerIndex, contextPointer, value)
}

func (interp *Interpreter) storeContextRegisters() {
	interp.storeInstructionPointerValue_inContext(interp.instructionPointer, interp.activeContext)
	interp.storeStackPointerValue_inContext(interp.stackPointer-TempFrameStart+1, interp.activeContext)
}

func (interp *Interpreter) fetchContextRegisters() {
	if interp.isBlockContext(interp.activeContext) {
		interp.homeContext = interp.memory.FetchPointer_ofObject(BlockHomeIndex, interp.activeContext)
	} else {
		interp.homeContext = interp.activeContext
	}
	if interp.homeContext == oops.NilPointer || interp.memory.FetchWordLengthOf(interp.homeContext) <= MethodIndex {
		interp.homeContext = interp.activeContext
	}
	interp.receiver = interp.memory.FetchPointer_ofObject(ReceiverIndex, interp.homeContext)
	interp.method = interp.memory.FetchPointer_ofObject(MethodIndex, interp.homeContext)
	interp.instructionPointer = interp.instructionPointerOfContext(interp.activeContext)
	interp.stackPointer = interp.stackPointerOfContext(interp.activeContext) + TempFrameStart - 1
}

func (interp *Interpreter) newActiveContext(aContext int) {
	interp.storeContextRegisters()
	interp.memory.DecreaseReferencesTo(interp.activeContext)
	interp.activeContext = aContext
	interp.memory.IncreaseReferencesTo(interp.activeContext)
	interp.fetchContextRegisters()
}

// Class specification helpers

func (interp *Interpreter) instanceSpecificationOf(classPointer int) int {
	return interp.memory.FetchPointer_ofObject(InstanceSpecificationIndex, classPointer)
}

func (interp *Interpreter) extractBits_to_of(firstBit, lastBit, value int) int {
	shift := uint16(value) >> uint(15-lastBit)
	mask := uint16((1 << (lastBit - firstBit + 1)) - 1)
	return int(shift & mask)
}

func (interp *Interpreter) fixedFieldsOf(classPointer int) int {
	return interp.extractBits_to_of(4, 14, interp.instanceSpecificationOf(classPointer))
}

func (interp *Interpreter) isPointers(classPointer int) bool {
	return interp.extractBits_to_of(0, 0, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) isWords(classPointer int) bool {
	return interp.extractBits_to_of(1, 1, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) isIndexable(classPointer int) bool {
	return interp.extractBits_to_of(2, 2, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) checkIndexableBoundsOf_in(index, array int) {
	interp.success(index >= 1)
	interp.success(index <= interp.lengthOf(array))
}

// Execution Loop & Bytecode Dispatch

func (interp *Interpreter) Cycle() {
	if interp.newProcessWaiting {
		interp.newProcessWaiting = false
		interp.resume(interp.newProcess)
		interp.memory.DecreaseReferencesTo(interp.newProcess)
		interp.newProcess = oops.NilPointer
	}
	interp.currentBytecode = interp.fetchByte()
	interp.totalCycles++
	if dbgOn && interp.totalCycles%5000 == 0 {
		d := 0
		for c := interp.activeContext; interp.memory.IsIntegerObject(c) == false && c != oops.NilPointer && d < 100000; d++ {
			c = interp.memory.FetchPointer_ofObject(SenderIndex, c)
		}
		fmt.Printf("cyc=%d depth=%d\n", interp.totalCycles, d)
	}
	interp.dispatchBytecode()
}

func (interp *Interpreter) fetchByte() int {
	if interp.memory.IsIntegerObject(interp.method) || interp.method == oops.NilPointer {
		return 0
	}
	byteVal := interp.memory.FetchByte_ofObject(interp.instructionPointer-1, interp.method)
	interp.instructionPointer++
	return byteVal
}

// dispatchSpecialSelector handles bytecodes 176..207. index is bytecode-176.
// The SpecialSelectors array stores (selector, argumentCount) pairs, so the
// selector is at index*2 and its argument count at index*2+1.
func (interp *Interpreter) dispatchSpecialSelector(index int) {
	selectorIndex := index * 2
	selector := interp.memory.FetchPointer_ofObject(selectorIndex, oops.SpecialSelectorsPointer)
	count := interp.fetchInteger_ofObject(selectorIndex+1, oops.SpecialSelectorsPointer)
	interp.sendSelector_argumentCount(selector, count)
}

func (interp *Interpreter) dispatchBytecode() {
	b := interp.currentBytecode
	switch {
	case b >= 0 && b <= 15:
		interp.push(interp.memory.FetchPointer_ofObject(b, interp.receiver))
	case b >= 16 && b <= 31:
		interp.push(interp.temporary(b - 16))
	case b >= 32 && b <= 63:
		interp.push(interp.literal(b - 32))
	case b >= 64 && b <= 95:
		interp.push(interp.memory.FetchPointer_ofObject(ValueIndex, interp.literal(b-64)))
	case b >= 96 && b <= 103:
		interp.memory.StorePointer_ofObject_withValue(b-96, interp.receiver, interp.stackTop())
		interp.pop(1)
	case b >= 104 && b <= 111:
		interp.storeTemporary_withValue(b-104, interp.stackTop())
		interp.pop(1)
	case b == 112:
		interp.push(interp.receiver)
	case b == 113:
		interp.push(oops.TruePointer)
	case b == 114:
		interp.push(oops.FalsePointer)
	case b == 115:
		interp.push(oops.NilPointer)
	case b == 116:
		interp.push(interp.memory.IntegerObjectOf(-1))
	case b == 117:
		interp.push(interp.memory.IntegerObjectOf(0))
	case b == 118:
		interp.push(interp.memory.IntegerObjectOf(1))
	case b == 119:
		interp.push(interp.memory.IntegerObjectOf(2))
	case b == 120:
		interp.returnValue_to(interp.receiver, interp.sender())
	case b == 121:
		interp.returnValue_to(oops.TruePointer, interp.sender())
	case b == 122:
		interp.returnValue_to(oops.FalsePointer, interp.sender())
	case b == 123:
		interp.returnValue_to(oops.NilPointer, interp.sender())
	case b == 124:
		interp.returnValue_to(interp.popStack(), interp.sender())
	case b == 125:
		interp.returnValue_to(interp.popStack(), interp.caller())
	case b == 128:
		interp.extendedPushBytecode()
	case b == 129:
		interp.extendedStoreBytecode()
	case b == 130:
		interp.extendedStoreAndPopBytecode()
	case b == 131:
		interp.singleExtendedSendBytecode()
	case b == 132:
		interp.doubleExtendedSendBytecode()
	case b == 133:
		interp.singleExtendedSuperBytecode()
	case b == 134:
		interp.doubleExtendedSuperBytecode()
	case b == 135:
		interp.popStack()
	case b == 136:
		top := interp.stackTop()
		interp.push(top)
	case b == 137:
		interp.push(interp.activeContext)
	case b >= 144 && b <= 151:
		offset := (b - 144) + 1
		interp.jump(offset)
	case b >= 152 && b <= 159:
		offset := (b - 152) + 1
		interp.jumpIf_by(oops.FalsePointer, offset)
	case b >= 160 && b <= 167:
		byte2 := interp.fetchByte()
		offset := ((b - 160) - 4)*256 + byte2
		interp.jump(offset)
	case b >= 168 && b <= 171:
		byte2 := interp.fetchByte()
		offset := (b-168)*256 + byte2
		interp.jumpIf_by(oops.TruePointer, offset)
	case b >= 172 && b <= 175:
		byte2 := interp.fetchByte()
		offset := (b-172)*256 + byte2
		interp.jumpIf_by(oops.FalsePointer, offset)
	case b >= 176 && b <= 207:
		interp.dispatchSpecialSelector(b - 176)
	case b >= 208 && b <= 223:
		interp.sendSelector_argumentCount(interp.literal(b-208), 0)
	case b >= 224 && b <= 239:
		interp.sendSelector_argumentCount(interp.literal(b-224), 1)
	case b >= 240 && b <= 255:
		interp.sendSelector_argumentCount(interp.literal(b-240), 2)
	}
}

func (interp *Interpreter) temporary(offset int) int {
	return interp.memory.FetchPointer_ofObject(offset+TempFrameStart, interp.homeContext)
}

func (interp *Interpreter) storeTemporary_withValue(offset, value int) {
	interp.memory.StorePointer_ofObject_withValue(offset+TempFrameStart, interp.homeContext, value)
}

func (interp *Interpreter) literal(offset int) int {
	return interp.literal_ofMethod(offset, interp.method)
}

func (interp *Interpreter) nilContextFields() {
	interp.memory.StorePointer_ofObject_withValue(SenderIndex, interp.activeContext, oops.NilPointer)
	interp.memory.StorePointer_ofObject_withValue(InstructionPointerIndex, interp.activeContext, oops.NilPointer)
}

// returnToActiveContext switches to aContext after nil-ing the outgoing
// context's Sender/IP fields, so that freeing the outgoing context (whose
// reference count typically drops to zero here) does not disturb the reference
// count of the context being returned to.
func (interp *Interpreter) returnToActiveContext(aContext int) {
	interp.memory.IncreaseReferencesTo(aContext)
	interp.nilContextFields()
	interp.memory.DecreaseReferencesTo(interp.activeContext)
	interp.activeContext = aContext
	interp.fetchContextRegisters()
}

func (interp *Interpreter) returnValue_to(resultPointer, contextPointer int) {
	sendersIP := interp.memory.FetchPointer_ofObject(InstructionPointerIndex, contextPointer)
	if contextPointer == oops.NilPointer || sendersIP == oops.NilPointer {
		interp.push(interp.activeContext)
		interp.push(resultPointer)
		interp.sendSelector_argumentCount(oops.CannotReturnSelector, 1)
		return
	}
	if dbgOn {
		isBlk := interp.isBlockContext(contextPointer)
		h := -1
		if isBlk {
			h = interp.memory.FetchPointer_ofObject(BlockHomeIndex, contextPointer)
		}
		if isBlk && h == oops.NilPointer {
			fmt.Printf("RET to ctx=%d cls=%d len=%d cyc=%d fields=", contextPointer,
				interp.memory.FetchClassOf(contextPointer), interp.memory.FetchWordLengthOf(contextPointer), interp.totalCycles)
			for i := 0; i < 6; i++ {
				fmt.Printf("%d ", interp.memory.FetchPointer_ofObject(i, contextPointer))
			}
			fmt.Println()
		}
	}
	interp.memory.IncreaseReferencesTo(resultPointer)
	interp.returnToActiveContext(contextPointer)
	interp.push(resultPointer)
	interp.memory.DecreaseReferencesTo(resultPointer)
}

// Extended Bytecodes

// extendedPushBytecode (128): descriptor is a 2-bit variable type in the high
// bits and a 6-bit index in the low bits (Bluebook extractBits 8..9 / 10..15).
func (interp *Interpreter) extendedPushBytecode() {
	descriptor := interp.fetchByte()
	variableType := (descriptor >> 6) & 3
	variableIndex := descriptor & 63
	switch variableType {
	case 0:
		interp.push(interp.memory.FetchPointer_ofObject(variableIndex, interp.receiver))
	case 1:
		interp.push(interp.temporary(variableIndex))
	case 2:
		interp.push(interp.literal(variableIndex))
	case 3:
		interp.push(interp.memory.FetchPointer_ofObject(ValueIndex, interp.literal(variableIndex)))
	}
}

// extendedStoreBytecode (129): 2-bit variable type + 6-bit index (as push).
func (interp *Interpreter) extendedStoreBytecode() {
	descriptor := interp.fetchByte()
	variableType := (descriptor >> 6) & 3
	variableIndex := descriptor & 63
	switch variableType {
	case 0:
		interp.memory.StorePointer_ofObject_withValue(variableIndex, interp.receiver, interp.stackTop())
	case 1:
		interp.storeTemporary_withValue(variableIndex, interp.stackTop())
	case 2:
		interp.error("illegal store")
	case 3:
		assoc := interp.literal(variableIndex)
		interp.memory.StorePointer_ofObject_withValue(ValueIndex, assoc, interp.stackTop())
	}
}

func (interp *Interpreter) extendedStoreAndPopBytecode() {
	interp.extendedStoreBytecode()
	interp.popStack()
}

func (interp *Interpreter) singleExtendedSendBytecode() {
	descriptor := interp.fetchByte()
	interp.sendSelector_argumentCount(interp.literal(descriptor&31), descriptor>>5)
}

func (interp *Interpreter) doubleExtendedDoSomethingBytecode() {
	descriptor1 := interp.fetchByte()
	descriptor2 := interp.fetchByte()
	opType := descriptor1 >> 5
	switch opType {
	case 0:
		interp.sendSelector_argumentCount(interp.literal(descriptor2), descriptor1&31)
	case 1:
		interp.sendSelector_argumentCount(interp.literal(descriptor2), descriptor1&31) // super send
	case 2:
		interp.push(interp.memory.FetchPointer_ofObject(descriptor2, interp.receiver))
	case 3:
		interp.push(interp.temporary(descriptor2))
	case 4:
		interp.push(interp.literal(descriptor2))
	case 5:
		interp.push(interp.memory.FetchPointer_ofObject(ValueIndex, interp.literal(descriptor2)))
	case 6:
		interp.memory.StorePointer_ofObject_withValue(descriptor2, interp.receiver, interp.stackTop())
	case 7:
		interp.storeTemporary_withValue(descriptor2, interp.stackTop())
	}
}

func (interp *Interpreter) methodClassOf(methodPointer int) int {
	literalCount := interp.literalCountOf(methodPointer)
	association := interp.literal_ofMethod(literalCount-1, methodPointer)
	return interp.memory.FetchPointer_ofObject(ValueIndex, association)
}

func (interp *Interpreter) superclassOf(classPointer int) int {
	return interp.memory.FetchPointer_ofObject(SuperclassIndex, classPointer)
}

func (interp *Interpreter) sendSelectorToClass(classPointer int) {
	interp.lookupAndExecute(interp.messageSelector, classPointer, interp.argumentCount)
}

func (interp *Interpreter) singleExtendedSuperBytecode() {
	descriptor := interp.fetchByte()
	interp.argumentCount = (descriptor >> 5) & 7
	interp.messageSelector = interp.literal(descriptor & 31)
	methodClass := interp.methodClassOf(interp.method)
	interp.sendSelectorToClass(interp.superclassOf(methodClass))
}

func (interp *Interpreter) secondExtendedSendBytecode() {
	descriptor := interp.fetchByte()
	interp.sendSelector_argumentCount(interp.literal(descriptor&63), descriptor>>6)
}

// doubleExtendedSendBytecode (bytecode 132): count and selector index follow.
func (interp *Interpreter) doubleExtendedSendBytecode() {
	count := interp.fetchByte()
	selector := interp.literal(interp.fetchByte())
	interp.sendSelector_argumentCount(selector, count)
}

// doubleExtendedSuperBytecode (bytecode 134): count and selector index follow,
// dispatched to the superclass of the method's class.
func (interp *Interpreter) doubleExtendedSuperBytecode() {
	interp.argumentCount = interp.fetchByte()
	interp.messageSelector = interp.literal(interp.fetchByte())
	methodClass := interp.methodClassOf(interp.method)
	interp.sendSelectorToClass(interp.superclassOf(methodClass))
}

// jump advances the instruction pointer by offset.
func (interp *Interpreter) jump(offset int) {
	interp.instructionPointer += offset
}

// jumpIf_by pops a boolean and jumps by offset when it equals condition.
// A non-boolean triggers a mustBeBoolean send.
func (interp *Interpreter) jumpIf_by(condition, offset int) {
	boolean := interp.popStack()
	if boolean == condition {
		interp.jump(offset)
		return
	}
	if boolean != oops.TruePointer && boolean != oops.FalsePointer {
		interp.unPop(1)
		interp.sendSelector_argumentCount(oops.MustBeBooleanSelector, 0)
	}
}

// Method Sending & Dispatch

func (interp *Interpreter) sendSelector_argumentCount(selector, argCount int) {
	interp.messageSelector = selector
	interp.argumentCount = argCount
	recv := interp.stackValue(argCount)
	cls := interp.memory.FetchClassOf(recv)
	interp.sendSelectorToClass(cls)
}

func (interp *Interpreter) lookupAndExecute(selector, classPointer, argCount int) {
	interp.messageSelector = selector
	interp.argumentCount = argCount
	for {
		meth, primIdx, found := interp.findMethodInClass(interp.messageSelector, classPointer)
		if found {
			if primIdx > 0 {
				if interp.executePrimitive(primIdx, interp.argumentCount) {
					return
				}
			} else {
				// Quick methods are handled without activating a context.
				switch interp.flagValueOf(meth) {
				case 5: // ^self — receiver already on stack top
					return
				case 6: // ^instVar
					interp.quickInstanceLoad(meth)
					return
				}
			}
			interp.executeNewMethod(meth, interp.argumentCount)
			return
		}
		// Selector not understood. Build a Message and retry with
		// doesNotUnderstand: in the same class (per Bluebook).
		if interp.messageSelector == oops.DoesNotUnderstandSelector {
			interp.error("Recursive not understood error encountered")
			return
		}
		interp.createActualMessage()
		interp.messageSelector = oops.DoesNotUnderstandSelector
		interp.argumentCount = 1
	}
}

// findMethodInClass resolves selector in classPointer's hierarchy, consulting
// and populating the method cache. found is false when the selector is absent.
func (interp *Interpreter) findMethodInClass(selector, classPointer int) (method, primIdx int, found bool) {
	hashVal := ((selector ^ classPointer) >> 1) & (MethodCacheSize - 1)
	entry := &interp.methodCache[hashVal]
	if entry.selector == selector && entry.class == classPointer {
		return entry.method, entry.prim, true
	}
	meth := interp.lookupMethodInClass(selector, classPointer)
	if meth == oops.NilPointer {
		return 0, 0, false
	}
	prim := interp.primitiveIndexOf(meth)
	entry.selector = selector
	entry.class = classPointer
	entry.method = meth
	entry.prim = prim
	return meth, prim, true
}

func (interp *Interpreter) lookupMethodInClass(selector, classPointer int) int {
	curr := classPointer
	for curr != oops.NilPointer && curr != NonPointer && !interp.memory.IsIntegerObject(curr) {
		dict := interp.memory.FetchPointer_ofObject(MessageDictionaryIndex, curr)
		meth := interp.lookupMethodInDictionary(selector, dict)
		if meth != oops.NilPointer {
			return meth
		}
		curr = interp.superclassOf(curr)
	}
	return oops.NilPointer
}

// lookupMethodInDictionary probes a method dictionary for selector using a
// linear scan with a single wrap-around (per Bluebook). Returns the matching
// CompiledMethod, or NilPointer if the selector is absent.
func (interp *Interpreter) lookupMethodInDictionary(selector, dictionary int) int {
	length := interp.memory.FetchWordLengthOf(dictionary)
	mask := length - SelectorStart - 1
	index := (mask & (selector >> 1)) + SelectorStart
	wrapAround := false
	for {
		nextSelector := interp.memory.FetchPointer_ofObject(index, dictionary)
		if nextSelector == oops.NilPointer {
			return oops.NilPointer
		}
		if nextSelector == selector {
			methodArray := interp.memory.FetchPointer_ofObject(MethodArrayIndex, dictionary)
			return interp.memory.FetchPointer_ofObject(index-SelectorStart, methodArray)
		}
		index++
		if index == length {
			if wrapAround {
				return oops.NilPointer
			}
			wrapAround = true
			index = SelectorStart
		}
	}
}

func (interp *Interpreter) flagValueOf(methodPointer int) int {
	// extractBits 0..2 of the header (Bluebook big-endian): (header >> 13) & 7
	return (interp.headerOf(methodPointer) >> 13) & 7
}

func (interp *Interpreter) fieldIndexOf(methodPointer int) int {
	// extractBits 3..7 of the header: (header >> 8) & 0x1f
	return (interp.headerOf(methodPointer) >> 8) & 0x1f
}

// quickInstanceLoad handles a flag-6 quick method (^instVar): pop the receiver
// and push the requested instance variable.
func (interp *Interpreter) quickInstanceLoad(methodPointer int) {
	thisReceiver := interp.popStack()
	fieldIndex := interp.fieldIndexOf(methodPointer)
	interp.push(interp.memory.FetchPointer_ofObject(fieldIndex, thisReceiver))
}

func (interp *Interpreter) headerExtensionOf(methodPointer int) int {
	literalCount := interp.literalCountOf(methodPointer)
	return interp.literal_ofMethod(literalCount-2, methodPointer)
}

func (interp *Interpreter) primitiveIndexOf(methodPointer int) int {
	if interp.flagValueOf(methodPointer) != 7 {
		return 0
	}
	// extractBits 7..14 of the header extension: (ext >> 1) & 0xff
	return (interp.headerExtensionOf(methodPointer) >> 1) & 0xff
}

func (interp *Interpreter) executeNewMethod(newMethod, argCount int) {
	// Method header bits are extracted from the raw header oop (Bluebook):
	//   temporaryCount   = extractBits 3..7  => (header >> 8) & 0x1f
	//   largeContextFlag = extractBits 8..8  => (header >> 7) & 1
	header := interp.headerOf(newMethod)
	tempCount := (header >> 8) & 0x1f
	largeContext := (header >> 7) & 1

	contextSize := 12 + TempFrameStart
	if largeContext == 1 {
		contextSize = 32 + TempFrameStart
	}

	newContext := interp.memory.InstantiateClass_withPointers(oops.ClassMethodContextPointer, contextSize)
	if newContext == oops.NilPointer {
		return
	}
	interp.memory.StorePointer_ofObject_withValue(SenderIndex, newContext, interp.activeContext)
	initialIP := (interp.literalCountOf(newMethod)+LiteralFrameStart)*2 + 1
	interp.storeInstructionPointerValue_inContext(initialIP, newContext)
	interp.storeStackPointerValue_inContext(tempCount, newContext)
	interp.memory.StorePointer_ofObject_withValue(MethodIndex, newContext, newMethod)

	// Transfer receiver + arguments from the sender's stack into the new
	// context: receiver -> ReceiverIndex, arg0 -> ReceiverIndex+1, and so on.
	// Remaining temporaries were already nil-initialized by the allocator.
	interp.transfer_fromIndex_ofObject_toIndex_ofObject(argCount+1, interp.stackPointer-argCount, interp.activeContext, ReceiverIndex, newContext)
	interp.pop(argCount + 1)

	interp.newActiveContext(newContext)
}

// transfer moves count fields from fromOop (starting at firstFrom) to toOop
// (starting at firstTo), nil-ing each source field so reference counts stay
// balanced (per Bluebook transfer:fromIndex:ofObject:toIndex:ofObject:).
func (interp *Interpreter) transfer_fromIndex_ofObject_toIndex_ofObject(count, firstFrom, fromOop, firstTo, toOop int) {
	fromIndex := firstFrom
	lastFrom := firstFrom + count
	toIndex := firstTo
	for fromIndex < lastFrom {
		oop := interp.memory.FetchPointer_ofObject(fromIndex, fromOop)
		interp.memory.StorePointer_ofObject_withValue(toIndex, toOop, oop)
		interp.memory.StorePointer_ofObject_withValue(fromIndex, fromOop, oops.NilPointer)
		fromIndex++
		toIndex++
	}
}

// createActualMessage reifies the current message (selector + arguments) into a
// Message object, replacing the arguments on the stack with it. Used to forward
// a not-understood send to doesNotUnderstand:.
func (interp *Interpreter) createActualMessage() {
	argumentArray := interp.memory.InstantiateClass_withPointers(oops.ClassArrayPointer, interp.argumentCount)
	message := interp.memory.InstantiateClass_withPointers(oops.ClassMessagePointer, MessageSize)
	interp.memory.StorePointer_ofObject_withValue(MessageSelectorIndex, message, interp.messageSelector)
	interp.memory.StorePointer_ofObject_withValue(MessageArgumentsIndex, message, argumentArray)

	interp.transfer_fromIndex_ofObject_toIndex_ofObject(interp.argumentCount, interp.stackPointer-(interp.argumentCount-1), interp.activeContext, 0, argumentArray)
	interp.pop(interp.argumentCount)
	interp.push(message)
	interp.argumentCount = 1
}

// Scheduler & Process Operations

func (interp *Interpreter) schedulerPointer() int {
	return interp.memory.FetchPointer_ofObject(ValueIndex, oops.SchedulerAssociationPointer)
}

func (interp *Interpreter) activeProcess() int {
	sched := interp.schedulerPointer()
	return interp.memory.FetchPointer_ofObject(ActiveProcessIndex, sched)
}

func (interp *Interpreter) firstContext() int {
	proc := interp.activeProcess()
	return interp.memory.FetchPointer_ofObject(SuspendedContextIndex, proc)
}

func (interp *Interpreter) AsynchronousSignal(aSemaphore int) {
	if aSemaphore == oops.NilPointer {
		return
	}
	if interp.semaphoreIndex < 63 {
		interp.semaphoreIndex++
		interp.semaphoreList[interp.semaphoreIndex] = aSemaphore
	}
}

func (interp *Interpreter) isInLowMemoryCondition() bool {
	return interp.oopsLeftLimit > 0 && interp.wordsLeftLimit > 0 &&
		(interp.memory.OopsLeft() < interp.oopsLeftLimit ||
			int(interp.memory.CoreLeft()) < interp.wordsLeftLimit)
}

func (interp *Interpreter) checkProcessSwitch() {
	if interp.checkLowMemory {
		memoryWasLow := interp.memoryIsLow
		interp.memoryIsLow = false
		if interp.lowSpaceSemaphore != oops.NilPointer && interp.oopsLeftLimit > 0 && interp.wordsLeftLimit > 0 {
			if interp.isInLowMemoryCondition() {
				interp.memory.GarbageCollect()
				if interp.isInLowMemoryCondition() {
					interp.memoryIsLow = true
					if !memoryWasLow {
						interp.AsynchronousSignal(interp.lowSpaceSemaphore)
					}
				}
			}
		}
		interp.checkLowMemory = false
	}

	for interp.semaphoreIndex >= 0 {
		sem := interp.semaphoreList[interp.semaphoreIndex]
		interp.semaphoreIndex--
		interp.synchronousSignal(sem)
	}
	highestProc := interp.wakeHighestPriority()
	if highestProc != oops.NilPointer {
		activeProc := interp.activeProcess()
		interp.memory.StorePointer_ofObject_withValue(SuspendedContextIndex, activeProc, interp.activeContext)
		interp.resume(highestProc)
	}
}

func (interp *Interpreter) synchronousSignal(aSemaphore int) {
	if interp.isEmptyList(aSemaphore) != 0 {
		count := interp.fetchInteger_ofObject(1, aSemaphore)
		interp.storeInteger_ofObject_withValue(1, aSemaphore, count+1)
	} else {
		proc := interp.removeFirstLinkOfList(aSemaphore)
		interp.resume(proc)
	}
}

func (interp *Interpreter) resume(aProcess int) {
	activeProc := interp.activeProcess()
	activePri := interp.fetchInteger_ofObject(PriorityIndex, activeProc)
	newPri := interp.fetchInteger_ofObject(PriorityIndex, aProcess)
	if newPri > activePri {
		interp.sleep(activeProc)
		interp.transferTo(aProcess)
	} else {
		interp.sleep(aProcess)
	}
}

func (interp *Interpreter) sleep(aProcess int) {
	pri := interp.fetchInteger_ofObject(PriorityIndex, aProcess)
	sched := interp.schedulerPointer()
	lists := interp.memory.FetchPointer_ofObject(ProcessListsIndex, sched)
	processList := interp.memory.FetchPointer_ofObject(pri-1, lists)
	interp.addLastLink_toList(aProcess, processList)
}

func (interp *Interpreter) transferTo(aProcess int) {
	sched := interp.schedulerPointer()
	interp.memory.StorePointer_ofObject_withValue(ActiveProcessIndex, sched, aProcess)
	newCntx := interp.memory.FetchPointer_ofObject(SuspendedContextIndex, aProcess)
	interp.newActiveContext(newCntx)
}

func (interp *Interpreter) wakeHighestPriority() int {
	sched := interp.schedulerPointer()
	lists := interp.memory.FetchPointer_ofObject(ProcessListsIndex, sched)
	listCount := interp.memory.FetchWordLengthOf(lists)
	for p := listCount - 1; p >= 0; p-- {
		processList := interp.memory.FetchPointer_ofObject(p, lists)
		if interp.isEmptyList(processList) == 0 {
			return interp.removeFirstLinkOfList(processList)
		}
	}
	return oops.NilPointer
}

func (interp *Interpreter) isEmptyList(aLinkedList int) int {
	first := interp.memory.FetchPointer_ofObject(FirstLinkIndex, aLinkedList)
	if first == oops.NilPointer {
		return 1
	}
	return 0
}

func (interp *Interpreter) removeFirstLinkOfList(aLinkedList int) int {
	first := interp.memory.FetchPointer_ofObject(FirstLinkIndex, aLinkedList)
	last := interp.memory.FetchPointer_ofObject(LastLinkIndex, aLinkedList)
	if first == last {
		interp.memory.StorePointer_ofObject_withValue(FirstLinkIndex, aLinkedList, oops.NilPointer)
		interp.memory.StorePointer_ofObject_withValue(LastLinkIndex, aLinkedList, oops.NilPointer)
	} else {
		next := interp.memory.FetchPointer_ofObject(NextLinkIndex, first)
		interp.memory.StorePointer_ofObject_withValue(FirstLinkIndex, aLinkedList, next)
	}
	interp.memory.StorePointer_ofObject_withValue(NextLinkIndex, first, oops.NilPointer)
	return first
}

func (interp *Interpreter) addLastLink_toList(aLink, aLinkedList int) {
	if interp.isEmptyList(aLinkedList) != 0 {
		interp.memory.StorePointer_ofObject_withValue(FirstLinkIndex, aLinkedList, aLink)
	} else {
		last := interp.memory.FetchPointer_ofObject(LastLinkIndex, aLinkedList)
		interp.memory.StorePointer_ofObject_withValue(NextLinkIndex, last, aLink)
	}
	interp.memory.StorePointer_ofObject_withValue(LastLinkIndex, aLinkedList, aLink)
	interp.memory.StorePointer_ofObject_withValue(MyListIndex, aLink, aLinkedList)
}

func betweenAnd(val, minVal, maxVal int) bool {
	return val >= minVal && val <= maxVal
}
