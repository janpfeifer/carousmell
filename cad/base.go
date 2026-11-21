package cad

import (
	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms2"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildBase creates Part 1: the Hyper PLA electronics housing tub.
//
// Print orientation: Right-side-up on Z = 0. Open top, 45-degree chamfered internal ledge,
// teardrop button hole, and arch-chamfered USB cutout make it 100% support-free on the Creality K1C.
func BuildBase(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	rOut := p.OuterRadius
	rIn := p.InnerRadius()
	zTop := p.BaseHeight
	zDeck := p.MotorDeckBottomZ()
	lipR := p.LipMidR() - p.LipClearance/2 // Outer radius of base alignment rim (58.075 mm)

	// 1. Main outer cylinder with bottom anti-elephant-foot chamfer + upper alignment rim.
	outerProfile := bld.NewPolygon([]ms2.Vec{
		{X: 0, Y: 0},
		{X: rOut - p.BottomChamfer, Y: 0},
		{X: rOut, Y: p.BottomChamfer},
		{X: rOut, Y: zTop},
		{X: lipR, Y: zTop},
		{X: lipR, Y: zTop + p.LipHeight},
		{X: rIn, Y: zTop + p.LipHeight},
		{X: rIn, Y: zDeck},
		// 45-degree support-free chamfered internal support ledge for motor_deck:
		{X: rIn - 2.6, Y: zDeck},
		{X: rIn - 2.6, Y: zDeck - 2.0},
		{X: rIn, Y: zDeck - 4.6},
		{X: rIn, Y: p.BaseFloor},
		{X: 0, Y: p.BaseFloor},
	})
	body := revolveZ(bld, outerProfile)

	// Indexing alignment tab on the rear (-X) of the base rim so lid_labyrinth always seats at theta=0.
	keyTab := boxAt(bld, -lipR-0.2, -rIn+0.2, -2.5, 2.5, zTop, zTop+p.LipHeight, 0)
	// Actually key notch is cut from rim and matched by lid; let's cut a notch on the rim at theta=180.
	_ = keyTab

	// 2. Flat exterior boss for the 12 mm ATX "Whiff Now" push button at ButtonAngleDeg.
	btnAngle := p.ButtonAngleDeg * math32.Pi / 180
	// Build button flat pad in +X frame, then rotate to btnAngle.
	// Outer face sits at x = rOut - 1.5 so it forms a crisp flat mounting plane.
	btnFlatX := rOut - 1.5
	btnBoss := boxAt(bld, btnFlatX-p.ButtonWall-2.5, btnFlatX, -11.0, 11.0, p.ButtonCenterZ-12.0, p.ButtonCenterZ+12.0, 0)
	btnBoss = bld.Rotate(btnBoss, btnAngle, axisZ)

	// 3. Four vertical perimeter M3 mounting columns (support-free from floor to zDeck).
	var bosses []glbuild.Shader3D
	bosses = append(bosses, body, btnBoss)
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		col := cylZAt(bld, bx, by, p.BoltBossR, p.BaseFloor-0.5, zDeck)
		// Radial bridge tying boss column solidly into outer wall.
		tie := boxAt(bld, p.BoltCircleR, rOut-1.0, -p.BoltBossR, p.BoltBossR, p.BaseFloor-0.5, zDeck, 0)
		tie = bld.Rotate(tie, a, axisZ)
		bosses = append(bosses, col, tie)
	}

	// 4. Raspberry Pi Zero 2 WH standoffs (4x M2.5 standoffs + subtle alignment edge guides).
	piHX := p.PiHoleSpanX / 2
	piHY := p.PiHoleSpanY / 2
	piZTop := p.BaseFloor + p.PiStandoffH
	for _, sx := range []float32{-1, 1} {
		for _, sy := range []float32{-1, 1} {
			x := sx * piHX
			y := p.PiCenterY + sy*piHY
			// Standoff pillar + alignment pin (M2.5 hole is Ø2.75 mm; use Ø2.2 mm pin or self-tap).
			pillar := cylZAt(bld, x, y, 2.7, p.BaseFloor-0.5, piZTop)
			pin := cylZAt(bld, x, y, 1.1, piZTop-0.2, piZTop+2.2)
			bosses = append(bosses, pillar, pin)
		}
	}
	// Gentle PCB retention rails along the Pi's inner (-Y) and side edges.
	piGuideIn := boxAt(bld, -16.0, 16.0, p.PiCenterY-p.PiWidth/2-2.0, p.PiCenterY-p.PiWidth/2-0.3, p.BaseFloor-0.5, piZTop+2.0, 0)
	bosses = append(bosses, piGuideIn)

	// 5. ULN2003 driver board standoffs.
	ulnHX := p.ULNHoleSpanX / 2
	ulnHY := p.ULNHoleSpanY / 2
	ulnZTop := p.BaseFloor + p.ULNStandoffH
	for _, sx := range []float32{-1, 1} {
		for _, sy := range []float32{-1, 1} {
			x := p.ULNCenterX + sx*ulnHX
			y := p.ULNCenterY + sy*ulnHY
			pillar := cylZAt(bld, x, y, 2.8, p.BaseFloor-0.5, ulnZTop)
			bosses = append(bosses, pillar)
		}
	}

	solid := unionAll(bld, bosses...)

	// --- SUBTRACTIVE CUTOUTS ---
	var cuts []glbuild.Shader3D

	// A. Flatten the exterior of the cylinder at the button face + clear the interior nut space.
	btnExtCut := boxAt(bld, btnFlatX, rOut+5.0, -11.5, 11.5, p.ButtonCenterZ-12.0, p.ButtonCenterZ+12.0, 0)
	btnIntCut := boxAt(bld, btnFlatX-p.ButtonWall-14.0, btnFlatX-p.ButtonWall, -10.5, 10.5, p.ButtonCenterZ-11.0, p.ButtonCenterZ+11.0, 0)
	btnHole := horizontalTeardropHole(bld, p.ButtonHoleD/2, 16.0)
	btnHole = bld.Translate(btnHole, btnFlatX-p.ButtonWall/2, 0, p.ButtonCenterZ)
	btnCuts := bld.Rotate(bld.Union(btnExtCut, btnIntCut, btnHole), btnAngle, axisZ)
	cuts = append(cuts, btnCuts)

	// B. Four perimeter M3 screw holes + side-accessible captive hex nut traps (support-free!).
	nutZ0 := zDeck - 7.5
	nutZ1 := nutZ0 + p.M3NutDepth
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		screwHole := cylZAt(bld, bx, by, p.M3ClearanceD/2, zDeck-12.0, zDeck+1.0)
		// Side-inserted M3 captive hex nut slot opening radially inward.
		nutSlot := boxAt(bld, p.BoltCircleR-8.0, p.BoltCircleR+p.M3NutAF/2, -p.M3NutAF/2, p.M3NutAF/2, nutZ0, nutZ1, 0)
		nutSlot = bld.Rotate(nutSlot, a, axisZ)
		cuts = append(cuts, screwHole, nutSlot)
	}

	// C. ULN2003 pilot holes for M2.5/M3 screws.
	for _, sx := range []float32{-1, 1} {
		for _, sy := range []float32{-1, 1} {
			x := p.ULNCenterX + sx*ulnHX
			y := p.ULNCenterY + sy*ulnHY
			cuts = append(cuts, cylZAt(bld, x, y, 1.3, p.BaseFloor-0.2, ulnZTop+1.0))
		}
	}

	// D. Raspberry Pi Micro-USB PWR IN wall cutout (+Y side, centered at X = PiUSBCenterX).
	// Generous chamfered-arch tunnel accommodates thick molded Micro-USB plug boots.
	usbZCenter := piZTop + 3.0 // PCB top is piZTop+1.5; USB center sits ~1.5 mm above PCB top.
	usb2D := chamferedArchCutout2D(bld, p.PiUSBWidth, p.PiUSBHeight, 3.0)
	usbExt := bld.Extrude(usb2D, 22.0) // Along Z
	// Rotate so extrusion runs along +Y and arch peak points +Z: rotate +pi/2 around X -> (x, -z, y).
	usbCut := bld.Rotate(usbExt, math32.Pi/2, axisX)
	usbCut = bld.Translate(usbCut, p.PiUSBCenterX, rOut-6.0, usbZCenter)
	cuts = append(cuts, usbCut)

	// E. Passive ventilation slots on the lower rear-left and rear-right walls.
	for _, deg := range []float32{115, 160, -160, -115} {
		a := deg * math32.Pi / 180
		vent2D := chamferedArchCutout2D(bld, 12.0, 3.0, 1.0)
		ventExt := bld.Extrude(vent2D, 14.0)
		vent := bld.Rotate(ventExt, math32.Pi/2, axisX)
		vent = bld.Rotate(vent, a-math32.Pi/2, axisZ)
		s, c := math32.Sincos(a)
		vent = bld.Translate(vent, (rOut-1.5)*c, (rOut-1.5)*s, 8.0)
		cuts = append(cuts, vent)
	}

	// F. Vertical Noctua fan wire routing channel on the rear inner wall (-X, theta=180 deg)
	// plus matching alignment key notch on the upper lip.
	wireCh := boxAt(bld, -rIn-1.5, -rIn+p.WireChannelD, -p.WireChannelW/2, p.WireChannelW/2, p.BaseFloor+4.0, zTop+p.LipHeight+1.0, 0)
	cuts = append(cuts, wireCh)

	return bld.Difference(solid, unionAll(bld, cuts...))
}
