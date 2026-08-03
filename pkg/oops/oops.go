package oops

// Well known oops for various objects in an image (G&R pg. 576)

// UndefinedObject and Booleans
const (
	NilPointer   = 2
	FalsePointer = 4
	TruePointer  = 6
)

// Root
const (
	SchedulerAssociationPointer = 8
	SmalltalkPointer            = 25286 // SystemDictionary
)

// Classes
const (
	ClassSmallInteger                = 12
	ClassStringPointer               = 14
	ClassArrayPointer                = 16
	ClassMethodContextPointer        = 22
	ClassBlockContextPointer         = 24
	ClassPointPointer                = 26
	ClassLargePositiveIntegerPointer = 28
	ClassMessagePointer              = 32
	ClassCharacterPointer            = 40
	ClassCompiledMethod              = 34
	ClassSymbolPointer               = 56
	ClassFloatPointer                = 20
	ClassSemaphorePointer            = 38
	ClassDisplayScreenPointer        = 834
	ClassUndefinedObject             = 25728
)

// Selectors
const (
	DoesNotUnderstandSelector = 42
	CannotReturnSelector      = 44
	MustBeBooleanSelector     = 52
)

// Tables
const (
	SpecialSelectorsPointer = 48
	CharacterTablePointer  = 50
)
