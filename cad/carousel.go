package cad

import (
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildCarousel creates Part 3: the 11-slot PETG rotary carousel.
//
// Features:
//   - Direct-drive central hub at (0, 0) with blind double-D bore matching the 28BYJ-48 shaft
//     (positive Z-stop on shaft tip sets exact 1.5 mm clearance above motor_deck).
//   - 10 drop-in cylindrical vial pockets (Slots 1-10) with >=2.2 mm PETG walls.
//   - Slot 0 (theta = 0 deg): solid park slot with visual dimple on top and 5x2 mm homing magnet
//     pocket recessed into the underside.
//   - 11 tall radial divider fins placed midway between adjacent slots, extending 2.5 mm above
//     the exposed wick tips, with an open outer radial edge for cross-wick airflow.
//
// Print orientation: Right-side-up on Z_local = 0. 100% support-free on the Creality K1C.
func BuildCarousel(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	z0 := p.CarouselBottomZ()
	zFloorTop := z0 + p.CarouselFloor
	zCupTop := z0 + p.CarouselCupTopLocalZ()
	zFinTop := p.CarouselFinTopZ()
	dTheta := p.SlotAngleRad()

	// 1. Base circular floor disc (solid vapor barrier + hub-to-ring structure).
	baseDisc := chamferedCylZ(bld, p.CarouselOuterR, z0, zFloorTop, 0.5, 0)

	// 2. Central direct-drive hub boss.
	hub := chamferedCylZ(bld, p.CarouselHubOD/2, z0, z0+p.CarouselHubH, 0.5, 0.6)

	// 3. Annular lower vial cup ring (Z = zFloorTop to zCupTop).
	cupRing := tubeZ(bld, p.CarouselInnerR, p.CarouselOuterR, zFloorTop-0.2, zCupTop)

	// 4. Eleven tall radial divider fins placed midway between adjacent slots (at theta_k + dTheta/2).
	// Using CircularArray evaluates only 2 instances per point for fast meshing & raymarching.
	fin := boxAt(bld, p.CarouselInnerR, p.CarouselOuterR, -p.FinThickness/2, p.FinThickness/2, zCupTop-0.5, zFinTop, 0)
	fins := bld.CircularArray(fin, p.NumSlots, p.NumSlots)
	fins = bld.Rotate(fins, dTheta/2, axisZ)

	solid := bld.Union(baseDisc, hub, cupRing, fins)

	// --- SUBTRACTIVE CUTOUTS ---
	var cuts []glbuild.Shader3D

	// A. Central blind double-D shaft bore (positive Z-stop at z0 + ShaftBoreDepth).
	shaftD := p.StepperShaftD + p.ShaftClearance
	shaftFlats := p.StepperShaftFlats + p.ShaftClearance
	boreTop := z0 + p.ShaftBoreDepth()
	bore := doubleDBore(bld, shaftD, shaftFlats, z0-1.0, boreTop)
	// Small bottom lead-in chamfer (0.5 mm) for smooth mounting onto the stepper shaft.
	boreChamfer := cylZ(bld, shaftD/2+0.4, z0-0.5, z0+0.5)
	cuts = append(cuts, bore, boreChamfer)

	// B. Ten drop-in 2 ml vial pockets for Slots 1..10 (leaving Slot 0 at theta=0 solid!).
	// CircularArray(pocket, 10, 11) puts instances at k = 0..9; rotating by +dTheta shifts to k = 1..10.
	vialPocket := cylZAt(bld, p.VialPitchR, 0, p.VialPocketD()/2, zFloorTop, zCupTop+1.0)
	vialPockets := bld.CircularArray(vialPocket, p.NumSlots-1, p.NumSlots)
	vialPockets = bld.Rotate(vialPockets, dTheta, axisZ)
	cuts = append(cuts, vialPockets)

	// C. Slot 0 (Park Slot, theta = 0 deg):
	//    - Underside recess for the Ø5 x 2 mm neodymium homing magnet.
	magR := (p.MagnetD + p.MagnetClearD) / 2
	magH := p.MagnetH + p.MagnetClearH
	magnetPocket := cylZAt(bld, p.VialPitchR, 0, magR, z0-0.5, z0+magH)
	//    - Shallow visual marker dimple on top of Slot 0 to identify the park slot during refilling.
	parkDimple := cylZAt(bld, p.VialPitchR, 0, 3.5, zCupTop-1.5, zCupTop+1.0)
	cuts = append(cuts, magnetPocket, parkDimple)

	return bld.Difference(solid, unionAll(bld, cuts...))
}
