package cad

import (
	"fmt"
	"image/color"

	"github.com/chewxy/math32"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// PartSpec describes a printable Carousmell part, its material, print orientation,
// exploded-view offset, and visual rendering color.
type PartSpec struct {
	Name        string
	Description string
	Material    string
	PrintNote   string
	Color       color.RGBA
	ExplodeZ    float32 // Vertical shift (mm) used in exploded assembly visualizations
	Build       func(bld *gsdf.Builder, p Params) glbuild.Shader3D
	// PrintTransform transforms the part from assembled global coordinates into its
	// optimal support-free Creality K1C print orientation resting flat on Z = 0.
	PrintTransform func(bld *gsdf.Builder, s glbuild.Shader3D, p Params) glbuild.Shader3D
}

// PrintableParts returns the 5 modular printable enclosure parts plus the calibration coupon.
func PrintableParts() []PartSpec {
	return []PartSpec{
		{
			Name:        "base",
			Description: "Electronics housing tub (Pi Zero 2 WH, ULN2003, M12 button)",
			Material:    "Hyper PLA",
			PrintNote:   "Prints right-side-up on Z=0, 100% support-free",
			Color:       color.RGBA{R: 62, G: 76, B: 94, A: 255}, // Slate charcoal
			ExplodeZ:    0,
			Build:       BuildBase,
			PrintTransform: func(_ *gsdf.Builder, s glbuild.Shader3D, _ Params) glbuild.Shader3D {
				return s
			},
		},
		{
			Name:        "motor_deck",
			Description: "Stepper motor & KY-024 Hall-sensor vapor barrier divider plate",
			Material:    "Hyper PLA / PETG",
			PrintNote:   "Exported upside-down on Z=0 so Hall rails print upward, 100% support-free",
			Color:       color.RGBA{R: 108, G: 122, B: 137, A: 255}, // Cool steel gray
			ExplodeZ:    20,
			Build:       BuildMotorDeck,
			PrintTransform: func(bld *gsdf.Builder, s glbuild.Shader3D, p Params) glbuild.Shader3D {
				// Flip 180 deg around X so top (Z = BaseHeight) rests on Z = 0.
				s = bld.Rotate(s, math32.Pi, axisX)
				return bld.Translate(s, 0, 0, p.BaseHeight)
			},
		},
		{
			Name:        "carousel",
			Description: "11-slot rotary carousel (10 vials + Slot 0 park/magnet + 11 radial fins)",
			Material:    "PETG (oil-vapor resistant)",
			PrintNote:   "Prints right-side-up on Z=0, 100% support-free",
			Color:       color.RGBA{R: 46, G: 168, B: 158, A: 255}, // Teal PETG
			ExplodeZ:    45,
			Build:       BuildCarousel,
			PrintTransform: func(bld *gsdf.Builder, s glbuild.Shader3D, p Params) glbuild.Shader3D {
				return bld.Translate(s, 0, 0, -p.CarouselBottomZ())
			},
		},
		{
			Name:        "lid_labyrinth",
			Description: "Stationary labyrinth seal plate, outer shell & flared front exhaust nozzle",
			Material:    "PETG (oil-vapor resistant)",
			PrintNote:   "Exported upside-down on Z=0 (ceiling on bed), 100% support-free",
			Color:       color.RGBA{R: 224, G: 230, B: 236, A: 255}, // Matte off-white PETG
			ExplodeZ:    92,
			Build:       BuildLidLabyrinth,
			PrintTransform: func(bld *gsdf.Builder, s glbuild.Shader3D, p Params) glbuild.Shader3D {
				// Flip 180 deg around X so flat ceiling (Z = LidTopZ) rests on Z = 0.
				s = bld.Rotate(s, math32.Pi, axisX)
				return bld.Translate(s, 0, 0, p.LidTopZ())
			},
		},
		{
			Name:        "duct_hood",
			Description: "45-degree 40mm Noctua fan cradle & smooth curved turning duct",
			Material:    "PETG (oil-vapor resistant)",
			PrintNote:   "Prints right-side-up on Z=0, 45-degree overhang limit, 100% support-free",
			Color:       color.RGBA{R: 230, G: 145, B: 68, A: 255}, // Warm amber-copper accent
			ExplodeZ:    136,
			Build:       BuildDuctHood,
			PrintTransform: func(bld *gsdf.Builder, s glbuild.Shader3D, p Params) glbuild.Shader3D {
				return bld.Translate(s, 0, 0, -p.HoodBottomZ())
			},
		},
		{
			Name:        "calibration_coupon",
			Description: "10-minute Creality K1C fit-check plate (double-D bores, vial, magnet, M3, M12)",
			Material:    "Hyper PLA & PETG",
			PrintNote:   "Prints flat on Z=0, 100% support-free",
			Color:       color.RGBA{R: 155, G: 114, B: 207, A: 255},
			ExplodeZ:    0,
			Build:       BuildCalibrationCoupon,
			PrintTransform: func(_ *gsdf.Builder, s glbuild.Shader3D, _ Params) glbuild.Shader3D {
				return s
			},
		},
	}
}

// AssemblyParts returns the 5 parts that make up the assembled Carousmell device
// (excluding calibration_coupon).
func AssemblyParts() []PartSpec {
	all := PrintableParts()
	return all[:5]
}

// FindPart returns the PartSpec with the given name.
func FindPart(name string) (PartSpec, error) {
	for _, spec := range PrintableParts() {
		if spec.Name == name {
			return spec, nil
		}
	}
	return PartSpec{}, fmt.Errorf("unknown part %q (valid: base, motor_deck, carousel, lid_labyrinth, duct_hood, calibration_coupon)", name)
}

// BuildAssembly returns the combined assembled SDF of all 5 enclosure parts.
// If cutaway is true, the Y < 0 front half is sliced away to expose the XZ airflow cross-section.
func BuildAssembly(bld *gsdf.Builder, p Params, cutaway bool) glbuild.Shader3D {
	specs := AssemblyParts()
	shapes := make([]glbuild.Shader3D, 0, len(specs))
	for _, sp := range specs {
		shapes = append(shapes, sp.Build(bld, p))
	}
	asm := unionAll(bld, shapes...)
	if cutaway {
		clip := boxAt(bld, -100, 100, -100, 0, -10, 200, 0)
		asm = bld.Difference(asm, clip)
	}
	return asm
}

// HardwareMockup holds non-printed hardware silhouettes (vials, wicks, stepper, fan, Pi)
// used only for engineering visualizations (PNG/SVG) to illustrate how parts fit together.
type HardwareMockup struct {
	Name  string
	Color color.RGBA
	Shape glbuild.Shader3D
}

// BuildHardwareMockups creates simplified visual silhouettes of the real COTS hardware
// (10 amber vials, 10 fiber wicks, 28BYJ-48 stepper, Noctua 40mm fan, Raspberry Pi Zero 2 WH)
// for visual verification in rendered cutaway/isometric images.
func BuildHardwareMockups(bld *gsdf.Builder, p Params) []HardwareMockup {
	// 1. 10 Amber Glass Vials (Slots 1..10, rotated by +dTheta from Slot 0)
	zVialBot := p.CarouselBottomZ() + p.CarouselFloor + 0.2
	vialBody := cylZAt(bld, p.VialPitchR, 0, p.VialOD/2-0.1, zVialBot, zVialBot+p.VialTotalH-6.0)
	vialNeck := cylZAt(bld, p.VialPitchR, 0, 4.5, zVialBot+p.VialTotalH-6.5, zVialBot+p.VialTotalH)
	vial := bld.Union(vialBody, vialNeck)
	vials := bld.Rotate(bld.CircularArray(vial, p.NumSlots-1, p.NumSlots), p.SlotAngleRad(), axisZ)
	// Also rotate a single "active" vial preview? Keep Slot 0 empty as parked, OR show vials in Slots 1..10.
	// Note: in cutaway along Y=0, Slot 0 is at +X (parked); let's also check if rotating carousel by
	// one slot in cutaway makes the active vial & wick super clear! We handle that in render_png.go.

	// 2. 10 Black Fiber Wicks (Ø3 mm protruding 8 mm above vial cap)
	wick := cylZAt(bld, p.VialPitchR, 0, 1.5, zVialBot+2.0, zVialBot+p.VialTotalH+p.WickProtrusion)
	wicks := bld.Rotate(bld.CircularArray(wick, p.NumSlots-1, p.NumSlots), p.SlotAngleRad(), axisZ)

	// 3. 28BYJ-48 Stepper Motor (body centered at (0, -8), shaft at (0, 0))
	zFlange := p.MotorDeckBottomZ()
	stepBody := cylZAt(bld, 0, p.StepperShaftOffsetY, p.StepperBodyD/2, zFlange-p.StepperBodyH, zFlange)
	stepBlue := boxAt(bld, -7.5, 7.5, p.StepperShaftOffsetY-17.5, p.StepperShaftOffsetY-10.0, zFlange-p.StepperBodyH+1.0, zFlange-1.0, 0.5)
	stepShaft := cylZ(bld, p.StepperShaftD/2, zFlange, p.ShaftTipZ())
	stepper := bld.Union(stepBody, stepBlue, stepShaft)

	// 4. Noctua NF-A4x10 5V PWM Fan (40x40x10 mm mounted behind the 45 deg plate)
	tiltRad := p.FanTiltDeg * math32.Pi / 180
	fanRotY := math32.Pi/2 + tiltRad
	fanFrame := bld.NewBox(p.FanSize, p.FanSize, p.FanThick, 1.0)
	fanHole := bld.NewCylinder(p.FanBoreD/2-0.5, p.FanThick+2.0, 0)
	fanHub := bld.NewCylinder(8.0, p.FanThick-1.0, 0.5)
	fan := bld.Union(bld.Difference(fanFrame, fanHole), fanHub)
	fan = bld.Translate(fan, 0, 0, -p.FanPlateThick-p.FanThick/2)
	fan = bld.Rotate(fan, fanRotY, axisY)
	fan = bld.Translate(fan, p.FanCenterX, 0, p.HoodBottomZ()+p.FanCenterZOff)

	// 5. Raspberry Pi Zero 2 WH PCB silhouette on standoffs
	piZ0 := p.BaseFloor + p.PiStandoffH
	piPCB := boxAt(bld, -p.PiLength/2, p.PiLength/2, p.PiCenterY-p.PiWidth/2, p.PiCenterY+p.PiWidth/2, piZ0, piZ0+1.6, 0.4)
	piHdr := boxAt(bld, -25.4, 25.4, p.PiCenterY-p.PiWidth/2+1.0, p.PiCenterY-p.PiWidth/2+6.0, piZ0+1.6, piZ0+10.0, 0)
	pi := bld.Union(piPCB, piHdr)

	return []HardwareMockup{
		{Name: "vials", Color: color.RGBA{R: 186, G: 104, B: 30, A: 255}, Shape: vials},
		{Name: "wicks", Color: color.RGBA{R: 38, G: 40, B: 46, A: 255}, Shape: wicks},
		{Name: "stepper", Color: color.RGBA{R: 175, G: 185, B: 196, A: 255}, Shape: stepper},
		{Name: "noctua_fan", Color: color.RGBA{R: 162, G: 120, B: 92, A: 255}, Shape: fan},
		{Name: "pi_zero", Color: color.RGBA{R: 48, G: 138, B: 74, A: 255}, Shape: pi},
	}
}
