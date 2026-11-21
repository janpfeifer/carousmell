package cad

import (
	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// BuildDuctHood creates Part 5: the upstream 45-degree 40mm Noctua NF-A4x10 fan mount
// and smooth curved turning duct.
//
// Design features:
//   - 45-degree tilted fan mounting face (prints 100% support-free on Creality K1C!).
//   - Open rear intake cradle so the Noctua fan draws whisper-quiet unobstructed room air
//     and gives full finger access behind the plate to pull the silicone anti-vibration pins through.
//   - Smooth cubic-Bezier lofted turning duct transitioning from the Ø38 mm circular fan bore
//     down into the single lid aperture at (+32, 0), with uniform 2.6 mm PETG wall thickness.
//   - Bolts directly onto lid_labyrinth with 4x M3x12 mm perimeter screws so lid + hood lift
//     off together as a single top assembly when refilling vials.
//
// Print orientation: Right-side-up on Z_local = 0 (Z_global = LidTopZ). 100% support-free.
func BuildDuctHood(bld *gsdf.Builder, p Params) glbuild.Shader3D {
	z0 := p.HoodBottomZ()
	rOut := p.OuterRadius
	flangeH := float32(3.2)
	tiltRad := p.FanTiltDeg * math32.Pi / 180
	sinT, cosT := math32.Sincos(tiltRad)

	// Center of the fan mounting plate (front/duct-facing side) in global coords.
	fanCX := p.FanCenterX
	fanCZ := z0 + p.FanCenterZOff

	// --- 1. Build the smooth curved inner airflow cavity along a cubic Bezier ---
	// P0: fan bore center on front of plate, tangent = (+cosT, 0, -sinT)
	// P3: active lid aperture center at (+VialPitchR, 0, z0), descending smoothly across the wick.
	p0 := ms3.Vec{X: fanCX, Y: 0, Z: fanCZ}
	c1 := ms3.Vec{X: fanCX + 18.0*cosT, Y: 0, Z: fanCZ - 18.0*sinT}
	c2 := ms3.Vec{X: p.VialPitchR - 6.0, Y: 0, Z: z0 + 18.5}
	p3 := ms3.Vec{X: p.VialPitchR - 0.25, Y: 0, Z: z0 + 1.0}

	// Dense, overlapping spheres along the cubic Bezier create a ripple-free smooth tapered tube!
	const steps = 22
	rStart := p.FanBoreD / 2
	rEnd := float32(7.0)
	segs := make([]glbuild.Shader3D, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		pt := cubicBezier3(p0, c1, c2, p3, t)
		rad := rStart + (rEnd-rStart)*math32.Pow(t, 1.15)
		s := bld.Translate(bld.NewSphere(rad), pt.X, pt.Y, pt.Z)
		segs = append(segs, s)
	}
	// Union of dense overlapping spheres has C1 smoothness and evaluates very cleanly on CPU.
	innerTube := bld.Union(segs...)

	// Clip the rear side of the first sphere cleanly behind the fan mounting plate so the cavity
	// meets the flat fan mounting face sharply without bulging backward.
	fanRotY := math32.Pi/2 + tiltRad
	frontHalf := bld.NewBox(160.0, 120.0, 120.0, 0)
	frontHalf = bld.Translate(frontHalf, 0, 0, 60.0-p.FanPlateThick-0.5)
	frontHalf = bld.Rotate(frontHalf, fanRotY, axisY)
	frontHalf = bld.Translate(frontHalf, fanCX, 0, fanCZ)
	innerTube = bld.Intersection(innerTube, frontHalf)

	// Transition collar right into the lid's sector aperture so there is zero step at z = z0.
	apSpanRad := p.ApertureSpanDeg * math32.Pi / 180
	apChute := sectorPrismZ(bld, p.CarouselInnerR+0.8, p.CarouselOuterR-1.3, apSpanRad, z0-2.0, z0+5.0)
	innerCavity := bld.SmoothUnion(3.0, innerTube, apChute)

	// --- 2. Outer Duct Shell via Exact SDF Offset ---
	// Offset by -DuctWall creates a uniform 2.6 mm wall everywhere around the smooth curved duct!
	outerDuct := bld.Offset(innerCavity, -p.DuctWall)

	// --- 3. 45-Degree Tilted Noctua 40mm Fan Mounting Plate & Support Gussets ---
	// Local plate lies in XY plane (44 x 44 x FanPlateThick), normal along +Z.
	// Rotate by +(pi/2 + tiltRad) around Y so local +Z points along (+cosT, 0, -sinT) (airflow dir).
	plateW := p.FanSize + 4.0 // 44 mm
	fanPlate := bld.NewBox(plateW, plateW, p.FanPlateThick, min(1.0, p.FanPlateThick/3))
	// Shift so front face sits at z_local = 0 and plate thickness extends toward rear (-z_local).
	fanPlate = bld.Translate(fanPlate, 0, 0, -p.FanPlateThick/2)
	fanPlate = bld.Rotate(fanPlate, fanRotY, axisY)
	fanPlate = bld.Translate(fanPlate, fanCX, 0, fanCZ)

	// Two sturdy 45-degree side support struts tying the tilted fan plate solidly down to the base flange.
	// Sits at Y = ±20.5 mm (just outside the 40 mm fan width) — zero overhang, zero airflow obstruction!
	strutL := boxAt(bld, fanCX-15.0, fanCX+12.0, 19.5, 22.5, z0, fanCZ+6.0, 0)
	strutR := boxAt(bld, fanCX-15.0, fanCX+12.0, -22.5, -19.5, z0, fanCZ+6.0, 0)
	strutTrim := bld.NewBox(80.0, 60.0, 80.0, 0)
	strutTrim = bld.Translate(strutTrim, 0, 0, 40.0-p.FanPlateThick)
	strutTrim = bld.Rotate(strutTrim, fanRotY, axisY)
	strutTrim = bld.Translate(strutTrim, fanCX, 0, fanCZ)
	struts := bld.Intersection(bld.Union(strutL, strutR), strutTrim)

	// --- 4. Base Mounting Flange & Sleek Front Perimeter Trim Ring ---
	baseFlange := cylZ(bld, rOut, z0, z0+flangeH)

	// Low-profile outer styling rim around the front and sides (height 14 mm, open at rear
	// behind X < -16 mm so the Noctua fan has unobstructed room-air intake & silicone-pin access!).
	rimRing := tubeZ(bld, rOut-p.ShellWall, rOut, z0, z0+14.0)
	rimFrontClip := boxAt(bld, -16.0, rOut+2.0, -rOut-2.0, rOut+2.0, z0, z0+15.0, 0)
	frontRim := bld.Intersection(rimRing, rimFrontClip)

	// Combine all positive geometry and slice cleanly at z >= z0 for a flat K1C print bed face.
	posAll := bld.Union(baseFlange, frontRim, outerDuct, fanPlate, struts)
	bedClip := boxAt(bld, -rOut-2.0, rOut+2.0, -rOut-2.0, rOut+2.0, z0, z0+p.HoodHeight+5.0, 0)
	solid := bld.Intersection(posAll, bedClip)

	// --- 5. Subtractive Cutouts ---
	var cuts []glbuild.Shader3D

	// A. Smooth curved inner airflow cavity.
	cuts = append(cuts, innerCavity)

	// B. Circular Ø38 mm fan intake bore & 4 corner mounting holes (Ø4.4 mm at 32x32 mm)
	// punched cleanly through the tilted fan plate (short depth so they don't nick the outer duct wall).
	fanBore := bld.Translate(bld.NewCylinder(p.FanBoreD/2, p.FanPlateThick+6.0, 0), 0, 0, -p.FanPlateThick/2)
	fanHoleHalf := p.FanHoleSpan / 2
	var fanHoles []glbuild.Shader3D
	fanHoles = append(fanHoles, fanBore)
	for _, sx := range []float32{-1, 1} {
		for _, sy := range []float32{-1, 1} {
			h := bld.NewCylinder(p.FanHoleD/2, p.FanPlateThick+3.0, 0)
			h = bld.Translate(h, sx*fanHoleHalf, sy*fanHoleHalf, -p.FanPlateThick/2)
			fanHoles = append(fanHoles, h)
		}
	}
	fanCutGroup := bld.Rotate(unionAll(bld, fanHoles...), fanRotY, axisY)
	fanCutGroup = bld.Translate(fanCutGroup, fanCX, 0, fanCZ)
	cuts = append(cuts, fanCutGroup)

	// C. Clear rear fan installation space behind the fan mounting plate (so duct outer wall
	// never bumps the rear face of the 40x40x10 mm Noctua fan!).
	rearClear := bld.NewBox(p.FanSize+1.0, p.FanSize+1.0, p.FanThick+16.0, 0)
	rearClear = bld.Translate(rearClear, 0, 0, -p.FanPlateThick-(p.FanThick+16.0)/2)
	rearClear = bld.Rotate(rearClear, fanRotY, axisY)
	rearClear = bld.Translate(rearClear, fanCX, 0, fanCZ)
	cuts = append(cuts, rearClear)

	// D. Four perimeter M3 clearance holes matching lid_labyrinth upper bosses.
	for _, a := range p.BoltAnglesRad() {
		s, c := math32.Sincos(a)
		bx, by := p.BoltCircleR*c, p.BoltCircleR*s
		cuts = append(cuts, cylZAt(bld, bx, by, p.M3ClearanceD/2, z0-1.0, z0+20.0))
	}

	// E. Rear Noctua 4-pin wire drop slot aligned with lid_labyrinth's wire pass-through.
	rIn := p.InnerRadius()
	wireSlot := boxAt(bld, -rIn+0.5, -rIn+0.5+p.WireChannelD, -p.WireChannelW/2, p.WireChannelW/2, z0-1.0, z0+flangeH+1.0, 0)
	cuts = append(cuts, wireSlot)

	return bld.Difference(solid, unionAll(bld, cuts...))
}

func cubicBezier3(p0, p1, p2, p3 ms3.Vec, t float32) ms3.Vec {
	u := 1 - t
	b0 := u * u * u
	b1 := 3 * u * u * t
	b2 := 3 * u * t * t
	b3 := t * t * t
	return ms3.Vec{
		X: b0*p0.X + b1*p1.X + b2*p2.X + b3*p3.X,
		Y: b0*p0.Y + b1*p1.Y + b2*p2.Y + b3*p3.Y,
		Z: b0*p0.Z + b1*p1.Z + b2*p2.Z + b3*p3.Z,
	}
}
