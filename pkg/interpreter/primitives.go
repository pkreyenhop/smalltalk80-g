package interpreter

import (
	"math"
	"time"

	"smalltalk80/pkg/bitblt"
	"smalltalk80/pkg/oops"
)

func (interp *Interpreter) executePrimitive(primitiveIndex, argCount int) bool {
	interp.initPrimitive()
	switch primitiveIndex {
	// Arithmetic Primitives (1..20)
	case 1:
		interp.primitiveAdd(argCount)
	case 2:
		interp.primitiveSubtract(argCount)
	case 3:
		interp.primitiveLessThan(argCount)
	case 4:
		interp.primitiveGreaterThan(argCount)
	case 5:
		interp.primitiveLessOrEqual(argCount)
	case 6:
		interp.primitiveGreaterOrEqual(argCount)
	case 7:
		interp.primitiveEqual(argCount)
	case 8:
		interp.primitiveNotEqual(argCount)
	case 9:
		interp.primitiveMultiply(argCount)
	case 10:
		interp.primitiveDivide(argCount)
	case 11:
		interp.primitiveMod(argCount)
	case 12:
		interp.primitiveDiv(argCount)
	case 13:
		interp.primitiveQuo(argCount)
	case 14:
		interp.primitiveBitAnd(argCount)
	case 15:
		interp.primitiveBitOr(argCount)
	case 16:
		interp.primitiveBitXor(argCount)
	case 17:
		interp.primitiveBitShift(argCount)
	case 18:
		interp.primitiveMakePoint(argCount)
	case 19:
		interp.primitiveFail()

	// Subscript & Stream Primitives (60..79)
	case 60:
		interp.primitiveAt()
	case 61:
		interp.primitiveAtPut()
	case 62:
		interp.primitiveSize()
	case 63:
		interp.primitiveStringAt()
	case 64:
		interp.primitiveStringAtPut()
	case 65:
		interp.primitiveNext()
	case 66:
		interp.primitiveNextPut()
	case 67:
		interp.primitiveAtEnd()

	// Storage Management Primitives (70..89)
	case 70:
		interp.primitiveNew()
	case 71:
		interp.primitiveNewWithArg()
	case 76:
		interp.primitiveAsObject()

	// Control Primitives (80..99)
	case 80:
		interp.primitiveBlockCopy()
	case 81:
		interp.primitiveValue()
	case 82:
		interp.primitiveValueWithArgs()
	case 83:
		interp.primitivePerform()
	case 84:
		interp.primitivePerformWithArgs()
	case 85:
		interp.primitiveSignal()
	case 86:
		interp.primitiveWait()
	case 87:
		interp.primitiveResume()
	case 88:
		interp.primitiveSuspend()
	case 89:
		interp.primitiveFlushCache()

	// System Primitives (110..129)
	case 110:
		interp.primitiveEquivalent()
	case 111:
		interp.primitiveClass()
	case 112:
		interp.primitiveCoreLeft()
	case 113:
		interp.primitiveQuit()
	case 114:
		interp.primitiveExitToDebugger()
	case 115:
		interp.primitiveOopsLeft()
	case 116:
		interp.primitiveSignalAtOopsLeftWordsLeft()

	// Input/Output & Graphics Primitives (90..109)
	case 90:
		interp.primitiveMousePoint()
	case 91:
		interp.primitiveCursorLocPut()
	case 92:
		interp.primitiveCursorLink()
	case 93:
		interp.primitiveInputSemaphore()
	case 94:
		interp.primitiveSampleInterval()
	case 95:
		interp.primitiveInputWord()
	case 96:
		interp.primitiveCopyBits()
	case 97:
		interp.primitiveSnapshot()
	case 98:
		interp.primitiveTimeWordsInto()
	case 99:
		interp.primitiveTickWordsInto()
	case 100:
		interp.primitiveSignalAtTick()
	case 101:
		interp.primitiveBeCursor()
	case 102:
		interp.primitiveBeDisplay()
	case 103:
		interp.primitiveScanCharacters()
	case 104:
		interp.primitiveDrawLoop()
	case 105:
		interp.primitiveStringReplace()

	// Vendor / Posix File Primitives (120..125)
	case 120:
		interp.primitiveBeSnapshotFile()
	case 121:
		interp.primitivePosixFileOperation()
	case 122:
		interp.primitivePosixDirectoryOperation()
	case 123:
		interp.primitivePosixLastErrorOperation()
	case 124:
		interp.primitivePosixErrorStringOperation()

	default:
		interp.primitiveFail()
	}

	if interp.successFlag {
		return true
	}
	return false
}

func (interp *Interpreter) initPrimitive() {
	interp.successFlag = true
}

func (interp *Interpreter) success(cond bool) {
	interp.successFlag = interp.successFlag && cond
}

func (interp *Interpreter) primitiveFail() {
	interp.successFlag = false
}

// Arithmetic Primitives

