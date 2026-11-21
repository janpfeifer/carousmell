package cad

import (
	"github.com/chewxy/math32"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildMotorDeck creates Part 2: the removable stepper & Hall-sensor divider plate.
//
// It sits inside the top ledge of base (Z = 39.0..42.0 mm), seals the electronics bay
// from essential oil vapors, mounts the 28BYJ-48 stepper motor with its shaft at exact (0,0),
// and holds the KY-024 Hall-effect sensor directly underneath Slot 0 (+32.0, 0).
//
// Print orientation: Upside-down (Z_global = 42.0 mm flat on bed) so the Hall sensor
// retaining rails print straight up — 100% support-free on the Creality K1C.
func BuildMotorDeck(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	z0 := p.MotorDeckBottomZ()
	z1 := p.BaseHeight
	deckR := p.InnerRadius() - p.MotorDeckGap

	// 1. Main circular divider plate.
	plate := cylZ(bld, deckR, z0, z1)

	// 2. KY-024 Hall-Effect Sensor cradle on underside centered at Slot 0 (+VialPitchR, 0).
	// The KY-024 PCB (35 x 15 mm) sits tangentially along Y with the Hall IC tip at (+32, 0).
	// Two clamping walls + an M3 mounting boss hang downward from z0 (print upward when flipped).
	hallX := p.VialPitchR
	hallRailL := boxAt(bld, hallX-9.5, hallX-7.7, -18.0, 4.0, z0-4.5, z0+0.2, 0)
	hallRailR := boxAt(bld, hallX+7.7, hallX+9.5, -18.0, 4.0, z0-4.5, z0+0.2, 0)
	hallStop := boxAt(bld, hallX-9.5, hallX+9.5, 4.0, 6.0, z0-4.5, z0+0.2, 0)
	hallBoss := cylZAt(bld, hallX, -14.0, 3.5, z0-5.0, z0+0.2)

	// 3. Subtle alignment ribs around the stepper motor mounting flange on underside.
	// Stepper body center is at (0, StepperShaftOffsetY = -8.0).
	solid := bld.Union(plate, hallRailL, hallRailR, hallStop, hallBoss)

	// --- SUBTRACTIVE CUTOUTS ---
	var cuts []glbuild.Shader3D

	// A. Central stepper shaft-boss clearance bore at exact (0, 0).
	cuts = append(cuts, cylZ(bld, p.StepperBossHoleD/2, z0-1.0, z1+1.0))

	// B. Stepper motor M3 mounting holes at (±17.5, -8.0) with top counterbore so M3 screw
	// heads sit flush below Z = 42.0 mm (zero interference with the carousel above!).
	earX := p.StepperEarSpacing / 2
	earY := p.StepperShaftOffsetY
	for _, sx := range []float32{-1, 1} {
		// Through-hole for M3 screw
		cuts = append(cuts, cylZAt(bld, sx*earX, earY, p.M3ClearanceD/2, z0-1.0, z1+1.0))
		// Shallow top counterbore (Ø6.0 mm, 1.4 mm deep) for button-head / countersunk M3 screw
		cuts = append(cuts, cylZAt(bld, sx*earX, earY, 3.1, z1-1.4, z1+1.0))
	}

	// C. Four perimeter M3 mounting holes matching base bosses (with top counterbore so
	// screw heads sit below the bottom of lid_labyrinth).
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		cuts = append(cuts, cylZAt(bld, bx, by, p.M3ClearanceD/2, z0-1.0, z1+1.0))
		cuts = append(cuts, cylZAt(bld, bx, by, 3.1, z1-1.5, z1+1.0))
	}

	// D. Hall-effect sensor sensing pocket recessed from underside at (+VialPitchR, 0),
	// leaving a thin 0.8 mm non-magnetic plastic diaphragm so the Hall IC sits within ~2 mm of the magnet
	// while preserving the electronics vapor seal.
	hallPocket := boxAt(bld,
		hallX-p.HallWindowSize/2, hallX+p.HallWindowSize/2,
		-p.HallWindowSize/2, p.HallWindowSize/2,
		z0-1.0, z1-p.HallWindowWall, 0,
	)
	cuts = append(cuts, hallPocket)

	// E. KY-024 M3 self-tapping pilot hole in hallBoss.
	cuts = append(cuts, cylZAt(bld, hallX, -14.0, 1.4, z0-5.5, z0+0.5))

	// F. Rear perimeter notch for the 4-pin Noctua fan cable (-X, theta=180 deg).
	wireNotch := boxAt(bld, -deckR-2.0, -deckR+p.WireChannelD+1.0, -p.WireChannelW/2, p.WireChannelW/2, z0-1.0, z1+1.0, 0)
	cuts = append(cuts, wireNotch)

	return bld.Difference(solid, unionAll(bld, cuts...))
}
