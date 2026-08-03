package gui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"smalltalk80/pkg/filesystem"
	"smalltalk80/pkg/hal"
	"smalltalk80/pkg/interpreter"
)

const (
	RedButton    = 130 // select
	YellowButton = 129 // doit
	BlueButton   = 128 // frame, close
)

var shiftMap = [128]byte{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
	11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
	21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
	31, ' ', '!', '"', '#', '$', '%', '&', '"', '(',
	')', '*', '+', '<', '_', '>', '?', ')', '!', '@',
	'#', '$', '%', '^', '&', '*', '(', ':', ':', '<',
	'+', '>', '?', '@', 'A', 'B', 'C', 'D', 'E', 'F',
	'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P',
	'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'{', '|', '}', '^', '_', '~', 'A', 'B', 'C', 'D',
	'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N',
	'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X',
	'Y', 'Z', '{', '|', '}', '~', 127,
}

type Options struct {
	RootDir        string
	SnapshotName   string
	ThreeButtons   bool
	CyclesPerFrame int
	DisplayScale   int
	Vsync          bool
	NoVsyncDelay   int
	Headless       bool
}

type EbitenVM struct {
	opts        Options
	fileSystem  filesystem.FileSystem
	interp      *interpreter.Interpreter
	startTime   time.Time
	mu          sync.Mutex
	inputQueue  []uint16

	inputSemaphore int
	scheduledSem   int
	scheduledTime  uint32

	displayWidth  int
	displayHeight int
	rgbaBuffer    []byte
	screenImage   *ebiten.Image
	dirty         bool

	cursorBits     [16]uint16
	cursorImage    *ebiten.Image
	showCursor     bool
	cursorX        int
	cursorY        int
	lastMouseX     int
	lastMouseY     int
	lastEventTime  uint32
	eventCount     int

	leftDownMods   uint32
	middleDownMods uint32
	rightDownMods  uint32

	quitSignalled bool
	imageName     string
}

func NewEbitenVM(opts Options) *EbitenVM {
	fs := filesystem.NewPosixST80FileSystem(opts.RootDir)
	vm := &EbitenVM{
		opts:          opts,
		fileSystem:    fs,
		startTime:     time.Now(),
		inputQueue:    make([]uint16, 0, 256),
		displayWidth:  640,
		displayHeight: 480,
		imageName:     opts.SnapshotName,
	}
	vm.interp = interpreter.New(vm, fs)
	return vm
}

func (vm *EbitenVM) Init() bool {
	return vm.interp.Init()
}

func (vm *EbitenVM) SetInputSemaphore(semaphore int) {
	vm.inputSemaphore = semaphore
}

func (vm *EbitenVM) GetSmalltalkEpochTime() uint32 {
	const timeOffset = 2177452800
	return uint32(time.Now().Unix()) + timeOffset
}

func (vm *EbitenVM) GetMsClock() uint32 {
	return uint32(time.Since(vm.startTime).Milliseconds())
}

func (vm *EbitenVM) SignalAt(semaphore int, msClockTime uint32) {
	vm.scheduledSem = semaphore
	vm.scheduledTime = msClockTime
	if semaphore != 0 {
		vm.checkScheduledSemaphore()
	}
}

func (vm *EbitenVM) checkScheduledSemaphore() {
	if vm.scheduledSem != 0 && vm.GetMsClock() >= vm.scheduledTime {
		vm.interp.AsynchronousSignal(vm.scheduledSem)
		vm.scheduledSem = 0
	}
}

func (vm *EbitenVM) SetCursorImage(image []uint16) {
	if len(image) >= 16 {
		copy(vm.cursorBits[:], image[:16])
		vm.updateCursorImage()
	}
}

func (vm *EbitenVM) updateCursorImage() {
	if vm.cursorImage == nil {
		vm.cursorImage = ebiten.NewImage(16, 16)
	}
	rgba := make([]byte, 16*16*4)
	for y := 0; y < 16; y++ {
		w := vm.cursorBits[y]
		for x := 0; x < 16; x++ {
			bit := (w & (1 << (15 - x))) != 0
			idx := (y*16 + x) * 4
			if bit {
				rgba[idx] = 0       // R
				rgba[idx+1] = 0     // G
				rgba[idx+2] = 0     // B
				rgba[idx+3] = 255   // A
			} else {
				rgba[idx] = 255     // R
				rgba[idx+1] = 255   // G
				rgba[idx+2] = 255   // B
				rgba[idx+3] = 0     // A (Transparent)
			}
		}
	}
	vm.cursorImage.WritePixels(rgba)
	vm.showCursor = true
}

