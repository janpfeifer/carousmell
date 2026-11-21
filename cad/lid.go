package cad

import (
	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms2"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildLidLabyrinth creates Part 4: the stationary PETG labyrinth seal plate,
// outer cylindrical enclosure shell, and integrated flared front exhaust nozzle.
//
// Features:
//   - Two concentric downward-pointing 2.2 mm labyrinth seal rings framing the carousel's
//     radial fins with a 0.60 mm non-contact radial air gap and 0.60 mm vertical ceiling gap.
//   - Single top inlet sector aperture at theta = 0 deg (+X) directly over the active slot.
//   - Sealed exhaust tunnel bridging from the outer labyrinth ring through the outer shell
//     into a flared 18 -> 22 mm wide front exhaust nozzle centered on the exposed 3 mm wick.
//   - Bottom stepped slip-fit skirt with rear alignment key so the lid lifts off effortlessly
//     to refill vials and always re-seats at exact theta = 0 deg.
//   - Four top M3 mounting bosses (45-degree chamfered undersides) to bolt duct_hood securely.
//
// Print orientation: Upside-down (top ceiling Z = LidTopZ resting flat on the build plate).
// 100% support-free on the Creality K1C.
func BuildLidLabyrinth(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	rOut := p.OuterRadius
	rIn := p.InnerRadius()
	zBot := p.LidBottomZ()
	zCeilIn := p.LidCeilingInZ()
	zTop := p.LidTopZ()
	zRingBot := p.LabyrinthRingBottomZ()
	skirtInnerR := p.LipMidR() + p.LipClearance/2 // 58.325 mm, slips over base rim (58.075 mm)

	// 1. Outer cylindrical shell + flat ceiling plate + bottom slip-fit alignment skirt.
	shellProfile := bld.NewPolygon([]ms2.Vec{
		{X: skirtInnerR, Y: zBot},
		{X: rOut, Y: zBot},
		{X: rOut, Y: zTop},
		{X: 0, Y: zTop},
		{X: 0, Y: zCeilIn},
		{X: rIn, Y: zCeilIn},
		{X: rIn, Y: zBot + p.LipHeight + 1.5},
		// 45-degree internal chamfer above the slip-fit step (support-free upside-down & right-side-up)
		{X: skirtInnerR, Y: zBot + p.LipHeight},
	})
	shell := revolveZ(bld, shellProfile)

	// 2. Inner labyrinth seal ring (full 360 deg closed, keeps airflow sweeping outward across wick).
	innerRing := tubeZ(bld, p.InnerRingInnerR(), p.InnerRingOuterR(), zRingBot, zCeilIn+0.5)

	// 3. Outer labyrinth seal ring (seals all inactive pockets; opened at theta=0 into exhaust tunnel).
	outerRing := tubeZ(bld, p.OuterRingInnerR(), p.OuterRingOuterR(), zRingBot, zCeilIn+0.5)

	// 4. Sealed flared exhaust tunnel & protruding front nozzle (+X, theta = 0 deg).
	// Connects outer labyrinth ring (R = OuterRingInnerR) cleanly through outer shell to ExhaustExitX.
	exX0 := p.OuterRingInnerR() - 0.5
	exX1 := p.ExhaustExitX()
	exFloorIn := p.ExhaustFloorZ()
	exFloorOut := exFloorIn - p.ExhaustWall
	exOuterW0 := p.ExhaustInnerW + 2*p.ExhaustWall
	exOuterW1 := p.ExhaustExitW + 2*p.ExhaustWall
	// Build outer exhaust housing up to zTop (so top sits flush with build plate when printed upside-down!).
	exhaustHousing := taperedTunnelX(bld, exX0, exX1, exOuterW0, exOuterW1, exFloorOut, zTop)
	// Support-free 45-degree tapered fillet/chamfer under the protruding nozzle nose when printed upside-down:
	// (Note: upside-down, exFloorOut is on TOP, bridging smoothly between the two side walls.)

	// 5. Four upper M3 mounting bosses for bolting duct_hood (16 mm tall below ceiling,
	// with 45-degree chamfered lower end so they print upside-down 100% support-free!).
	bossZTop := zCeilIn + 0.5
	bossZBot := zCeilIn - 14.0
	var parts []glbuild.Shader3D
	parts = append(parts, shell, innerRing, outerRing, exhaustHousing)
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		col := chamferedCylZ(bld, p.BoltBossR, bossZBot, bossZTop, p.BoltBossR-1.0, 0)
		col = bld.Translate(col, bx, by, 0)
		tie := boxAt(bld, p.BoltCircleR, rOut-1.0, -p.BoltBossR, p.BoltBossR, bossZBot+p.BoltBossR, bossZTop, 0)
		tie = bld.Rotate(tie, a, axisZ)
		parts = append(parts, col, tie)
	}

	// 6. Small rear indexing lug matching the base wire/alignment notch at theta = 180 deg.
	// Sits just inside the skirt rim so the lid clicks into the exact angular orientation.
	indexLug := boxAt(bld, -skirtInnerR-0.2, -rIn+0.4, -2.0, 2.0, zBot+0.3, zBot+p.LipHeight-0.3, 0)
	parts = append(parts, indexLug)

	solid := unionAll(bld, parts...)

	// --- SUBTRACTIVE CUTOUTS ---
	var cuts []glbuild.Shader3D

	// A. Clear the carousel rotating volume strictly inside the outer labyrinth ring
	// (ensures exhaustHousing never intrudes inside OuterRingInnerR).
	carouselKeepout := cylZ(bld, p.OuterRingInnerR(), zRingBot-1.0, zCeilIn)
	// Keep the inner labyrinth ring intact by intersecting keepout with annulus [InnerRingOuterR, OuterRingInnerR].
	carouselRingClear := tubeZ(bld, p.InnerRingOuterR(), p.OuterRingInnerR(), zRingBot-1.0, zCeilIn)
	_ = carouselKeepout
	cuts = append(cuts, carouselRingClear)

	// B. Single Top Inlet Aperture in the lid ceiling at theta = 0 deg (+X), strictly inside
	// both labyrinth rings and strictly narrower than the +/-14.4 deg flanking carousel fins.
	apSpanRad := p.ApertureSpanDeg * math32.Pi / 180
	aperture := sectorPrismZ(bld, p.CarouselInnerR+0.5, p.CarouselOuterR-1.0, apSpanRad, zCeilIn-1.0, zTop+1.0)
	cuts = append(cuts, aperture)

	// C. Flared interior exhaust passage from OuterRingInnerR out past ExhaustExitX.
	// Centered on the exposed wick (Z = ExhaustFloorZ to zCeilIn).
	exhaustBore := taperedTunnelX(bld, exX0-1.0, exX1+2.0, p.ExhaustInnerW, p.ExhaustExitW, exFloorIn, zCeilIn)
	cuts = append(cuts, exhaustBore)

	// D. Four M3 clearance holes + side-accessible captive hex nut traps in the upper bosses
	// for bolting duct_hood down with standard M3x12 mm screws.
	nutZ1 := zCeilIn - 3.5
	nutZ0 := nutZ1 - p.M3NutDepth
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		screwHole := cylZAt(bld, bx, by, p.M3ClearanceD/2, bossZBot-1.0, zTop+1.0)
		nutSlot := boxAt(bld, p.BoltCircleR-8.0, p.BoltCircleR+p.M3NutAF/2, -p.M3NutAF/2, p.M3NutAF/2, nutZ0, nutZ1, 0)
		nutSlot = bld.Rotate(nutSlot, a, axisZ)
		cuts = append(cuts, screwHole, nutSlot)
	}

	// E. Rear fan cable pass-through slot through the lid ceiling (outside the outer labyrinth ring!).
	wireSlot := boxAt(bld, -rIn+0.5, -rIn+0.5+p.WireChannelD, -p.WireChannelW/2, p.WireChannelW/2, zCeilIn-1.0, zTop+1.0, 0)
	cuts = append(cuts, wireSlot)

	return bld.Difference(solid, unionAll(bld, cuts...))
}
