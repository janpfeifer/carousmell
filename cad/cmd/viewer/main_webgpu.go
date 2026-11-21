//go:build webgpu

// Command viewer launches the native GPU raymarched gsdf visualizer window (GLFW + WebGPU).
// Requires system Wayland/X11/macOS development headers; build with:
//
//	go run -tags webgpu ./cad/cmd/viewer --part assembly --cutaway
package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/janpfeifer/carousmell/cad"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
	"github.com/soypat/gsdf/gsdfaux"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	partFlag := flag.String("part", "assembly", "Part to view: assembly, base, motor_deck, carousel, lid_labyrinth, duct_hood, calibration_coupon")
	cutaway := flag.Bool("cutaway", false, "Slice Y < 0 half-space in assembly view")
	flag.Parse()

	p := cad.DefaultParams()
	var bld gsdf.Builder
	var shape glbuild.Shader3D
	if *partFlag == "assembly" {
		shape = cad.BuildAssembly(&bld, p, *cutaway)
	} else {
		sp, err := cad.FindPart(*partFlag)
		if err != nil {
			log.Fatal(err)
		}
		shape = sp.Build(&bld, p)
	}
	if err := bld.Err(); err != nil {
		log.Fatalf("building shape: %v", err)
	}
	// Center around origin for orbit camera.
	c := shape.Bounds().Center()
	shape = bld.Translate(shape, -c.X, -c.Y, -c.Z)
	if err := gsdfaux.UI(shape, gsdfaux.UIConfig{Width: 1024, Height: 768}); err != nil {
		log.Fatalf("gsdf WebGPU UI: %v", err)
	}
}
