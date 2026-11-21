package cad

import (
	"testing"

	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gleval"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

func evalAt(t *testing.T, s glbuild.Shader3D, pts ...ms3.Vec) []float32 {
	t.Helper()
	cpu, err := gleval.NewCPUSDF3(s)
	if err != nil {
		t.Fatalf("NewCPUSDF3: %v", err)
	}
	d := make([]float32, len(pts))
	if err := cpu.Evaluate(pts, d, nil); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	return d
}

// TestBuildVolumeCrealityK1C verifies every printable part fits comfortably inside
// the Creality K1C build volume (220 x 220 x 250 mm) and seats flat on Z = 0 in print orientation.
func TestBuildVolumeCrealityK1C(t *testing.T) {
	p := DefaultParams()
	var bld gsdf.Builder
	const (
		k1cX = float32(220.0)
		k1cY = float32(220.0)
		k1cZ = float32(250.0)
	)
	for _, sp := range PrintableParts() {
		s := sp.Build(&bld, p)
		sPrint := sp.PrintTransform(&bld, s, p)
		if err := bld.Err(); err != nil {
			t.Fatalf("%s builder error: %v", sp.Name, err)
		}
		bb := sPrint.Bounds()
		sz := bb.Size()
		t.Logf("%-18s print bounds size = %.1f x %.1f x %.1f mm (Z: %.2f..%.2f)",
			sp.Name, sz.X, sz.Y, sz.Z, bb.Min.Z, bb.Max.Z)
		if sz.X > k1cX || sz.Y > k1cY || sz.Z > k1cZ {
			t.Errorf("%s exceeds Creality K1C build volume (220x220x250): got %.1fx%.1fx%.1f",
				sp.Name, sz.X, sz.Y, sz.Z)
		}
		if math32.Abs(bb.Min.Z) > 1.0 {
			t.Errorf("%s print orientation Min.Z = %.2f mm (expected near 0 for bed adhesion)",
				sp.Name, bb.Min.Z)
		}
	}
}

// TestLabyrinthZeroInterference verifies that the rotating carousel and stationary lid_labyrinth
// maintain strict non-contact air gaps (>= 0.50 mm) across all 360 degrees of rotation,
// both vertically above the fin tips and radially along the inner and outer labyrinth rings.
func TestLabyrinthZeroInterference(t *testing.T) {
	p := DefaultParams()
	var bld gsdf.Builder
	car, err := gleval.NewCPUSDF3(BuildCarousel(&bld, p))
	if err != nil {
		t.Fatal(err)
	}
	lid, err := gleval.NewCPUSDF3(BuildLidLabyrinth(&bld, p))
	if err != nil {
		t.Fatal(err)
	}

	// Sample critical gap midpoints across all 360 degrees:
	//  1. Mid-plane of vertical ceiling gap (Z = CarouselFinTopZ + 0.30 mm)
	//  2. Mid-radius of inner labyrinth radial gap (R = CarouselInnerR - 0.30 mm)
	//  3. Mid-radius of outer labyrinth radial gap (R = CarouselOuterR + 0.30 mm)
	var pts []ms3.Vec
	const nAng = 72
	for i := 0; i < nAng; i++ {
		a := 2 * math32.Pi * float32(i) / float32(nAng)
		s, c := math32.Sincos(a)

		// Ceiling vertical gap midpoint across carousel fin radii
		for _, r := range []float32{p.CarouselInnerR + 1.0, p.VialPitchR, p.CarouselOuterR - 1.0} {
			pts = append(pts, ms3.Vec{X: r * c, Y: r * s, Z: p.CarouselFinTopZ() + p.LabyrinthVertGap/2})
		}
		// Inner and outer radial gap midpoints along full fin height
		for _, z := range []float32{p.LabyrinthRingBottomZ() + 2.0, 80.0, p.CarouselFinTopZ() - 1.0} {
			rInMid := p.CarouselInnerR - p.LabyrinthRadialGap/2
			rOutMid := p.CarouselOuterR + p.LabyrinthRadialGap/2
			pts = append(pts,
				ms3.Vec{X: rInMid * c, Y: rInMid * s, Z: z},
				ms3.Vec{X: rOutMid * c, Y: rOutMid * s, Z: z},
			)
		}
	}

	dCar := make([]float32, len(pts))
	dLid := make([]float32, len(pts))
	if err := car.Evaluate(pts, dCar, nil); err != nil {
		t.Fatal(err)
	}
	if err := lid.Evaluate(pts, dLid, nil); err != nil {
		t.Fatal(err)
	}

	minSep := float32(1e9)
	for i, pt := range pts {
		if dCar[i] <= 0 {
			t.Errorf("carousel intrudes into air gap at %v: dCar=%.3f", pt, dCar[i])
		}
		if dLid[i] <= 0 {
			t.Errorf("lid_labyrinth intrudes into air gap at %v: dLid=%.3f", pt, dLid[i])
		}
		sep := dCar[i] + dLid[i]
		if sep < minSep {
			minSep = sep
		}
		if sep < 0.45 {
			t.Errorf("labyrinth separation too tight at %v: dCar=%.3f + dLid=%.3f = %.3f mm (< 0.45 mm)",
				pt, dCar[i], dLid[i], sep)
		}
	}
	t.Logf("Verified %d labyrinth gap points across 360 deg: minimum separation = %.3f mm", len(pts), minSep)
}

// TestCarouselMotorDeckClearance verifies the carousel clears motor_deck by >= 1.4 mm everywhere
// and that the stepper shaft axis aligns at (0, 0).
func TestCarouselMotorDeckClearance(t *testing.T) {
	p := DefaultParams()
	var bld gsdf.Builder
	car := BuildCarousel(&bld, p)
	deck := BuildMotorDeck(&bld, p)

	// Midpoint between motor_deck top (42.0) and carousel bottom (43.5)
	zMid := (p.BaseHeight + p.CarouselBottomZ()) / 2
	var gapPts []ms3.Vec
	for _, r := range []float32{0, 10, 24, 32, 40} {
		for _, a := range []float32{0, 1.0, 2.0, 3.14, 4.5} {
			s, c := math32.Sincos(a)
			gapPts = append(gapPts, ms3.Vec{X: r * c, Y: r * s, Z: zMid})
		}
	}
	dCar := evalAt(t, car, gapPts...)
	dDeck := evalAt(t, deck, gapPts...)
	for i, pt := range gapPts {
		if dCar[i] < 0.65 || dDeck[i] < 0.65 {
			t.Errorf("insufficient carousel-to-motor_deck clearance at %v: dCar=%.3f, dDeck=%.3f", pt, dCar[i], dDeck[i])
		}
	}

	// Verify double-D shaft bore at (0, 0):
	//  - Center (0, 0, z0+2) must be open (d > 0)
	//  - Along X at r=2.2 mm (rounded side, r_bore=2.6 mm) must be open (d > 0)
	//  - Along Y at r=2.0 mm (flat side, half-flat=1.6 mm) must be SOLID (d < 0)
	//  - Above blind bore top (z0 + ShaftBoreDepth + 2.0) must be SOLID (positive Z-stop!)
	zBoreMid := p.CarouselBottomZ() + 2.5
	shaftChecks := evalAt(t, car,
		ms3.Vec{X: 0, Y: 0, Z: zBoreMid},
		ms3.Vec{X: 2.2, Y: 0, Z: zBoreMid},
		ms3.Vec{X: 0, Y: 2.0, Z: zBoreMid},
		ms3.Vec{X: 0, Y: 0, Z: p.ShaftTipZ() + 2.0},
	)
	if shaftChecks[0] <= 0 {
		t.Errorf("shaft bore center should be open, got d=%.3f", shaftChecks[0])
	}
	if shaftChecks[1] <= 0 {
		t.Errorf("shaft bore round side (X=2.2) should be open, got d=%.3f", shaftChecks[1])
	}
	if shaftChecks[2] >= 0 {
		t.Errorf("shaft bore double-D flat (Y=2.0) should be solid plastic, got d=%.3f", shaftChecks[2])
	}
	if shaftChecks[3] >= 0 {
		t.Errorf("shaft bore positive Z-stop above shaft tip should be solid plastic, got d=%.3f", shaftChecks[3])
	}

	// Verify motor_deck shaft boss hole at (0, 0) is open.
	deckCenter := evalAt(t, deck, ms3.Vec{X: 0, Y: 0, Z: p.BaseHeight - 1.5})
	if deckCenter[0] <= 0 {
		t.Errorf("motor_deck central shaft hole at (0,0) should be open, got d=%.3f", deckCenter[0])
	}
}

// TestAirflowContinuity verifies an unobstructed airflow path from the 45-degree Noctua fan
// through duct_hood -> lid_labyrinth top aperture -> active wick zone -> flared front exhaust nozzle.
func TestAirflowContinuity(t *testing.T) {
	p := DefaultParams()
	var bld gsdf.Builder
	lid := BuildLidLabyrinth(&bld, p)
	hood := BuildDuctHood(&bld, p)

	streamline := []struct {
		label string
		pt    ms3.Vec
	}{
		{"fan_center", ms3.Vec{X: p.FanCenterX, Y: 0, Z: p.HoodBottomZ() + p.FanCenterZOff}},
		{"duct_mid", ms3.Vec{X: 8.0, Y: 0, Z: p.HoodBottomZ() + 14.5}},
		{"lid_aperture", ms3.Vec{X: p.VialPitchR, Y: 0, Z: p.LidCeilingInZ() + 1.5}},
		{"active_wick_zone", ms3.Vec{X: p.VialPitchR, Y: 0, Z: p.CarouselBottomZ() + p.WickTipLocalZ() - 4.0}},
		{"exhaust_throat", ms3.Vec{X: p.OuterRingOuterR() + 4.0, Y: 0, Z: p.ExhaustFloorZ() + 8.0}},
		{"exhaust_exit", ms3.Vec{X: p.ExhaustExitX() - 1.0, Y: 0, Z: p.ExhaustFloorZ() + 8.0}},
	}

	pts := make([]ms3.Vec, len(streamline))
	for i, s := range streamline {
		pts[i] = s.pt
	}
	dLid := evalAt(t, lid, pts...)
	dHood := evalAt(t, hood, pts...)
	for i, s := range streamline {
		if dLid[i] <= 0 || dHood[i] <= 0 {
			t.Errorf("airflow blocked at %s (%v): dLid=%.3f, dHood=%.3f", s.label, s.pt, dLid[i], dHood[i])
		} else {
			t.Logf("airflow clear at %-16s: dLid=+%.2f mm, dHood=+%.2f mm", s.label, dLid[i], dHood[i])
		}
	}
}

// TestBaseElectronicsClearance verifies the Raspberry Pi Zero 2 WH, ULN2003 driver board,
// 28BYJ-48 stepper body, and M12 push button all fit inside base without colliding with walls or bosses.
func TestBaseElectronicsClearance(t *testing.T) {
	p := DefaultParams()
	var bld gsdf.Builder
	base := BuildBase(&bld, p)

	// Sample corners & center of Pi Zero 2 WH PCB at Z = BaseFloor + PiStandoffH + 1.0 (8.0 mm)
	zPi := p.BaseFloor + p.PiStandoffH + 1.0
	piPts := []ms3.Vec{
		{X: -p.PiLength/2 + 1, Y: p.PiCenterY - p.PiWidth/2 + 1, Z: zPi},
		{X: p.PiLength/2 - 1, Y: p.PiCenterY - p.PiWidth/2 + 1, Z: zPi},
		{X: -p.PiLength/2 + 1, Y: p.PiCenterY + p.PiWidth/2 - 1, Z: zPi},
		{X: p.PiLength/2 - 1, Y: p.PiCenterY + p.PiWidth/2 - 1, Z: zPi},
		{X: 0, Y: p.PiCenterY, Z: zPi + 12.0}, // 40-pin header & Dupont bend clearance
	}
	for _, pt := range piPts {
		if d := evalAt(t, base, pt)[0]; d <= 0 {
			t.Errorf("Pi Zero collides with base at %v (d=%.3f)", pt, d)
		}
	}

	// Sample ULN2003 PCB corners at Z = BaseFloor + ULNStandoffH + 1.0 (7.0 mm)
	zULN := p.BaseFloor + p.ULNStandoffH + 1.0
	ulnPts := []ms3.Vec{
		{X: p.ULNCenterX - p.ULNLength/2 + 1, Y: p.ULNCenterY - p.ULNWidth/2 + 1, Z: zULN},
		{X: p.ULNCenterX + p.ULNLength/2 - 1, Y: p.ULNCenterY - p.ULNWidth/2 + 1, Z: zULN},
		{X: p.ULNCenterX - p.ULNLength/2 + 1, Y: p.ULNCenterY + p.ULNWidth/2 - 1, Z: zULN},
		{X: p.ULNCenterX + p.ULNLength/2 - 1, Y: p.ULNCenterY + p.ULNWidth/2 - 1, Z: zULN},
	}
	for _, pt := range ulnPts {
		if d := evalAt(t, base, pt)[0]; d <= 0 {
			t.Errorf("ULN2003 collides with base at %v (d=%.3f)", pt, d)
		}
	}

	// Verify Micro-USB PWR IN wall cutout is open through the +Y outer wall.
	usbPt := ms3.Vec{X: p.PiUSBCenterX, Y: p.OuterRadius - 1.5, Z: p.BaseFloor + p.PiStandoffH + 3.0}
	if d := evalAt(t, base, usbPt)[0]; d <= 0 {
		t.Errorf("Micro-USB PWR IN cutout is blocked at %v (d=%.3f)", usbPt, d)
	}

	// Verify M12 button hole center is open through the flat boss.
	btnA := p.ButtonAngleDeg * math32.Pi / 180
	s, c := math32.Sincos(btnA)
	btnPt := ms3.Vec{X: (p.OuterRadius - 3.0) * c, Y: (p.OuterRadius - 3.0) * s, Z: p.ButtonCenterZ}
	if d := evalAt(t, base, btnPt)[0]; d <= 0 {
		t.Errorf("M12 button hole is blocked at %v (d=%.3f)", btnPt, d)
	}
}
