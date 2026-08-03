package bitblt

import (
	"smalltalk80/pkg/objmemory"
	"smalltalk80/pkg/oops"
)

type CharacterScanner struct {
	*BitBlt
	xTable         int
	lastIndex      int
	stopConditions int
}

func NewCharacterScanner(
	memory *objmemory.ObjectMemory,
	destForm, sourceForm, halftoneForm, combinationRule,
	destX, destY, width, height,
	sourceX, sourceY, clipX, clipY, clipWidth, clipHeight,
	xTable, lastIndex, stopConditions int,
) *CharacterScanner {
	bb := NewBitBlt(memory, destForm, sourceForm, halftoneForm, combinationRule, destX, destY, width, height, sourceX, sourceY, clipX, clipY, clipWidth, clipHeight)
	return &CharacterScanner{
		BitBlt:         bb,
		xTable:         xTable,
		lastIndex:      lastIndex,
		stopConditions: stopConditions,
	}
}

func (cs *CharacterScanner) UpdateDestX() int {
	return cs.destX
}

func (cs *CharacterScanner) UpdatedWidth() int {
	return cs.width
}

func (cs *CharacterScanner) UpdatedSourceX() int {
	return cs.sourceX
}

func (cs *CharacterScanner) UpdatedLastIndex() int {
	return cs.lastIndex
}

func (cs *CharacterScanner) ScanCharactersFrom_to_in_rightX_stopConditions_displaying(
	startIndex, stopIndex, sourceString, rightX, stops int, displaying bool,
) int {
	const endOfRun = 257
	const crossedX = 258

	cs.lastIndex = startIndex
	for cs.lastIndex <= stopIndex {
		asciiVal := cs.memory.FetchByte_ofObject(cs.lastIndex-1, sourceString)

		if cs.memory.FetchPointer_ofObject(asciiVal, cs.stopConditions) != oops.NilPointer {
			return cs.memory.FetchPointer_ofObject(asciiVal, stops)
		}

		cs.sourceX = cs.memory.IntegerValueOf(cs.memory.FetchPointer_ofObject(asciiVal, cs.xTable))
		cs.width = cs.memory.IntegerValueOf(cs.memory.FetchPointer_ofObject(asciiVal+1, cs.xTable)) - cs.sourceX
		nextDestX := cs.destX + cs.width

		if nextDestX > rightX {
			return cs.memory.FetchPointer_ofObject(crossedX-1, stops)
		}
		if displaying {
			cs.CopyBits()
		}
		cs.destX = nextDestX
		cs.lastIndex++
	}

	cs.lastIndex = stopIndex
	return cs.memory.FetchPointer_ofObject(endOfRun-1, stops)
}
