package hal

type HAL interface {
	// Specify the semaphore to signal on input
	SetInputSemaphore(semaphore int)

	// The number of seconds since 00:00 in the morning of January 1, 1901
	GetSmalltalkEpochTime() uint32

	// The number of milliseconds since the millisecond clock was last reset or rolled over
	GetMsClock() uint32

	// Schedule a semaphore to be signaled at a time
	SignalAt(semaphore int, msClockTime uint32)

	// Set the cursor image (a 16 word form)
	SetCursorImage(image []uint16)

	// Set/Get mouse cursor location
	SetCursorLocation(x, y int)
	GetCursorLocation() (x, y int)
	SetLinkCursor(link bool)

	// Set display size
	SetDisplaySize(width, height int) bool

	// Notify that screen contents changed
	DisplayChanged(x, y, width, height int)

	// Input queue
	NextInputWord() (uint16, bool)

	// Report catastrophic failure
	Error(message string)

	// Lifetime
	SignalQuit()
	ExitToDebugger()

	// Snapshot name
	GetImageName() string
	SetImageName(newName string)
}
