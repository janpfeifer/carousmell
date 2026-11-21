package cad

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
	"github.com/soypat/gsdf/gleval"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

type slicePart struct {
	name  string
	color color.RGBA
	sdf   gleval.SDF3
}

// RenderCrossSectionSVG generates an annotated 2D engineering blueprint SVG containing:
//   - Left panel: Exact XZ airflow & labyrinth cross-section sliced through Y = 0
//     from the actual 3D SDFs (with active vial + wick at +X, airflow arrows, and key dimensions).
//   - Right panel: Exact XY horizontal cross-section through the carousel & electronics layout.
//
// Because GitHub renders SVG natively in README.md, this provides a crisp, zoomable
// technical schematic verified straight from the CAD SDFs.
func RenderCrossSectionSVG(p Params, outPath string) error {
	var bld gsdf.Builder

	compile := func(name string, s glbuild.Shader3D, c color.RGBA) (slicePart, error) {
		cpu, err := gleval.NewCPUSDF3(s)
		if err != nil {
			return slicePart{}, err
		}
		return slicePart{name: name, color: c, sdf: cpu}, nil
	}

	// Build all 5 printable parts (with carousel rotated by 1 slot so active vial & wick sit at +X).
	var parts []slicePart
	for _, sp := range AssemblyParts() {
		s := sp.Build(&bld, p)
		if sp.Name == "carousel" {
			s = bld.Rotate(s, -p.SlotAngleRad(), axisZ)
		}
		spSlice, err := compile(sp.Name, s, sp.Color)
		if err != nil {
			return err
		}
		parts = append(parts, spSlice)
	}
	for _, hw := range BuildHardwareMockups(&bld, p) {
		s := hw.Shape
		if hw.Name == "vials" || hw.Name == "wicks" {
			s = bld.Rotate(s, -p.SlotAngleRad(), axisZ)
		}
		spSlice, err := compile(hw.Name, s, hw.Color)
		if err != nil {
			return err
		}
		parts = append(parts, spSlice)
	}

	// --- 1. Sample exact XZ cross-section at Y = 0 ---
	// X in [-70, +76] mm (146 mm), Z in [-4, +154] mm (158 mm) -> 4 px/mm = 584 x 632 px.
	const (
		xzScale   = 4.0
		xzMinX    = float32(-70.0)
		xzMaxX    = float32(76.0)
		xzMinZ    = float32(-4.0)
		xzMaxZ    = float32(154.0)
		xzW       = int((xzMaxX - xzMinX) * xzScale)
		xzH       = int((xzMaxZ - xzMinZ) * xzScale)
		xyScale   = 4.0
		xyHalf    = float32(68.0)
		xySize    = int(2 * xyHalf * xyScale)
	)

	xzImg := sampleSlice(parts, xzW, xzH, func(px, py int) ms3.Vec {
		x := xzMinX + (float32(px)+0.5)/xzScale
		z := xzMaxZ - (float32(py)+0.5)/xzScale
		return ms3.Vec{X: x, Y: 0, Z: z}
	})

	// --- 2. Sample exact XY carousel plan slice at Z = CarouselBottomZ + 32 mm (through vial necks & radial fins) ---
	zCarSlice := p.CarouselBottomZ() + p.CarouselCupTopLocalZ() + 8.0
	xyCarImg := sampleSlice(parts, xySize, xySize, func(px, py int) ms3.Vec {
		x := -xyHalf + (float32(px)+0.5)/xyScale
		y := xyHalf - (float32(py)+0.5)/xyScale
		return ms3.Vec{X: x, Y: y, Z: zCarSlice}
	})

	// --- 3. Sample exact XY base electronics floorplan at Z = 12 mm ---
	xyBaseImg := sampleSlice(parts, xySize, xySize, func(px, py int) ms3.Vec {
		x := -xyHalf + (float32(px)+0.5)/xyScale
		y := xyHalf - (float32(py)+0.5)/xyScale
		return ms3.Vec{X: x, Y: y, Z: 12.0}
	})

	xzB64, err := encodePNGBase64(xzImg)
	if err != nil {
		return err
	}
	xyCarB64, err := encodePNGBase64(xyCarImg)
	if err != nil {
		return err
	}
	xyBaseB64, err := encodePNGBase64(xyBaseImg)
	if err != nil {
		return err
	}

	// Coordinate helpers mapping mm -> SVG pixels in the left XZ panel (origin at (36, 76), size 520 x 563).
	const (
		leftX = 36.0
		leftY = 82.0
		leftW = 520.0
		leftH = leftW * float64(xzH) / float64(xzW)
	)
	mx := func(xMM float32) float64 {
		return leftX + float64(xMM-xzMinX)/float64(xzMaxX-xzMinX)*leftW
	}
	mz := func(zMM float32) float64 {
		return leftY + float64(xzMaxZ-zMM)/float64(xzMaxZ-xzMinZ)*leftH
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1120 710" width="1120" height="710" font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif">
  <defs>
    <marker id="arrow" viewBox="0 0 10 10" refX="7" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
      <path d="M 0 1.5 L 8.5 5 L 0 8.5 z" fill="#0284c7"/>
    </marker>
    <marker id="dim" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse">
      <path d="M 0 2 L 8 5 L 0 8 z" fill="#334155"/>
    </marker>
  </defs>

  <rect width="1120" height="710" fill="#f8fafc" rx="12"/>
  <text x="36" y="38" font-size="22" font-weight="700" fill="#0f172a">Carousmell — Exact SDF Cross-Sections &amp; Airflow Verification</text>
  <text x="36" y="60" font-size="13" fill="#475569">Generated directly from Go (soypat/gsdf) CAD geometry · All dimensions in mm · Creality K1C support-free layout (Ø%.0f × %.1f mm)</text>

  <!-- Left Panel: XZ Cross-Section (Y = 0) -->
  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#ffffff" stroke="#cbd5e1" stroke-width="1.5" rx="8"/>
  <image x="%.1f" y="%.1f" width="%.1f" height="%.1f" href="data:image/png;base64,%s"/>
  <text x="%.1f" y="%.1f" font-size="13" font-weight="600" fill="#0f172a">A. XZ Airflow Cross-Section (Y = 0)</text>

  <!-- Stepper shaft centerline X = 0 -->
  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#64748b" stroke-width="1" stroke-dasharray="5,4"/>
  <text x="%.1f" y="%.1f" font-size="10.5" fill="#475569">Shaft Axis (X=0)</text>

  <!-- Airflow streamline arrows (Fan -> Curved Duct -> Lid Aperture -> Active Wick -> Flared Exhaust) -->
  <path d="M %.1f %.1f Q %.1f %.1f %.1f %.1f T %.1f %.1f Q %.1f %.1f %.1f %.1f" fill="none" stroke="#0284c7" stroke-width="3" stroke-linecap="round" marker-end="url(#arrow)"/>
  <text x="%.1f" y="%.1f" font-size="11.5" font-weight="600" fill="#0369a1">45° Push Airflow</text>
  <text x="%.1f" y="%.1f" font-size="11" font-weight="600" fill="#0369a1">Exhaust (22 mm)</text>

  <!-- Callouts on XZ slice -->
  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#dc2626" stroke-width="1.2"/>
  <text x="%.1f" y="%.1f" font-size="11" font-weight="600" fill="#b91c1c">0.6 mm Labyrinth Gap</text>

  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#334155" stroke-width="1.2"/>
  <text x="%.1f" y="%.1f" font-size="11" fill="#1e293b">Active 2 ml Vial &amp; 3 mm Wick</text>

  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#334155" stroke-width="1.2"/>
  <text x="%.1f" y="%.1f" font-size="11" fill="#1e293b">motor_deck Vapor Barrier</text>

  <!-- Overall height dimension -->
  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#334155" stroke-width="1.2" marker-start="url(#dim)" marker-end="url(#dim)"/>
  <text x="%.1f" y="%.1f" font-size="11" font-weight="600" fill="#334155" transform="rotate(-90 %.1f %.1f)">H = %.1f mm</text>

  <!-- Right Top Panel: XY Carousel & Labyrinth Plan -->
  <rect x="584" y="82" width="250" height="266" fill="#ffffff" stroke="#cbd5e1" stroke-width="1.5" rx="8"/>
  <image x="594" y="108" width="230" height="230" href="data:image/png;base64,%s"/>
  <text x="596" y="102" font-size="12.5" font-weight="600" fill="#0f172a">B. Carousel &amp; Labyrinth (Z = %.0f mm)</text>

  <!-- Right Top-Right Panel: XY Electronics Base Plan -->
  <rect x="846" y="82" width="250" height="266" fill="#ffffff" stroke="#cbd5e1" stroke-width="1.5" rx="8"/>
  <image x="856" y="108" width="230" height="230" href="data:image/png;base64,%s"/>
  <text x="858" y="102" font-size="12.5" font-weight="600" fill="#0f172a">C. Base Electronics Bay (Z = 12 mm)</text>

  <!-- Legend & Print Spec Card -->
  <rect x="584" y="366" width="512" height="279" fill="#ffffff" stroke="#cbd5e1" stroke-width="1.5" rx="8"/>
  <text x="602" y="394" font-size="14" font-weight="700" fill="#0f172a">Printable Parts &amp; Creality K1C Verification</text>

  <g font-size="12" fill="#334155">
    <rect x="602" y="410" width="14" height="14" rx="3" fill="#3e4c5e"/>
    <text x="624" y="421"><tspan font-weight="600" fill="#0f172a">1. base.stl</tspan> (Hyper PLA) — Electronics tub, Pi Zero, ULN2003, M12 button</text>

    <rect x="602" y="434" width="14" height="14" rx="3" fill="#6c7a89"/>
    <text x="624" y="445"><tspan font-weight="600" fill="#0f172a">2. motor_deck.stl</tspan> (PLA/PETG) — Stepper bridge &amp; KY-024 Hall cradle</text>

    <rect x="602" y="458" width="14" height="14" rx="3" fill="#2ea89e"/>
    <text x="624" y="469"><tspan font-weight="600" fill="#0f172a">3. carousel.stl</tspan> (PETG) — 11-slot rotor, double-D hub, 11 radial fins</text>

    <rect x="602" y="482" width="14" height="14" rx="3" fill="#dce3ea" stroke="#94a3b8"/>
    <text x="624" y="493"><tspan font-weight="600" fill="#0f172a">4. lid_labyrinth.stl</tspan> (PETG) — 0.6 mm seal rings + flared 22 mm nozzle</text>

    <rect x="602" y="506" width="14" height="14" rx="3" fill="#e69144"/>
    <text x="624" y="517"><tspan font-weight="600" fill="#0f172a">5. duct_hood.stl</tspan> (PETG) — 45° Noctua 40 mm fan cradle &amp; curved duct</text>

    <line x1="602" y1="536" x2="1078" y2="536" stroke="#e2e8f0" stroke-width="1"/>
    <text x="602" y="558" font-weight="600" fill="#0f172a">DFM &amp; Tolerance Summary (Verified by automated SDF tests):</text>
    <text x="602" y="578">• Build volume: Ø%.0f × %.1f mm (well inside K1C 220 × 220 × 250 mm)</text>
    <text x="602" y="597">• Zero supports required: lid &amp; motor_deck exported upside-down; 45° duct</text>
    <text x="602" y="616">• Non-contact labyrinth: %.2f mm radial &amp; %.2f mm vertical air gaps</text>
    <text x="602" y="635">• Min PETG wall ≥ 2.2 mm · Includes calibration_coupon.stl fit-check</text>
  </g>
</svg>
`,
		2*p.OuterRadius, p.HoodTopZ(),
		leftX, leftY, leftW, leftH,
		leftX, leftY, leftW, leftH, xzB64,
		leftX+12, leftY+20,
		// Centerline
		mx(0), mz(0), mx(0), mz(p.HoodTopZ()),
		mx(2), mz(14),
		// Airflow path
		mx(-26), mz(p.HoodBottomZ()+36),
		mx(-4), mz(p.HoodBottomZ()+14),
		mx(20), mz(p.HoodBottomZ()+9),
		mx(32), mz(p.LidCeilingInZ()-3),
		mx(34), mz(84),
		mx(72), mz(84),
		mx(-60), mz(p.HoodBottomZ()+42),
		mx(45), mz(72),
		// Labyrinth gap callout
		mx(-32), mz(p.LidCeilingInZ()), mx(-64), mz(p.LidCeilingInZ()+14),
		mx(-67), mz(p.LidCeilingInZ()+12),
		// Active vial callout
		mx(32), mz(62), mx(44), mz(54),
		mx(20), mz(51),
		// motor_deck callout
		mx(-30), mz(p.BaseHeight-1.5), mx(-64), mz(33),
		mx(-67), mz(31),
		// Overall height dimension
		mx(-66), mz(0), mx(-66), mz(p.HoodTopZ()),
		mx(-68)-4, mz(72), mx(-68)-4, mz(72), p.HoodTopZ(),
		// Right slices
		xyCarB64, zCarSlice,
		xyBaseB64,
		// Summary specs
		2*p.OuterRadius, p.HoodTopZ(),
		p.LabyrinthRadialGap, p.LabyrinthVertGap,
	)

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}

func sampleSlice(parts []slicePart, w, h int, toWorld func(px, py int) ms3.Vec) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var vp gleval.VecPool
	pos := make([]ms3.Vec, w)
	dist := make([]float32, w)
	bestDist := make([]float32, w)
	bestIdx := make([]int, w)

	bg := color.RGBA{R: 252, G: 253, B: 255, A: 255}
	outline := color.RGBA{R: 30, G: 41, B: 59, A: 255}

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			pos[px] = toWorld(px, py)
			bestDist[px] = 1e9
			bestIdx[px] = -1
		}
		for idx := range parts {
			_ = parts[idx].sdf.Evaluate(pos, dist, &vp)
			for px := 0; px < w; px++ {
				if dist[px] < bestDist[px] {
					bestDist[px] = dist[px]
					bestIdx[px] = idx
				}
			}
		}
		for px := 0; px < w; px++ {
			d := bestDist[px]
			switch {
			case d < -0.35:
				img.SetRGBA(px, py, parts[bestIdx[px]].color)
			case d <= 0.15:
				img.SetRGBA(px, py, outline)
			default:
				img.SetRGBA(px, py, bg)
			}
		}
	}
	return img
}

func encodePNGBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