var _ hal.HAL = (*EbitenVM)(nil)

func (vm *EbitenVM) SetCursorLocation(x, y int) {
	vm.cursorX = x
	vm.cursorY = y
}

func (vm *EbitenVM) GetCursorLocation() (x, y int) {
	if vm.opts.Headless {
		return vm.cursorX, vm.cursorY
	}
	cx, cy := ebiten.CursorPosition()
	if vm.opts.DisplayScale > 1 {
		cx /= vm.opts.DisplayScale
		cy /= vm.opts.DisplayScale
	}
	return cx, cy
}

func (vm *EbitenVM) SetLinkCursor(link bool) {}

func (vm *EbitenVM) SetDisplaySize(width, height int) bool {
	vm.displayWidth = width
	vm.displayHeight = height
	vm.rgbaBuffer = make([]byte, width*height*4)
	if !vm.opts.Headless {
		vm.screenImage = ebiten.NewImage(width, height)
		ebiten.SetWindowSize(width*vm.opts.DisplayScale, height*vm.opts.DisplayScale)
	}
	vm.dirty = true
	return true
}

func (vm *EbitenVM) DisplayChanged(x, y, width, height int) {
	vm.dirty = true
}

func (vm *EbitenVM) NextInputWord() (uint16, bool) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if len(vm.inputQueue) == 0 {
		return 0, false
	}
	w := vm.inputQueue[0]
	vm.inputQueue = vm.inputQueue[1:]
	return w, true
}

func (vm *EbitenVM) queueInputWord(word uint16) {
	vm.mu.Lock()
	vm.inputQueue = append(vm.inputQueue, word)
	vm.mu.Unlock()
	if vm.inputSemaphore != 0 {
		vm.interp.AsynchronousSignal(vm.inputSemaphore)
	}
}

func (vm *EbitenVM) queueInputWordTyped(typeVal, param uint16) {
	vm.queueInputWord(((typeVal & 0xf) << 12) | (param & 0xfff))
}

func (vm *EbitenVM) queueInputTimeWords() {
	now := vm.GetMsClock()
	var delta uint32
	if vm.eventCount == 0 {
		delta = 0
	} else {
		delta = now - vm.lastEventTime
	}
	vm.eventCount++

	if delta <= 4095 {
		vm.queueInputWordTyped(0, uint16(delta))
	} else {
		absTime := vm.GetSmalltalkEpochTime()
		vm.queueInputWordTyped(5, 0)
		vm.queueInputWord(uint16((absTime >> 16) & 0xffff))
		vm.queueInputWord(uint16(absTime & 0xffff))
	}
	vm.lastEventTime = now
}

func (vm *EbitenVM) Error(message string) {
	fmt.Fprintf(os.Stderr, "%s\n", message)
	os.Exit(1)
}

func (vm *EbitenVM) SignalQuit() {
	vm.quitSignalled = true
}

func (vm *EbitenVM) ExitToDebugger() {
	os.Exit(1)
}

func (vm *EbitenVM) GetImageName() string {
	return vm.imageName
}

func (vm *EbitenVM) SetImageName(name string) {
	vm.imageName = name
}

// Ebitengine Game interface implementation

func (vm *EbitenVM) Update() error {
	if vm.quitSignalled {
		return ebiten.Termination
	}

	vm.processInputEvents()
	vm.checkScheduledSemaphore()
	vm.interp.CheckLowMemoryConditions()

	for i := 0; i < vm.opts.CyclesPerFrame && !vm.quitSignalled; i++ {
		vm.interp.Cycle()
	}

	if !vm.opts.Vsync && vm.opts.NoVsyncDelay > 0 {
		time.Sleep(time.Duration(vm.opts.NoVsyncDelay) * time.Millisecond)
	}
	return nil
}

