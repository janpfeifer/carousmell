package cad

import (
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildCalibrationCoupon creates Part 6: a 10-minute Creality K1C fit-check test plate
// generated from the exact same Params struct.
//
// Features (left to right on a 66 x 32 x 9 mm plate):
//  1. Three 28BYJ-48 double-D shaft bore test holes (ShaftClearance -0.10, +0.00, +0.10 mm)
//     with 1, 2, and 3 small tick notches along the front edge so you can tell them apart.
//  2. One 2 ml amber glass vial pocket (Ø12.6 mm).
//  3. One 5 x 2 mm neodymium magnet recess (Ø5.2 x 2.2 mm deep).
//  4. One M3 clearance hole (Ø3.4 mm) with underside captive M3 hex nut trap (5.8 mm AF x 2.6 mm).
//  5. One M12 push-button mounting hole (Ø12.3 mm).
func BuildCalibrationCoupon(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	const (
		plateX = float32(66.0)
		plateY = float32(32.0)
		plateZ = float32(9.0)
	)
	plate := boxAt(bld, -plateX/2, plateX/2, -plateY/2, plateY/2, 0, plateZ, 1.0)
	// Slice bottom completely flat at Z = 0 for crisp K1C bed adhesion.
	plate = bld.Intersection(plate, boxAt(bld, -plateX, plateX, -plateY, plateY, 0, plateZ+1.0, 0))

	var cuts []glbuild.Shader3D

	// 1. Three double-D shaft bore clearance tests: -0.10, 0.00, +0.10 mm relative to p.ShaftClearance.
	deltas := []float32{-0.10, 0.00, +0.10}
	xs := []float32{-24.0, -13.0, -2.0}
	for i, d := range deltas {
		clr := max(float32(0.05), p.ShaftClearance+d)
		bore := doubleDBore(bld, p.StepperShaftD+clr, p.StepperShaftFlats+clr, -1.0, plateZ+1.0)
		bore = bld.Translate(bore, xs[i], 6.5, 0)
		cuts = append(cuts, bore)

		// Small front edge tick marks (1, 2, or 3 notches) to identify each tolerance after printing.
		for k := 0; k <= i; k++ {
			tx := xs[i] + (float32(k)-float32(i)/2)*2.4
			tick := boxAt(bld, tx-0.6, tx+0.6, plateY/2-2.0, plateY/2+1.0, plateZ-2.0, plateZ+1.0, 0)
			cuts = append(cuts, tick)
		}
	}

	// 2. 2 ml amber vial pocket (Ø12.6 mm, 7 mm deep blind pocket).
	cuts = append(cuts, cylZAt(bld, -19.0, -7.0, p.VialPocketD()/2, 2.0, plateZ+1.0))

	// 3. 5 x 2 mm magnet recess (Ø5.2 mm, 2.2 mm deep from top).
	magR := (p.MagnetD + p.MagnetClearD) / 2
	magH := p.MagnetH + p.MagnetClearH
	cuts = append(cuts, cylZAt(bld, -2.0, -7.0, magR, plateZ-magH, plateZ+1.0))

	// 4. M3 clearance hole + captive hex nut trap on underside.
	cuts = append(cuts,
		cylZAt(bld, 10.0, 6.5, p.M3ClearanceD/2, -1.0, plateZ+1.0),
		hexNutTrapZ(bld, 10.0, 6.5, p.M3NutAF, -0.5, p.M3NutDepth),
	)

	// 5. M12 ATX push-button mounting hole (Ø12.3 mm).
	cuts = append(cuts, cylZAt(bld, 22.0, -2.0, p.ButtonHoleD/2, -1.0, plateZ+1.0))

	return bld.Difference(plate, unionAll(bld, cuts...))
}
