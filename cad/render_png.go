package cad

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/chewxy/math32"
	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gleval"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// RenderPart pairs an SDF evaluator with a surface albedo for multi-part raymarching.
type renderObject struct {
	sdf   gleval.SDF3
	bb    ms3.Box
	color [3]float32 // Linear-ish RGB in [0,1]
}

// RenderOptions configures CPU SDF raymarching for README visualizations.
type RenderOptions struct {
	Width, Height int
	CameraPos     ms3.Vec
	Target        ms3.Vec
	FOVDeg        float32
	CutawayY      bool // Slice away Y < 0 half-space to expose the XZ airflow cross-section
	Exploded      bool // Separate parts vertically by PartSpec.ExplodeZ
	ShowHardware  bool // Include COTS hardware silhouettes (vials, wicks, stepper, fan, Pi)
	RotateToVial  bool // Rotate carousel by 1 slot so an active vial & wick sits at +X
}

func rgbaToVec3(c color.RGBA) [3]float32 {
	return [3]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255}
}

func buildRenderScene(p Params, opts RenderOptions) ([]renderObject, error) {
	var bld gsdf.Builder
	var objs []renderObject

	addShape := func(s glbuild.Shader3D, c color.RGBA, explodeZ float32) error {
		if opts.Exploded && explodeZ != 0 {
			s = bld.Translate(s, 0, 0, explodeZ)
		}
		if opts.CutawayY {
			clip := boxAt(&bld, -140, 140, -140, 0, -20, 400, 0)
			s = bld.Difference(s, clip)
		}
		cpu, err := gleval.NewCPUSDF3(s)
		if err != nil {
			return err
		}
		objs = append(objs, renderObject{
			sdf:   cpu,
			bb:    cpu.Bounds(),
			color: rgbaToVec3(c),
		})
		return nil
	}

	for _, sp := range AssemblyParts() {
		s := sp.Build(&bld, p)
		if sp.Name == "carousel" && opts.RotateToVial {
			// Rotate Slot 1 onto +X (theta = 0) so the active amber vial & wick sit in the cutaway!
			s = bld.Rotate(s, -p.SlotAngleRad(), axisZ)
		}
		if err := addShape(s, sp.Color, sp.ExplodeZ); err != nil {
			return nil, fmt.Errorf("compiling %s for render: %w", sp.Name, err)
		}
	}

	if opts.ShowHardware {
		for _, hw := range BuildHardwareMockups(&bld, p) {
			s := hw.Shape
			var ez float32
			switch hw.Name {
			case "vials", "wicks":
				if opts.RotateToVial {
					s = bld.Rotate(s, -p.SlotAngleRad(), axisZ)
				}
				ez = 45
			case "stepper":
				ez = 20
			case "noctua_fan":
				ez = 136
			case "pi_zero":
				ez = 0
			}
			// Keep active vial & wick unclipped on cutaway if desired? Clipping at Y=0 shows
			// the wick cleanly inside the amber glass vial, just like diagram_2.jpg!
			if err := addShape(s, hw.Color, ez); err != nil {
				return nil, fmt.Errorf("compiling hardware %s: %w", hw.Name, err)
			}
		}
	}

	if err := bld.Err(); err != nil {
		return nil, err
	}
	return objs, nil
}

// evalScene evaluates the closest object distance and index at a single point using a local VecPool.
func evalScene(objs []renderObject, pt ms3.Vec, vp *gleval.VecPool) (float32, int) {
	pos := [1]ms3.Vec{pt}
	dist := [1]float32{0}
	bestD := float32(1e9)
	bestIdx := -1
	for i := range objs {
		// Fast AABB lower-bound skip: if point is further from AABB than bestD, skip SDF evaluation!
		dBox := boxDistance(objs[i].bb, pt)
		if dBox >= bestD {
			continue
		}
		_ = objs[i].sdf.Evaluate(pos[:], dist[:], vp)
		if dist[0] < bestD {
			bestD = dist[0]
			bestIdx = i
		}
	}
	return bestD, bestIdx
}

func boxDistance(b ms3.Box, p ms3.Vec) float32 {
	var dx, dy, dz float32
	if p.X < b.Min.X {
		dx = b.Min.X - p.X
	} else if p.X > b.Max.X {
		dx = p.X - b.Max.X
	}
	if p.Y < b.Min.Y {
		dy = b.Min.Y - p.Y
	} else if p.Y > b.Max.Y {
		dy = p.Y - b.Max.Y
	}
	if p.Z < b.Min.Z {
		dz = b.Min.Z - p.Z
	} else if p.Z > b.Max.Z {
		dz = p.Z - b.Max.Z
	}
	return math32.Sqrt(dx*dx + dy*dy + dz*dz)
}

