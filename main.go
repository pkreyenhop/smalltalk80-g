package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"smalltalk80/pkg/gui"
)

func main() {
	rootDir := flag.String("directory", "", "Directory containing snapshot image and sources (required)")
	imageName := flag.String("image", "snapshot.im", "Snapshot image file name")
	cycles := flag.Int("cycles", 1800, "Number of VM instructions per update loop")
	delay := flag.Int("delay", 0, "Delay in ms after frame if vsync is off")
	scale2x := flag.Bool("2x", false, "Specifies 2x display scaling")
	vsync := flag.Bool("vsync", false, "Turn on vertical sync")
	threeButtons := flag.Bool("three", false, "Use three button mouse mapping")
	headless := flag.Bool("headless", false, "Run in headless mode without window")

	flag.Parse()

	if *rootDir == "" {
		fmt.Println("usage: Smalltalk -directory root-directory [-image snapshot] [-cycles count] [-delay ms] [-2x] [-vsync] [-three] [-headless]")
		os.Exit(1)
	}

	cleanDir := strings.TrimRight(*rootDir, "/\\")

	scale := 1
	if *scale2x {
		scale = 2
	}

	opts := gui.Options{
		RootDir:        cleanDir,
		SnapshotName:   *imageName,
		ThreeButtons:   *threeButtons,
		CyclesPerFrame: *cycles,
		DisplayScale:   scale,
		Vsync:          *vsync,
		NoVsyncDelay:   *delay,
		Headless:       *headless,
	}

	vm := gui.NewEbitenVM(opts)
	if !vm.Init() {
		fmt.Fprintf(os.Stderr, "VM failed to initialize (invalid/missing directory or snapshot?)\n")
		os.Exit(1)
	}

	if *headless {
		fmt.Println("Smalltalk-80 running in headless mode...")
		for i := 0; i < 100; i++ {
			if err := vm.Update(); err != nil {
				break
			}
		}
		if vm.DumpDisplayPBM("/tmp/st-display.pbm") {
			fmt.Println("Display dumped to /tmp/st-display.pbm")
		} else {
			fmt.Println("No display bits available to dump")
		}
		fmt.Println("Headless run completed successfully.")
		return
	}

	ebiten.SetWindowTitle("Smalltalk-80 (Go)")
	ebiten.SetVsyncEnabled(*vsync)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(vm); err != nil && err != ebiten.Termination {
		fmt.Fprintf(os.Stderr, "Error running Smalltalk-80: %v\n", err)
		os.Exit(1)
	}
}