func (interp *Interpreter) primitiveAdd(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 + val2
		if interp.memory.IsIntegerValue(res) {
			interp.push(interp.memory.IntegerObjectOf(res))
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveSubtract(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 - val2
		if interp.memory.IsIntegerValue(res) {
			interp.push(interp.memory.IntegerObjectOf(res))
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveLessThan(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val1 < val2 {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveGreaterThan(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val1 > val2 {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveLessOrEqual(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val1 <= val2 {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveGreaterOrEqual(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val1 >= val2 {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveEqual(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		if rcvr == arg {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveNotEqual(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		if rcvr != arg {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveMultiply(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 * val2
		if interp.memory.IsIntegerValue(res) {
			interp.push(interp.memory.IntegerObjectOf(res))
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveDivide(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val2 != 0 && (val1%val2) == 0 {
			res := val1 / val2
			if interp.memory.IsIntegerValue(res) {
				interp.push(interp.memory.IntegerObjectOf(res))
				return
			}
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveMod(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val2 != 0 {
			res := val1 % val2
			if (res < 0 && val2 > 0) || (res > 0 && val2 < 0) {
				res += val2
			}
			if interp.memory.IsIntegerValue(res) {
				interp.push(interp.memory.IntegerObjectOf(res))
				return
			}
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveDiv(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val2 != 0 {
			res := int(math.Floor(float64(val1) / float64(val2)))
			if interp.memory.IsIntegerValue(res) {
				interp.push(interp.memory.IntegerObjectOf(res))
				return
			}
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveQuo(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		if val2 != 0 {
			res := val1 / val2
			if interp.memory.IsIntegerValue(res) {
				interp.push(interp.memory.IntegerObjectOf(res))
				return
			}
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveBitAnd(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 & val2
		interp.push(interp.memory.IntegerObjectOf(res))
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveBitOr(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 | val2
		interp.push(interp.memory.IntegerObjectOf(res))
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveBitXor(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		val2 := interp.memory.IntegerValueOf(arg)
		res := val1 ^ val2
		interp.push(interp.memory.IntegerObjectOf(res))
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveBitShift(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) && interp.memory.IsIntegerObject(arg) {
		val1 := interp.memory.IntegerValueOf(rcvr)
		shift := interp.memory.IntegerValueOf(arg)
		var res int
		if shift >= 0 {
			res = val1 << uint(shift)
		} else {
			res = val1 >> uint(-shift)
		}
		if interp.memory.IsIntegerValue(res) {
			interp.push(interp.memory.IntegerObjectOf(res))
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveMakePoint(argCount int) {
	arg := interp.popStack()
	rcvr := interp.popStack()
	pt := interp.memory.InstantiateClass_withPointers(oops.ClassPointPointer, 2)
	interp.memory.StorePointer_ofObject_withValue(0, pt, rcvr)
	interp.memory.StorePointer_ofObject_withValue(1, pt, arg)
	interp.push(pt)
}

// Subscript & Stream Primitives

func (interp *Interpreter) primitiveAt() {
	indexObj := interp.popStack()
	arrayObj := interp.popStack()
	if interp.memory.IsIntegerObject(indexObj) {
		idx := interp.memory.IntegerValueOf(indexObj)
		interp.checkIndexableBoundsOf_in(idx, arrayObj)
		if interp.successFlag {
			val := interp.memory.FetchPointer_ofObject(idx-1, arrayObj)
			interp.push(val)
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveAtPut() {
	valObj := interp.popStack()
	indexObj := interp.popStack()
	arrayObj := interp.popStack()
	if interp.memory.IsIntegerObject(indexObj) {
		idx := interp.memory.IntegerValueOf(indexObj)
		interp.checkIndexableBoundsOf_in(idx, arrayObj)
		if interp.successFlag {
			interp.memory.StorePointer_ofObject_withValue(idx-1, arrayObj, valObj)
			interp.push(valObj)
			return
		}
	}
	interp.unPop(3)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveSize() {
	arrayObj := interp.popStack()
	sz := interp.lengthOf(arrayObj)
	interp.push(interp.memory.IntegerObjectOf(sz))
}

func (interp *Interpreter) lengthOf(arrayObj int) int {
	cls := interp.memory.FetchClassOf(arrayObj)
	fixed := interp.fixedFieldsOf(cls)
	if interp.isWords(cls) {
		return interp.memory.FetchWordLengthOf(arrayObj) - fixed
	}
	if interp.isPointers(cls) {
		return interp.memory.FetchWordLengthOf(arrayObj) - fixed
	}
	return interp.memory.FetchByteLengthOf(arrayObj) - (fixed * 2)
}

func (interp *Interpreter) primitiveStringAt() {
	indexObj := interp.popStack()
	arrayObj := interp.popStack()
	if interp.memory.IsIntegerObject(indexObj) {
		idx := interp.memory.IntegerValueOf(indexObj)
		interp.checkIndexableBoundsOf_in(idx, arrayObj)
		if interp.successFlag {
			b := interp.memory.FetchByte_ofObject(idx-1, arrayObj)
			charObj := interp.memory.FetchPointer_ofObject(b, oops.CharacterTablePointer)
			interp.push(charObj)
			return
		}
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveStringAtPut() {
	valObj := interp.popStack()
	indexObj := interp.popStack()
	arrayObj := interp.popStack()
	if interp.memory.IsIntegerObject(indexObj) && interp.memory.FetchClassOf(valObj) == oops.ClassCharacterPointer {
		idx := interp.memory.IntegerValueOf(indexObj)
		charVal := interp.memory.FetchPointer_ofObject(0, valObj)
		ascii := interp.memory.IntegerValueOf(charVal)
		interp.checkIndexableBoundsOf_in(idx, arrayObj)
		if interp.successFlag {
			interp.memory.StoreByte_ofObject_withValue(idx-1, arrayObj, ascii)
			interp.push(valObj)
			return
		}
	}
	interp.unPop(3)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveNext() {
	stream := interp.popStack()
	array := interp.memory.FetchPointer_ofObject(StreamArrayIndex, stream)
	idx := interp.fetchInteger_ofObject(StreamIndexIndex, stream)
	limit := interp.fetchInteger_ofObject(StreamReadLimitIndex, stream)
	if idx < limit {
		idx++
		interp.storeInteger_ofObject_withValue(StreamIndexIndex, stream, idx)
		cls := interp.memory.FetchClassOf(array)
		if cls == oops.ClassStringPointer {
			b := interp.memory.FetchByte_ofObject(idx-1, array)
			charObj := interp.memory.FetchPointer_ofObject(b, oops.CharacterTablePointer)
			interp.push(charObj)
		} else {
			val := interp.memory.FetchPointer_ofObject(idx-1, array)
			interp.push(val)
		}
		return
	}
	interp.unPop(1)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveNextPut() {
	val := interp.popStack()
	stream := interp.popStack()
	array := interp.memory.FetchPointer_ofObject(StreamArrayIndex, stream)
	idx := interp.fetchInteger_ofObject(StreamIndexIndex, stream)
	limit := interp.fetchInteger_ofObject(StreamWriteLimitIndex, stream)
	if idx < limit {
		idx++
		interp.storeInteger_ofObject_withValue(StreamIndexIndex, stream, idx)
		cls := interp.memory.FetchClassOf(array)
		if cls == oops.ClassStringPointer && interp.memory.FetchClassOf(val) == oops.ClassCharacterPointer {
			charVal := interp.memory.FetchPointer_ofObject(0, val)
			ascii := interp.memory.IntegerValueOf(charVal)
			interp.memory.StoreByte_ofObject_withValue(idx-1, array, ascii)
		} else {
			interp.memory.StorePointer_ofObject_withValue(idx-1, array, val)
		}
		interp.push(val)
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveAtEnd() {
	stream := interp.popStack()
	idx := interp.fetchInteger_ofObject(StreamIndexIndex, stream)
	limit := interp.fetchInteger_ofObject(StreamReadLimitIndex, stream)
	if idx >= limit {
		interp.push(oops.TruePointer)
	} else {
		interp.push(oops.FalsePointer)
	}
}

// Storage Management Primitives

// positive16BitValueOf returns the non-negative integer value of a SmallInteger
// or a 2-byte LargePositiveInteger; it fails the primitive otherwise.
func (interp *Interpreter) positive16BitValueOf(integerPointer int) int {
	if interp.memory.IsIntegerObject(integerPointer) {
		v := interp.memory.IntegerValueOf(integerPointer)
		if v >= 0 {
			return v
		}
		interp.primitiveFail()
		return 0
	}
	if interp.memory.FetchClassOf(integerPointer) != oops.ClassLargePositiveIntegerPointer ||
		interp.memory.FetchByteLengthOf(integerPointer) != 2 {
		interp.primitiveFail()
		return 0
	}
	return interp.memory.FetchByte_ofObject(0, integerPointer) + interp.memory.FetchByte_ofObject(1, integerPointer)*256
}

// primitiveNew (70): basicNew — instantiate a fixed-size instance of the class
// on the stack, choosing pointer or word format from its instanceSpecification.
func (interp *Interpreter) primitiveNew() {
	cls := interp.popStack()
	size := interp.fixedFieldsOf(cls)
	interp.success(!interp.isIndexable(cls))
	if interp.successFlag {
		if interp.isPointers(cls) {
			interp.push(interp.memory.InstantiateClass_withPointers(cls, size))
		} else {
			interp.push(interp.memory.InstantiateClass_withWords(cls, size))
		}
	} else {
		interp.unPop(1)
	}
}

// primitiveNewWithArg (71): basicNew: — instantiate an indexable instance with
// the requested number of indexed fields plus the class's fixed fields.
func (interp *Interpreter) primitiveNewWithArg() {
	size := interp.positive16BitValueOf(interp.popStack())
	interp.success(size <= 65533)
	cls := interp.popStack()
	interp.success(interp.isIndexable(cls))
	if interp.successFlag {
		size += interp.fixedFieldsOf(cls)
		switch {
		case interp.isPointers(cls):
			interp.push(interp.memory.InstantiateClass_withPointers(cls, size))
		case interp.isWords(cls):
			interp.push(interp.memory.InstantiateClass_withWords(cls, size))
		default:
			interp.push(interp.memory.InstantiateClass_withBytes(cls, size))
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveHash() {
	h := interp.hash(interp.popStack())
	interp.push(interp.memory.IntegerObjectOf(h))
}

func (interp *Interpreter) primitiveAsObject() {
	rcvr := interp.popStack()
	if interp.memory.IsIntegerObject(rcvr) {
		val := interp.memory.IntegerValueOf(rcvr)
		interp.push(val * 2)
		return
	}
	interp.unPop(1)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveSpecialObjects() {
	interp.popStack()
	specialArray := interp.memory.FetchPointer_ofObject(ValueIndex, oops.SpecialSelectorsPointer)
	interp.push(specialArray)
}

func (interp *Interpreter) primitiveAsSymbol() {
	// Returns self if symbol
}

// Control Primitives

func (interp *Interpreter) primitiveBlockCopy() {
	blockArgumentCount := interp.popStack()
	context := interp.popStack()
	var methodContext int
	if interp.isBlockContext(context) {
		methodContext = interp.memory.FetchPointer_ofObject(BlockHomeIndex, context)
	} else {
		methodContext = context
	}
	// The new BlockContext is sized to match its home method context so it has
	// room for the block's evaluation stack.
	contextSize := interp.memory.FetchWordLengthOf(methodContext)
	newContext := interp.memory.InstantiateClass_withPointers(oops.ClassBlockContextPointer, contextSize)
	initialIP := interp.memory.IntegerObjectOf(interp.instructionPointer + 3)
	interp.memory.StorePointer_ofObject_withValue(BlockInitialIPIndex, newContext, initialIP)
	interp.memory.StorePointer_ofObject_withValue(BlockIPIndex, newContext, initialIP)
	interp.storeStackPointerValue_inContext(0, newContext)
	interp.memory.StorePointer_ofObject_withValue(BlockArgumentCountIndex, newContext, blockArgumentCount)
	interp.memory.StorePointer_ofObject_withValue(BlockHomeIndex, newContext, methodContext)
	if dbgOn && newContext == 16776 {
		println("BLOCKCOPY created 16776 home=", methodContext, "cyc=", int(interp.totalCycles))
	}
	interp.push(newContext)
}

func (interp *Interpreter) primitiveValue() {
	blockContext := interp.stackValue(interp.argumentCount)
	if interp.memory.FetchClassOf(blockContext) != oops.ClassBlockContextPointer {
		interp.primitiveFail()
		return
	}
	blockArgumentCount := interp.argumentCountOfBlock(blockContext)
	interp.success(interp.argumentCount == blockArgumentCount)
	if !interp.successFlag {
		return
	}
	interp.transfer_fromIndex_ofObject_toIndex_ofObject(interp.argumentCount,
		interp.stackPointer-interp.argumentCount+1, interp.activeContext,
		TempFrameStart, blockContext)
	interp.pop(interp.argumentCount + 1)
	initialIP := interp.memory.FetchPointer_ofObject(BlockInitialIPIndex, blockContext)
	interp.memory.StorePointer_ofObject_withValue(BlockIPIndex, blockContext, initialIP)
	interp.storeStackPointerValue_inContext(interp.argumentCount, blockContext)
	interp.memory.StorePointer_ofObject_withValue(BlockCallerIndex, blockContext, interp.activeContext)
	interp.newActiveContext(blockContext)
}

func (interp *Interpreter) primitiveValueWithArgs() {
	argsObj := interp.popStack()
	blk := interp.popStack()
	if interp.memory.FetchClassOf(blk) != oops.ClassBlockContextPointer ||
		interp.memory.FetchClassOf(argsObj) != oops.ClassArrayPointer {
		interp.unPop(2)
		interp.primitiveFail()
		return
	}
	argCount := interp.lengthOf(argsObj)
	cntxArgCount := interp.argumentCountOfBlock(blk)
	if argCount == cntxArgCount {
		interp.transfer_fromIndex_ofObject_toIndex_ofObject(argCount, 0, argsObj, TempFrameStart, blk)
		initialIP := interp.memory.FetchPointer_ofObject(BlockInitialIPIndex, blk)
		interp.memory.StorePointer_ofObject_withValue(BlockIPIndex, blk, initialIP)
		interp.storeStackPointerValue_inContext(argCount, blk)
		interp.memory.StorePointer_ofObject_withValue(BlockCallerIndex, blk, interp.activeContext)
		interp.newActiveContext(blk)
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitivePerform() {
	selector := interp.popStack()
	interp.sendSelector_argumentCount(selector, 0)
}

func (interp *Interpreter) primitivePerformWithArgs() {
	argsObj := interp.popStack()
	selector := interp.popStack()
	argCount := interp.lengthOf(argsObj)
	for i := 0; i < argCount; i++ {
		interp.push(interp.memory.FetchPointer_ofObject(i, argsObj))
	}
	interp.sendSelector_argumentCount(selector, argCount)
}

func (interp *Interpreter) primitiveSignal() {
	sem := interp.popStack()
	interp.synchronousSignal(sem)
}

func (interp *Interpreter) primitiveWait() {
	sem := interp.popStack()
	count := interp.fetchInteger_ofObject(1, sem)
	if count > 0 {
		interp.storeInteger_ofObject_withValue(1, sem, count-1)
	} else {
		proc := interp.activeProcess()
		interp.addLastLink_toList(proc, sem)
		interp.checkProcessSwitch()
	}
}

func (interp *Interpreter) primitiveResume() {
	proc := interp.popStack()
	interp.resume(proc)
}

func (interp *Interpreter) primitiveSuspend() {
	proc := interp.popStack()
	activeProc := interp.activeProcess()
	if proc == activeProc {
		interp.checkProcessSwitch()
	} else {
		list := interp.memory.FetchPointer_ofObject(MyListIndex, proc)
		interp.removeFirstLinkOfList(list)
	}
}

func (interp *Interpreter) primitiveFlushCache() {
	interp.initializeMethodCache()
}

// System Primitives

func (interp *Interpreter) primitiveClass() {
	rcvr := interp.popStack()
	cls := interp.memory.FetchClassOf(rcvr)
	interp.push(cls)
}

func (interp *Interpreter) primitiveEquivalent() {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if rcvr == arg {
		interp.push(oops.TruePointer)
	} else {
		interp.push(oops.FalsePointer)
	}
}

func (interp *Interpreter) primitiveCoreLeft() {
	interp.popStack()
	interp.push(interp.memory.IntegerObjectOf(int(interp.memory.CoreLeft())))
}

func (interp *Interpreter) primitiveQuit() {
	if interp.hal != nil {
		interp.hal.SignalQuit()
	}
}

func (interp *Interpreter) primitiveExitToDebugger() {
	if interp.hal != nil {
		interp.hal.ExitToDebugger()
	}
}

func (interp *Interpreter) primitiveOopsLeft() {
	interp.popStack()
	interp.push(interp.memory.IntegerObjectOf(interp.memory.OopsLeft()))
}

func (interp *Interpreter) primitiveSignalAtOopsLeftWordsLeft() {
	wordsLeftObj := interp.popStack()
	oopsLeftObj := interp.popStack()
	semObj := interp.popStack()
	if interp.memory.IsIntegerObject(oopsLeftObj) && interp.memory.IsIntegerObject(wordsLeftObj) {
		interp.lowSpaceSemaphore = semObj
		interp.oopsLeftLimit = interp.memory.IntegerValueOf(oopsLeftObj)
		interp.wordsLeftLimit = interp.memory.IntegerValueOf(wordsLeftObj)
		return
	}
	interp.unPop(3)
	interp.primitiveFail()
}

// IO & BitBlt Primitives

func (interp *Interpreter) primitiveMousePoint() {
	interp.popStack()
	var x, y int
	if interp.hal != nil {
		x, y = interp.hal.GetCursorLocation()
	}
	pt := interp.memory.InstantiateClass_withPointers(oops.ClassPointPointer, 2)
	interp.memory.StorePointer_ofObject_withValue(0, pt, interp.memory.IntegerObjectOf(x))
	interp.memory.StorePointer_ofObject_withValue(1, pt, interp.memory.IntegerObjectOf(y))
	interp.push(pt)
}

func (interp *Interpreter) primitiveCursorLocPut() {
	ptObj := interp.popStack()
	xObj := interp.memory.FetchPointer_ofObject(0, ptObj)
	yObj := interp.memory.FetchPointer_ofObject(1, ptObj)
	if interp.memory.IsIntegerObject(xObj) && interp.memory.IsIntegerObject(yObj) {
		x := interp.memory.IntegerValueOf(xObj)
		y := interp.memory.IntegerValueOf(yObj)
		if interp.hal != nil {
			interp.hal.SetCursorLocation(x, y)
		}
		return
	}
	interp.unPop(1)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveCursorLink() {
	linkObj := interp.popStack()
	if interp.hal != nil {
		interp.hal.SetLinkCursor(linkObj == oops.TruePointer)
	}
}

func (interp *Interpreter) primitiveInputSemaphore() {
	semObj := interp.popStack()
	if interp.hal != nil {
		interp.hal.SetInputSemaphore(semObj)
	}
}

func (interp *Interpreter) primitiveSampleInterval() {
	interp.popStack()
	interp.push(interp.memory.IntegerObjectOf(10))
}

func (interp *Interpreter) primitiveInputWord() {
	interp.popStack()
	if interp.hal != nil {
		if word, ok := interp.hal.NextInputWord(); ok {
			interp.push(interp.memory.IntegerObjectOf(int(word)))
			return
		}
	}
	interp.push(oops.NilPointer)
}

func (interp *Interpreter) primitiveCopyBits() {
	bbObj := interp.stackTop()
	destForm := interp.memory.FetchPointer_ofObject(0, bbObj)
	sourceForm := interp.memory.FetchPointer_ofObject(1, bbObj)
	halftoneForm := interp.memory.FetchPointer_ofObject(2, bbObj)
	combRule := interp.fetchInteger_ofObject(3, bbObj)
	destX := interp.fetchInteger_ofObject(4, bbObj)
	destY := interp.fetchInteger_ofObject(5, bbObj)
	width := interp.fetchInteger_ofObject(6, bbObj)
	height := interp.fetchInteger_ofObject(7, bbObj)
	sourceX := interp.fetchInteger_ofObject(8, bbObj)
	sourceY := interp.fetchInteger_ofObject(9, bbObj)
	clipX := interp.fetchInteger_ofObject(10, bbObj)
	clipY := interp.fetchInteger_ofObject(11, bbObj)
	clipWidth := interp.fetchInteger_ofObject(12, bbObj)
	clipHeight := interp.fetchInteger_ofObject(13, bbObj)

	bb := bitblt.NewBitBlt(interp.memory, destForm, sourceForm, halftoneForm, combRule, destX, destY, width, height, sourceX, sourceY, clipX, clipY, clipWidth, clipHeight)
	if bb.CopyBits() {
		ux, uy, uw, uh := bb.GetUpdatedBounds()
		if uw > 0 && uh > 0 {
			interp.updateDisplay(destForm, uh, uw, ux, uy)
		}
	} else {
		interp.primitiveFail()
	}
}

func (interp *Interpreter) updateDisplay(destForm, updatedHeight, updatedWidth, updatedX, updatedY int) {
	if interp.currentDisplay != 0 && destForm == interp.currentDisplay {
		if interp.hal != nil {
			interp.hal.DisplayChanged(updatedX, updatedY, updatedWidth, updatedHeight)
		}
	}
}

func (interp *Interpreter) primitiveSnapshot() {
	interp.popStack()
	imageName := "snapshot.im"
	if interp.hal != nil && interp.hal.GetImageName() != "" {
		imageName = interp.hal.GetImageName()
	}
	if interp.memory.SaveSnapshot(interp.fileSystem, imageName) {
		interp.push(oops.TruePointer)
	} else {
		interp.push(oops.FalsePointer)
	}
}

func (interp *Interpreter) primitiveTimeWordsInto() {
	arrayObj := interp.popStack()
	epochSecs := uint32(0)
	if interp.hal != nil {
		epochSecs = interp.hal.GetSmalltalkEpochTime()
	} else {
		const offset = 2177452800
		epochSecs = uint32(time.Now().Unix()) + offset
	}
	highWord := int((epochSecs >> 16) & 0xffff)
	lowWord := int(epochSecs & 0xffff)
	interp.memory.StorePointer_ofObject_withValue(0, arrayObj, interp.memory.IntegerObjectOf(highWord))
	interp.memory.StorePointer_ofObject_withValue(1, arrayObj, interp.memory.IntegerObjectOf(lowWord))
}

func (interp *Interpreter) primitiveTickWordsInto() {
	arrayObj := interp.popStack()
	msClock := uint32(0)
	if interp.hal != nil {
		msClock = interp.hal.GetMsClock()
	} else {
		msClock = uint32(time.Now().UnixMilli() & 0xffffffff)
	}
	highWord := int((msClock >> 16) & 0xffff)
	lowWord := int(msClock & 0xffff)
	interp.memory.StorePointer_ofObject_withValue(0, arrayObj, interp.memory.IntegerObjectOf(highWord))
	interp.memory.StorePointer_ofObject_withValue(1, arrayObj, interp.memory.IntegerObjectOf(lowWord))
}

func (interp *Interpreter) primitiveSignalAtTick() {
	tickObj := interp.popStack()
	semObj := interp.popStack()
	if interp.memory.IsIntegerObject(tickObj) {
		ticks := interp.memory.IntegerValueOf(tickObj)
		if interp.hal != nil {
			interp.hal.SignalAt(semObj, uint32(ticks))
		}
		return
	}
	interp.unPop(2)
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveBeCursor() {
	cursorForm := interp.popStack()
	bitsObj := interp.memory.FetchPointer_ofObject(0, cursorForm)
	if interp.memory.FetchWordLengthOf(bitsObj) >= 16 {
		img := make([]uint16, 16)
		for i := 0; i < 16; i++ {
			img[i] = uint16(interp.memory.FetchWord_ofObject(i, bitsObj))
		}
		if interp.hal != nil {
			interp.hal.SetCursorImage(img)
		}
	}
}

func (interp *Interpreter) primitiveBeDisplay() {
	// Register the receiver Form as the current display. The receiver is left on
	// the stack (beDisplay returns self). Smalltalk shrinks the display to
	// height 100 before saving the image; ignore that transient resize.
	newDisplay := interp.stackTop()
	if interp.currentDisplay == newDisplay {
		return
	}
	w := interp.fetchInteger_ofObject(1, newDisplay)
	h := interp.fetchInteger_ofObject(2, newDisplay)
	if h > 100 {
		if interp.hal != nil {
			interp.hal.SetDisplaySize(w, h)
		}
		interp.currentDisplay = newDisplay
		interp.currentDisplayWidth = w
		interp.currentDisplayHeight = h
	} else {
		interp.currentDisplay = 0
	}
}

func (interp *Interpreter) primitiveScanCharacters() {
	displayingObj := interp.popStack()
	stopsObj := interp.popStack()
	rightXObj := interp.popStack()
	sourceStringObj := interp.popStack()
	stopIndexObj := interp.popStack()
	startIndexObj := interp.popStack()
	scannerObj := interp.popStack()

	displaying := (displayingObj == oops.TruePointer)
	stops := stopsObj
	rightX := interp.memory.IntegerValueOf(rightXObj)
	sourceString := sourceStringObj
	stopIndex := interp.memory.IntegerValueOf(stopIndexObj)
	startIndex := interp.memory.IntegerValueOf(startIndexObj)

	destForm := interp.memory.FetchPointer_ofObject(0, scannerObj)
	sourceForm := interp.memory.FetchPointer_ofObject(1, scannerObj)
	halftoneForm := interp.memory.FetchPointer_ofObject(2, scannerObj)
	combRule := interp.fetchInteger_ofObject(3, scannerObj)
	destX := interp.fetchInteger_ofObject(4, scannerObj)
	destY := interp.fetchInteger_ofObject(5, scannerObj)
	width := interp.fetchInteger_ofObject(6, scannerObj)
	height := interp.fetchInteger_ofObject(7, scannerObj)
	sourceX := interp.fetchInteger_ofObject(8, scannerObj)
	sourceY := interp.fetchInteger_ofObject(9, scannerObj)
	clipX := interp.fetchInteger_ofObject(10, scannerObj)
	clipY := interp.fetchInteger_ofObject(11, scannerObj)
	clipWidth := interp.fetchInteger_ofObject(12, scannerObj)
	clipHeight := interp.fetchInteger_ofObject(13, scannerObj)
	lastIndex := interp.fetchInteger_ofObject(14, scannerObj)
	xTable := interp.memory.FetchPointer_ofObject(15, scannerObj)
	stopConditions := interp.memory.FetchPointer_ofObject(16, scannerObj)

	cs := bitblt.NewCharacterScanner(
		interp.memory, destForm, sourceForm, halftoneForm, combRule,
		destX, destY, width, height, sourceX, sourceY,
		clipX, clipY, clipWidth, clipHeight,
		xTable, lastIndex, stopConditions,
	)

	result := cs.ScanCharactersFrom_to_in_rightX_stopConditions_displaying(startIndex, stopIndex, sourceString, rightX, stops, displaying)

	interp.storeInteger_ofObject_withValue(4, scannerObj, cs.UpdateDestX())
	interp.storeInteger_ofObject_withValue(6, scannerObj, cs.UpdatedWidth())
	interp.storeInteger_ofObject_withValue(8, scannerObj, cs.UpdatedSourceX())
	interp.storeInteger_ofObject_withValue(14, scannerObj, cs.UpdatedLastIndex())

	if displaying {
		ux, uy, uw, uh := cs.GetUpdatedBounds()
		if uw > 0 && uh > 0 {
			interp.updateDisplay(destForm, uh, uw, ux, uy)
		}
	}
	interp.push(result)
}

func (interp *Interpreter) primitiveDrawLoop() {
	// Optional line drawing helper
	interp.primitiveFail()
}

func (interp *Interpreter) primitiveStringReplace() {
	interp.primitiveFail()
}

// Vendor / POSIX File Primitives

func (interp *Interpreter) primitiveBeSnapshotFile() {
	nameObj := interp.popStack()
	nLen := interp.memory.FetchByteLengthOf(nameObj)
	bytes := make([]byte, nLen)
	for i := 0; i < nLen; i++ {
		bytes[i] = byte(interp.memory.FetchByte_ofObject(i, nameObj))
	}
	if interp.hal != nil {
		interp.hal.SetImageName(string(bytes))
	}
	interp.push(oops.TruePointer)
}

func (interp *Interpreter) primitivePosixFileOperation() {
	opObj := interp.popStack()
	if !interp.memory.IsIntegerObject(opObj) {
		interp.unPop(1)
		interp.primitiveFail()
		return
	}
	op := interp.memory.IntegerValueOf(opObj)
	switch op {
	case 1: // Open file: (name)
		nameObj := interp.popStack()
		nLen := interp.memory.FetchByteLengthOf(nameObj)
		bytes := make([]byte, nLen)
		for i := 0; i < nLen; i++ {
			bytes[i] = byte(interp.memory.FetchByte_ofObject(i, nameObj))
		}
		fd := interp.fileSystem.OpenFile(string(bytes))
		interp.push(interp.memory.IntegerObjectOf(fd))
	case 2: // Create file: (name)
		nameObj := interp.popStack()
		nLen := interp.memory.FetchByteLengthOf(nameObj)
		bytes := make([]byte, nLen)
		for i := 0; i < nLen; i++ {
			bytes[i] = byte(interp.memory.FetchByte_ofObject(i, nameObj))
		}
		fd := interp.fileSystem.CreateFile(string(bytes))
		interp.push(interp.memory.IntegerObjectOf(fd))
	case 3: // Close file: (fd)
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		res := interp.fileSystem.CloseFile(fd)
		interp.push(interp.memory.IntegerObjectOf(res))
	case 4: // Read file: (fd, buffer, count)
		countObj := interp.popStack()
		bufObj := interp.popStack()
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		count := interp.memory.IntegerValueOf(countObj)
		buf := make([]byte, count)
		n := interp.fileSystem.Read(fd, buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				interp.memory.StoreByte_ofObject_withValue(i, bufObj, int(buf[i]))
			}
		}
		interp.push(interp.memory.IntegerObjectOf(n))
	case 5: // Write file: (fd, buffer, count)
		countObj := interp.popStack()
		bufObj := interp.popStack()
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		count := interp.memory.IntegerValueOf(countObj)
		buf := make([]byte, count)
		for i := 0; i < count; i++ {
			buf[i] = byte(interp.memory.FetchByte_ofObject(i, bufObj))
		}
		n := interp.fileSystem.Write(fd, buf)
		interp.push(interp.memory.IntegerObjectOf(n))
	case 6: // Seek: (fd, pos)
		posObj := interp.popStack()
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		pos := interp.memory.IntegerValueOf(posObj)
		res := interp.fileSystem.SeekTo(fd, pos)
		interp.push(interp.memory.IntegerObjectOf(res))
	case 7: // File size: (fd)
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		res := interp.fileSystem.FileSize(fd)
		interp.push(interp.memory.IntegerObjectOf(res))
	case 8: // Truncate: (fd, length)
		lenObj := interp.popStack()
		fdObj := interp.popStack()
		fd := interp.memory.IntegerValueOf(fdObj)
		length := interp.memory.IntegerValueOf(lenObj)
		if interp.fileSystem.TruncateTo(fd, length) {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitivePosixDirectoryOperation() {
	opObj := interp.popStack()
	if !interp.memory.IsIntegerObject(opObj) {
		interp.unPop(1)
		interp.primitiveFail()
		return
	}
	op := interp.memory.IntegerValueOf(opObj)
	switch op {
	case 1: // Delete file: (name)
		nameObj := interp.popStack()
		nLen := interp.memory.FetchByteLengthOf(nameObj)
		bytes := make([]byte, nLen)
		for i := 0; i < nLen; i++ {
			bytes[i] = byte(interp.memory.FetchByte_ofObject(i, nameObj))
		}
		if interp.fileSystem.DeleteFile(string(bytes)) {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
	case 2: // Rename file: (old, new)
		newObj := interp.popStack()
		oldObj := interp.popStack()
		oLen := interp.memory.FetchByteLengthOf(oldObj)
		nLen := interp.memory.FetchByteLengthOf(newObj)
		oldBytes := make([]byte, oLen)
		newBytes := make([]byte, nLen)
		for i := 0; i < oLen; i++ {
			oldBytes[i] = byte(interp.memory.FetchByte_ofObject(i, oldObj))
		}
		for i := 0; i < nLen; i++ {
			newBytes[i] = byte(interp.memory.FetchByte_ofObject(i, newObj))
		}
		if interp.fileSystem.RenameFile(string(oldBytes), string(newBytes)) {
			interp.push(oops.TruePointer)
		} else {
			interp.push(oops.FalsePointer)
		}
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitivePosixLastErrorOperation() {
	interp.popStack()
	errCode := interp.fileSystem.LastError()
	interp.push(interp.memory.IntegerObjectOf(errCode))
}

func (interp *Interpreter) primitivePosixErrorStringOperation() {
	codeObj := interp.popStack()
	code := interp.memory.IntegerValueOf(codeObj)
	msg := interp.fileSystem.ErrorText(code)
	strObj := interp.memory.InstantiateClass_withBytes(oops.ClassStringPointer, len(msg))
	for i := 0; i < len(msg); i++ {
		interp.memory.StoreByte_ofObject_withValue(i, strObj, int(msg[i]))
	}
	interp.push(strObj)
}