func rayAABB(ro, rd ms3.Vec, b ms3.Box) (float32, float32, bool) {
	tmin := float32(0.0)
	tmax := float32(1e6)
	for axis := 0; axis < 3; axis++ {
		var o, d, bmin, bmax float32
		switch axis {
		case 0:
			o, d, bmin, bmax = ro.X, rd.X, b.Min.X, b.Max.X
		case 1:
			o, d, bmin, bmax = ro.Y, rd.Y, b.Min.Y, b.Max.Y
		case 2:
			o, d, bmin, bmax = ro.Z, rd.Z, b.Min.Z, b.Max.Z
		}
		if math32.Abs(d) < 1e-7 {
			if o < bmin || o > bmax {
				return 0, 0, false
			}
		} else {
			invD := 1.0 / d
			t1 := (bmin - o) * invD
			t2 := (bmax - o) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tmin {
				tmin = t1
			}
			if t2 < tmax {
				tmax = t2
			}
			if tmin > tmax {
				return 0, 0, false
			}
		}
	}
	return tmin, tmax, true
}

// RenderPNG raymarches the exact 3D SDFs into a studio-lit RGBA PNG image.
func RenderPNG(p Params, opts RenderOptions, outPath string) error {
	if opts.Width <= 0 {
		opts.Width = 960
	}
	if opts.Height <= 0 {
		opts.Height = 720
	}
	if opts.FOVDeg <= 0 {
		opts.FOVDeg = 32
	}

	objs, err := buildRenderScene(p, opts)
	if err != nil {
		return err
	}

	// Global bounding box (expanded slightly for safe ray entry/exit).
	sceneBB := objs[0].bb
	for _, o := range objs[1:] {
		sceneBB = sceneBB.Union(o.bb)
	}
	sceneBB.Min = ms3.AddScalar(-3.0, sceneBB.Min)
	sceneBB.Max = ms3.AddScalar(3.0, sceneBB.Max)

	// Camera orthonormal basis (Z-up).
	forward := ms3.Unit(ms3.Sub(opts.Target, opts.CameraPos))
	worldUp := ms3.Vec{Z: 1}
	right := ms3.Unit(ms3.Cross(forward, worldUp))
	up := ms3.Cross(right, forward)

	aspect := float32(opts.Width) / float32(opts.Height)
	tanHalfFOV := math32.Tan(opts.FOVDeg * math32.Pi / 360.0)

	// Three-point studio lighting directions.
	keyLight := ms3.Unit(ms3.Vec{X: 0.60, Y: -0.75, Z: 0.90})
	fillLight := ms3.Unit(ms3.Vec{X: -0.70, Y: -0.35, Z: 0.40})
	rimLight := ms3.Unit(ms3.Vec{X: -0.45, Y: 0.85, Z: 0.65})

	img := image.NewRGBA(image.Rect(0, 0, opts.Width, opts.Height))
	workers := max(1, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		yStart := w * opts.Height / workers
		yEnd := (w + 1) * opts.Height / workers
		go func(y0, y1 int) {
			defer wg.Done()
			var vp gleval.VecPool
			for py := y0; py < y1; py++ {
				// Subtle studio vertical background gradient (clean off-white / light slate).
				bgT := float32(py) / float32(opts.Height)
				bgR := 0.96 - 0.06*bgT
				bgG := 0.97 - 0.05*bgT
				bgB := 0.99 - 0.04*bgT

				ndcY := 1.0 - 2.0*(float32(py)+0.5)/float32(opts.Height)
				for px := 0; px < opts.Width; px++ {
					ndcX := 2.0*(float32(px)+0.5)/float32(opts.Width) - 1.0
					rd := ms3.Unit(ms3.Add(
						forward,
						ms3.Add(
							ms3.Scale(ndcX*aspect*tanHalfFOV, right),
							ms3.Scale(ndcY*tanHalfFOV, up),
						),
					))

					colR, colG, colB := bgR, bgG, bgB

					// Floor shadow plane at Z = 0 for grounding.
					floorHit := false
					var floorT float32
					if rd.Z < -1e-4 && opts.CameraPos.Z > 0 {
						ft := -opts.CameraPos.Z / rd.Z
						if ft > 0 && ft < 900 {
							fp := ms3.Add(opts.CameraPos, ms3.Scale(ft, rd))
							r2 := fp.X*fp.X + fp.Y*fp.Y
							if r2 < 110*110 {
								floorHit = true
								floorT = ft
								// Soft contact shadow directly under the cylindrical base.
								distFromBase := math32.Sqrt(r2)
								shad := smoothstep(42.0, 92.0, distFromBase)
								colR *= 0.84 + 0.16*shad
								colG *= 0.85 + 0.15*shad
								colB *= 0.87 + 0.13*shad
							}
						}
					}

					t0, t1, hitBB := rayAABB(opts.CameraPos, rd, sceneBB)
					if hitBB {
						t := t0
						hitIdx := -1
						const maxSteps = 140
						const hitEps = float32(0.08)
						for step := 0; step < maxSteps && t < t1; step++ {
							pt := ms3.Add(opts.CameraPos, ms3.Scale(t, rd))
							d, idx := evalScene(objs, pt, &vp)
							if d < hitEps {
								hitIdx = idx
								break
							}
							// Conservative step (0.85x) prevents overshooting non-exact boolean differences.
							t += max(d*0.85, hitEps)
						}

						if hitIdx >= 0 && (!floorHit || t < floorT) {
							hitPos := ms3.Add(opts.CameraPos, ms3.Scale(t, rd))
							// Tetrahedral finite-difference normal on the hit object.
							n := calcNormal(objs[hitIdx].sdf, hitPos, &vp)

							// 4-tap fast ambient occlusion along normal.
							ao := float32(1.0)
							for k := 1; k <= 4; k++ {
								stepDist := float32(k) * 1.6
								sd, _ := evalScene(objs, ms3.Add(hitPos, ms3.Scale(stepDist, n)), &vp)
								ao -= (stepDist - max(sd, 0)) * (0.14 / float32(k))
							}
							ao = clamp01(ao)

							// Soft directional shadow toward key light.
							shadow := float32(1.0)
							st := float32(1.2)
							for k := 0; k < 22; k++ {
								sp := ms3.Add(hitPos, ms3.Scale(st, keyLight))
								if boxDistance(sceneBB, sp) > 4.0 {
									break
								}
								sd, _ := evalScene(objs, sp, &vp)
								if sd < 0.08 {
									shadow = 0.42
									break
								}
								shadow = min(shadow, 0.42+0.58*clamp01(sd*6.0/st))
								st += max(sd*0.85, 0.6)
							}

							albedo := objs[hitIdx].color
							// Highlight freshly sliced cutaway planes subtly so cross-sections pop!
							if opts.CutawayY && math32.Abs(hitPos.Y) < 0.25 && n.Y < -0.85 {
								albedo[0] = min(1.0, albedo[0]*1.12+0.05)
								albedo[1] = min(1.0, albedo[1]*1.12+0.05)
								albedo[2] = min(1.0, albedo[2]*1.12+0.05)
							}

							diffKey := max(ms3.Dot(n, keyLight), 0) * shadow
							diffFill := max(ms3.Dot(n, fillLight), 0)
							diffRim := max(ms3.Dot(n, rimLight), 0)
							hemi := 0.5 + 0.5*n.Z

							// Blinn-Phong specular highlight for clean plastic sheen.
							halfV := ms3.Unit(ms3.Sub(keyLight, rd))
							spec := math32.Pow(max(ms3.Dot(n, halfV), 0), 36.0) * 0.22 * shadow

							light := 0.26*hemi*ao + 0.64*diffKey + 0.20*diffFill*ao + 0.16*diffRim*ao
							colR = clamp01(albedo[0]*light + spec)
							colG = clamp01(albedo[1]*light + spec)
							colB = clamp01(albedo[2]*light + spec)
						}
					}

					img.SetRGBA(px, py, color.RGBA{
						R: uint8(math.Round(float64(gammaEncode(colR)) * 255)),
						G: uint8(math.Round(float64(gammaEncode(colG)) * 255)),
						B: uint8(math.Round(float64(gammaEncode(colB)) * 255)),
						A: 255,
					})
				}
			}
		}(yStart, yEnd)
	}
	wg.Wait()

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	encErr := png.Encode(f, img)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func calcNormal(sdf gleval.SDF3, p ms3.Vec, vp *gleval.VecPool) ms3.Vec {
	const e = float32(0.18)
	pts := [4]ms3.Vec{
		{X: p.X + e, Y: p.Y - e, Z: p.Z - e},
		{X: p.X - e, Y: p.Y - e, Z: p.Z + e},
		{X: p.X - e, Y: p.Y + e, Z: p.Z - e},
		{X: p.X + e, Y: p.Y + e, Z: p.Z + e},
	}
	var d [4]float32
	_ = sdf.Evaluate(pts[:], d[:], vp)
	return ms3.Unit(ms3.Vec{
		X: d[0] - d[1] - d[2] + d[3],
		Y: -d[0] - d[1] + d[2] + d[3],
		Z: -d[0] + d[1] - d[2] + d[3],
	})
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothstep(e0, e1, x float32) float32 {
	t := clamp01((x - e0) / (e1 - e0))
	return t * t * (3 - 2*t)
}

func gammaEncode(v float32) float32 {
	return math32.Pow(clamp01(v), 1.0/1.35)
}