func (vm *EbitenVM) processInputEvents() {
	if vm.inputSemaphore == 0 {
		return
	}

	// Mouse Motion
	mx, my := vm.GetCursorLocation()
	if mx != vm.lastMouseX || my != vm.lastMouseY {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(1, uint16(mx))
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(1, uint16(my))
		vm.lastMouseX = mx
		vm.lastMouseY = my
	}

	// Mouse Buttons
	vm.handleMouseButton(ebiten.MouseButtonLeft, RedButton)
	vm.handleMouseButton(ebiten.MouseButtonRight, YellowButton)
	vm.handleMouseButton(ebiten.MouseButtonMiddle, BlueButton)

	// Keyboard Input Characters
	var chars []rune
	chars = ebiten.AppendInputChars(chars)
	for _, ch := range chars {
		if ch <= 127 {
			asciiVal := byte(ch)
			if asciiVal == '\n' {
				asciiVal = '\r'
			}
			vm.queueInputTimeWords()
			vm.queueInputWordTyped(3, uint16(asciiVal))
			vm.queueInputWordTyped(4, uint16(asciiVal))
		}
	}

	// Special keys
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(3, 13)
		vm.queueInputWordTyped(4, 13)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(3, 8)
		vm.queueInputWordTyped(4, 8)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(3, 9)
		vm.queueInputWordTyped(4, 9)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(3, 27)
		vm.queueInputWordTyped(4, 27)
	}
}

func (vm *EbitenVM) handleMouseButton(btn ebiten.MouseButton, defaultSTBtn int) {
	stBtn := defaultSTBtn
	if !vm.opts.ThreeButtons {
		if btn == ebiten.MouseButtonLeft {
			if ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyMeta) {
				stBtn = BlueButton
			} else if ebiten.IsKeyPressed(ebiten.KeyControl) {
				stBtn = YellowButton
			} else {
				stBtn = RedButton
			}
		} else if btn == ebiten.MouseButtonRight {
			stBtn = YellowButton
		}
	}

	if inpututil.IsMouseButtonJustPressed(btn) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(3, uint16(stBtn))
	}
	if inpututil.IsMouseButtonJustReleased(btn) {
		vm.queueInputTimeWords()
		vm.queueInputWordTyped(4, uint16(stBtn))
	}
}

// DumpDisplayPBM writes the current Smalltalk display bitmap to a PBM (P1) file
// for headless verification. Returns false if no display is available yet.
func (vm *EbitenVM) DumpDisplayPBM(path string) bool {
	width, height := vm.interp.GetDisplayExtent()
	if width <= 0 || height <= 0 {
		return false
	}
	displayBits := vm.interp.GetDisplayBits(width, height)
	if displayBits == 0 {
		return false
	}
	f, err := os.Create(path)
	if err != nil {
		return false
	}
	defer f.Close()
	wordsPerLine := (width + 15) / 16
	fmt.Fprintf(f, "P1\n%d %d\n", width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			w := x / 16
			b := x % 16
			word := vm.interp.FetchWord_ofDislayBits(y*wordsPerLine+w, displayBits)
			bit := 0
			if word&(1<<(15-b)) != 0 {
				bit = 1
			}
			fmt.Fprintf(f, "%d ", bit)
		}
		fmt.Fprintln(f)
	}
	return true
}

func (vm *EbitenVM) Draw(screen *ebiten.Image) {
	displayBits := vm.interp.GetDisplayBits(vm.displayWidth, vm.displayHeight)
	if displayBits != 0 && vm.dirty {
		wordsPerLine := (vm.displayWidth + 15) / 16
		for y := 0; y < vm.displayHeight; y++ {
			for w := 0; w < wordsPerLine; w++ {
				wordIndex := y*wordsPerLine + w
				sourceWord := vm.interp.FetchWord_ofDislayBits(wordIndex, displayBits)
				for b := 0; b < 16; b++ {
					x := w*16 + b
					if x >= vm.displayWidth {
						break
					}
					bit := (sourceWord & (1 << (15 - b))) != 0
					idx := (y*vm.displayWidth + x) * 4
					if bit {
						vm.rgbaBuffer[idx] = 0     // R
						vm.rgbaBuffer[idx+1] = 0   // G
						vm.rgbaBuffer[idx+2] = 0   // B
						vm.rgbaBuffer[idx+3] = 255 // A
					} else {
						vm.rgbaBuffer[idx] = 255   // R
						vm.rgbaBuffer[idx+1] = 255 // G
						vm.rgbaBuffer[idx+2] = 255 // B
						vm.rgbaBuffer[idx+3] = 255 // A
					}
				}
			}
		}
		if vm.screenImage != nil {
			vm.screenImage.WritePixels(vm.rgbaBuffer)
		}
		vm.dirty = false
	}

	if vm.screenImage != nil {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(vm.screenImage, op)
	}

	if vm.showCursor && vm.cursorImage != nil {
		mx, my := vm.GetCursorLocation()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(vm.cursorImage, op)
	}
}

func (vm *EbitenVM) Layout(outsideWidth, outsideHeight int) (int, int) {
	return vm.displayWidth, vm.displayHeight
}
