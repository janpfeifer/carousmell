package cad

import (
	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms2"
	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

var (
	axisX = ms3.Vec{X: 1}
	axisY = ms3.Vec{Y: 1}
	axisZ = ms3.Vec{Z: 1}
)

// cylZ creates a vertical cylinder along Z spanning [z0, z1] with radius r.
func cylZ(bld *gsdf.Builder, r, z0, z1 float32) glbuild.Shader3D {
	h := z1 - z0
	c := bld.NewCylinder(r, h, 0)
	return bld.Translate(c, 0, 0, (z0+z1)/2)
}

// cylZAt creates a vertical cylinder centered at (x, y) spanning [z0, z1] with radius r.
func cylZAt(bld *gsdf.Builder, x, y, r, z0, z1 float32) glbuild.Shader3D {
	h := z1 - z0
	c := bld.NewCylinder(r, h, 0)
	return bld.Translate(c, x, y, (z0+z1)/2)
}

// tubeZ creates a hollow vertical cylinder (annulus) spanning [z0, z1] from rIn to rOut.
func tubeZ(bld *gsdf.Builder, rIn, rOut, z0, z1 float32) glbuild.Shader3D {
	outer := bld.NewCircle(rOut)
	inner := bld.NewCircle(rIn)
	ring2D := bld.Difference2D(outer, inner)
	ext := bld.Extrude(ring2D, z1-z0)
	return bld.Translate(ext, 0, 0, (z0+z1)/2)
}

// boxAt creates an axis-aligned box spanning [x0,x1] x [y0,y1] x [z0,z1] with optional edge rounding.
func boxAt(bld *gsdf.Builder, x0, x1, y0, y1, z0, z1, round float32) glbuild.Shader3D {
	b := bld.NewBox(x1-x0, y1-y0, z1-z0, round)
	return bld.Translate(b, (x0+x1)/2, (y0+y1)/2, (z0+z1)/2)
}

// revolveZ revolves a 2D profile defined in (R >= 0, Z) coordinates around the Z-axis.
// Note: gsdf.Revolve revolves around the Y-axis; rotating +pi/2 around X maps (r, z) -> (R, Z).
func revolveZ(bld *gsdf.Builder, profileRZ glbuild.Shader2D) glbuild.Shader3D {
	rev := bld.Revolve(profileRZ, 0)
	return bld.Rotate(rev, math32.Pi/2, axisX)
}

// chamferedCylZ creates a vertical cylinder spanning [z0, z1] with optional 45-degree
// chamfers on bottom (cBot) and top (cTop) outer edges.
func chamferedCylZ(bld *gsdf.Builder, r, z0, z1, cBot, cTop float32) glbuild.Shader3D {
	verts := []ms2.Vec{
		{X: 0, Y: z0},
		{X: r - cBot, Y: z0},
	}
	if cBot > 0 {
		verts = append(verts, ms2.Vec{X: r, Y: z0 + cBot})
	}
	if cTop > 0 {
		verts = append(verts, ms2.Vec{X: r, Y: z1 - cTop})
	}
	verts = append(verts,
		ms2.Vec{X: r - cTop, Y: z1},
		ms2.Vec{X: 0, Y: z1},
	)
	return revolveZ(bld, bld.NewPolygon(verts))
}

// doubleDBore creates a vertical double-D prism matching the 28BYJ-48 stepper shaft,
// centered at (0, 0) spanning [z0, z1]. Flats are parallel to the X-axis (normal along ±Y).
func doubleDBore(bld *gsdf.Builder, diameter, acrossFlats, z0, z1 float32) glbuild.Shader3D {
	circle := bld.NewCircle(diameter / 2)
	slab := bld.NewRectangle(diameter+2.0, acrossFlats, 0)
	profile := bld.Intersection2D(circle, slab)
	ext := bld.Extrude(profile, z1-z0)
	return bld.Translate(ext, 0, 0, (z0+z1)/2)
}

// hexNutTrapZ creates a vertical hexagonal nut trap centered at (x, y) spanning [z0, z1]
// with face-to-face width acrossFlats.
// Note: gsdf.NewHexagonalPrism takes apothem (half across-flats) and half-height.
func hexNutTrapZ(bld *gsdf.Builder, x, y, acrossFlats, z0, z1 float32) glbuild.Shader3D {
	h := bld.NewHexagonalPrism(acrossFlats/2, (z1-z0)/2)
	return bld.Translate(h, x, y, (z0+z1)/2)
}

// sectorPrismZ creates an annular sector centered at angle 0 (+X axis), spanning
// radius [rIn, rOut], total angular width spanRad, and height [z0, z1].
func sectorPrismZ(bld *gsdf.Builder, rIn, rOut, spanRad, z0, z1 float32) glbuild.Shader3D {
	const steps = 16
	half := spanRad / 2
	verts := make([]ms2.Vec, 0, 2*(steps+1))
	for i := 0; i <= steps; i++ {
		a := -half + spanRad*float32(i)/float32(steps)
		s, c := math32.Sincos(a)
		verts = append(verts, ms2.Vec{X: rOut * c, Y: rOut * s})
	}
	for i := steps; i >= 0; i-- {
		a := -half + spanRad*float32(i)/float32(steps)
		s, c := math32.Sincos(a)
		verts = append(verts, ms2.Vec{X: rIn * c, Y: rIn * s})
	}
	poly := bld.NewPolygon(verts)
	ext := bld.Extrude(poly, z1-z0)
	return bld.Translate(ext, 0, 0, (z0+z1)/2)
}

// teardropHole2D creates a 2D circular hole of radius r with a 45-degree top teardrop peak
// (along +Y in 2D) so horizontal holes print cleanly without top sag on FDM printers.
func teardropHole2D(bld *gsdf.Builder, r float32) glbuild.Shader2D {
	circle := bld.NewCircle(r)
	// Tangent lines at +/-45 deg from top intersect at y = r * sqrt(2).
	s := r * math32.Sqrt2
	tri := bld.NewPolygon([]ms2.Vec{
		{X: -r * math32.Sqrt2 / 2, Y: r * math32.Sqrt2 / 2},
		{X: r * math32.Sqrt2 / 2, Y: r * math32.Sqrt2 / 2},
		{X: 0, Y: s},
	})
	return bld.Union2D(circle, tri)
}

// horizontalTeardropHole creates a horizontal teardrop bore along the +X axis (with teardrop
// peak pointing +Z), centered at (0, 0, 0) with length along X.
func horizontalTeardropHole(bld *gsdf.Builder, r, length float32) glbuild.Shader3D {
	td := teardropHole2D(bld, r)
	// Extrude along Z, then rotate so local +Z -> +X and local +Y -> +Z.
	ext := bld.Extrude(td, length)
	// First rotate +pi/2 around X: (x, y, z) -> (x, -z, y), so local +Y -> +Z, local +Z -> -Y.
	// Then rotate +pi/2 around Z: (x, y, z) -> (-y, x, z), so -Y -> +X and +Z stays +Z.
	rot := bld.Rotate(ext, math32.Pi/2, axisX)
	return bld.Rotate(rot, math32.Pi/2, axisZ)
}

// chamferedArchCutout2D creates a 2D rectangular cutout of width w, height h centered at (0,0)
// whose top two corners are chamfered at 45 degrees by chamfer c (support-free horizontal slot).
func chamferedArchCutout2D(bld *gsdf.Builder, w, h, c float32) glbuild.Shader2D {
	hw, hh := w/2, h/2
	if c <= 0 {
		return bld.NewRectangle(w, h, 0)
	}
	return bld.NewPolygon([]ms2.Vec{
		{X: -hw, Y: -hh},
		{X: hw, Y: -hh},
		{X: hw, Y: hh - c},
		{X: hw - c, Y: hh},
		{X: -hw + c, Y: hh},
		{X: -hw, Y: hh - c},
	})
}

// taperedTunnelX creates a straight flared box/trapezoid along X from x0 to x1,
// tapering linearly in Y from width w0 at x0 to width w1 at x1, and spanning Z in [z0, z1].
func taperedTunnelX(bld *gsdf.Builder, x0, x1, w0, w1, z0, z1 float32) glbuild.Shader3D {
	poly := bld.NewPolygon([]ms2.Vec{
		{X: x0, Y: -w0 / 2},
		{X: x1, Y: -w1 / 2},
		{X: x1, Y: w1 / 2},
		{X: x0, Y: w0 / 2},
	})
	ext := bld.Extrude(poly, z1-z0)
	return bld.Translate(ext, 0, 0, (z0+z1)/2)
}

// unionAll safely unions 1 or more 3D SDFs.
func unionAll(bld *gsdf.Builder, shapes ...glbuild.Shader3D) glbuild.Shader3D {
	if len(shapes) == 0 {
		panic("unionAll called with no shapes")
	}
	if len(shapes) == 1 {
		return shapes[0]
	}
	return bld.Union(shapes...)
}
