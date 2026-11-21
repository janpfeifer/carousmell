// Command generate builds the parametric Carousmell CAD models using github.com/soypat/gsdf,
// exports Creality K1C-ready binary STL files, renders README-ready PNG & SVG engineering
// visualizations, and generates a standalone interactive 3D WebGL HTML viewer.
//
// Usage:
//
//	go run ./cad/cmd/generate                     # Generate all STLs (0.5mm), PNGs, SVG & HTML viewer
//	go run ./cad/cmd/generate --res 0.25          # High-resolution print-ready STLs
//	go run ./cad/cmd/generate --part carousel     # Export only one part
//	go run ./cad/cmd/generate --serve             # Generate & launch local interactive 3D UI server
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janpfeifer/carousmell/cad"
	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf"
)

func main() {
	var (
		partFlag     = flag.String("part", "all", "Part to generate: all, base, motor_deck, carousel, lid_labyrinth, duct_hood, calibration_coupon, assembly")
		resFlag      = flag.Float64("res", 0.50, "Marching-cubes voxel resolution in mm (0.50 = fast preview, 0.20-0.25 = final print)")
		viewerRes    = flag.Float64("viewer-res", 1.15, "Mesh resolution in mm embedded into viewer.html (keeps HTML compact & fast)")
		stlDir       = flag.String("out", "cad/out", "Output directory for generated .stl files")
		docsDir      = flag.String("docs", "docs", "Output directory for PNG, SVG, and interactive HTML visualizations")
		skipSTL      = flag.Bool("no-stl", false, "Skip STL export")
		skipVisuals  = flag.Bool("no-visuals", false, "Skip PNG/SVG/HTML visualization generation")
		withAssembly = flag.Bool("assembly", true, "Also export combined assembly_preview.stl")
		imgWidth     = flag.Int("img-width", 960, "Rendered PNG width in pixels")
		serveFlag    = flag.Bool("serve", false, "Start a local HTTP server displaying the interactive 3D viewer UI")
		serveAddr    = flag.String("addr", "127.0.0.1:8080", "Address for --serve interactive UI")
	)
	flag.Parse()

	p := cad.DefaultParams()
	totalStart := time.Now()

	// 1. Export printable .stl files.
	if !*skipSTL {
		if err := exportSTLs(p, *partFlag, float32(*resFlag), *stlDir, *withAssembly); err != nil {
			log.Fatalf("STL export failed: %v", err)
		}
	}

	// 2. Generate README visualizations (PNG renders, annotated SVG cross-section, interactive HTML).
	viewerPath := filepath.Join(*docsDir, "viewer.html")
	if !*skipVisuals {
		if err := generateVisuals(p, *docsDir, *imgWidth, float32(*viewerRes)); err != nil {
			log.Fatalf("Visualization generation failed: %v", err)
		}
	}

	fmt.Printf("\nDone in %s.\n", time.Since(totalStart).Round(time.Millisecond))

	// 3. Optionally serve the interactive 3D CAD viewer UI locally.
	if *serveFlag {
		if _, err := os.Stat(viewerPath); err != nil {
			log.Fatalf("viewer.html not found at %s (run without --no-visuals first): %v", viewerPath, err)
		}
		serveViewer(*docsDir, *serveAddr)
	}
}

func exportSTLs(p cad.Params, partName string, res float32, outDir string, withAssembly bool) error {
	var specs []cad.PartSpec
	switch strings.ToLower(strings.TrimSpace(partName)) {
	case "all":
		specs = cad.PrintableParts()
	case "assembly":
		return exportAssemblySTL(p, res, filepath.Join(outDir, "assembly_preview.stl"))
	default:
		sp, err := cad.FindPart(partName)
		if err != nil {
			return err
		}
		specs = []cad.PartSpec{sp}
	}

	fmt.Printf("=== Exporting Creality K1C STL files (resolution = %.2f mm) ===\n", res)
	for _, sp := range specs {
		var bld gsdf.Builder
		s := sp.Build(&bld, p)
		sPrint := sp.PrintTransform(&bld, s, p)
		if err := bld.Err(); err != nil {
			return fmt.Errorf("building %s: %w", sp.Name, err)
		}
		outPath := filepath.Join(outDir, sp.Name+".stl")
		stats, _, err := cad.ExportPartSTL(sp.Name, sPrint, res, outPath)
		if err != nil {
			return err
		}
		sz := stats.Bounds.Size()
		fmt.Printf("  %-22s -> %-34s %7d tris | %5.1f x %5.1f x %5.1f mm | %s\n",
			sp.Name, stats.Path, stats.Triangles, sz.X, sz.Y, sz.Z, stats.Duration.Round(time.Millisecond))
	}

	if partName == "all" && withAssembly {
		if err := exportAssemblySTL(p, max(res, 0.55), filepath.Join(outDir, "assembly_preview.stl")); err != nil {
			return err
		}
	}
	return nil
}

func exportAssemblySTL(p cad.Params, res float32, outPath string) error {
	var bld gsdf.Builder
	asm := cad.BuildAssembly(&bld, p, false)
	if err := bld.Err(); err != nil {
		return fmt.Errorf("building assembly: %w", err)
	}
	stats, _, err := cad.ExportPartSTL("assembly_preview", asm, res, outPath)
	if err != nil {
		return err
	}
	sz := stats.Bounds.Size()
	fmt.Printf("  %-22s -> %-34s %7d tris | %5.1f x %5.1f x %5.1f mm | %s\n",
		"assembly_preview", stats.Path, stats.Triangles, sz.X, sz.Y, sz.Z, stats.Duration.Round(time.Millisecond))
	return nil
}

func generateVisuals(p cad.Params, docsDir string, width int, viewerRes float32) error {
	height := width * 3 / 4
	fmt.Printf("\n=== Generating README Visualizations (%s) ===\n", docsDir)

	// 1. Shaded 3D Isometric Render
	t0 := time.Now()
	isoPath := filepath.Join(docsDir, "render_isometric.png")
	err := cad.RenderPNG(p, cad.RenderOptions{
		Width:        width,
		Height:       height,
		CameraPos:    ms3.Vec{X: 235, Y: -215, Z: 192},
		Target:       ms3.Vec{X: 2, Y: 0, Z: 70},
		FOVDeg:       33,
		ShowHardware: true,
	}, isoPath)
	if err != nil {
		return fmt.Errorf("render isometric PNG: %w", err)
	}
	fmt.Printf("  %-22s -> %-34s (%dx%d) | %s\n", "isometric PNG", isoPath, width, height, time.Since(t0).Round(time.Millisecond))

	// 2. Shaded 3D Cutaway Cross-Section Render (exposes active vial, 0.6 mm labyrinth, 45 deg duct & electronics)
	t0 = time.Now()
	cutPath := filepath.Join(docsDir, "render_cutaway.png")
	err = cad.RenderPNG(p, cad.RenderOptions{
		Width:        width,
		Height:       height,
		CameraPos:    ms3.Vec{X: 48, Y: -280, Z: 122},
		Target:       ms3.Vec{X: 4, Y: 0, Z: 73},
		FOVDeg:       32,
		CutawayY:     true,
		ShowHardware: true,
		RotateToVial: true,
	}, cutPath)
	if err != nil {
		return fmt.Errorf("render cutaway PNG: %w", err)
	}
	fmt.Printf("  %-22s -> %-34s (%dx%d) | %s\n", "cutaway PNG", cutPath, width, height, time.Since(t0).Round(time.Millisecond))

	// 3. Shaded 3D Exploded Assembly Render
	t0 = time.Now()
	expPath := filepath.Join(docsDir, "render_exploded.png")
	err = cad.RenderPNG(p, cad.RenderOptions{
		Width:        width,
		Height:       width * 10 / 9,
		CameraPos:    ms3.Vec{X: 350, Y: -320, Z: 270},
		Target:       ms3.Vec{X: 0, Y: 0, Z: 138},
		FOVDeg:       36,
		Exploded:     true,
		ShowHardware: true,
	}, expPath)
	if err != nil {
		return fmt.Errorf("render exploded PNG: %w", err)
	}
	fmt.Printf("  %-22s -> %-34s (%dx%d) | %s\n", "exploded PNG", expPath, width, width*10/9, time.Since(t0).Round(time.Millisecond))

	// 4. Annotated 2D Technical Cross-Section SVG
	t0 = time.Now()
	svgPath := filepath.Join(docsDir, "cross_section.svg")
	if err := cad.RenderCrossSectionSVG(p, svgPath); err != nil {
		return fmt.Errorf("render SVG: %w", err)
	}
	fmt.Printf("  %-22s -> %-34s | %s\n", "technical SVG", svgPath, time.Since(t0).Round(time.Millisecond))

	// 5. Standalone Interactive 3D WebGL Viewer HTML
	t0 = time.Now()
	var viewerParts []cad.ViewerPartData
	for _, sp := range cad.AssemblyParts() {
		var bld gsdf.Builder
		s := sp.Build(&bld, p)
		if err := bld.Err(); err != nil {
			return err
		}
		tris, _, err := cad.MeshSDF(s, viewerRes)
		if err != nil {
			return fmt.Errorf("meshing %s for HTML viewer: %w", sp.Name, err)
		}
		viewerParts = append(viewerParts, cad.ViewerPartData{Spec: sp, Triangles: tris})
	}
	htmlPath := filepath.Join(docsDir, "viewer.html")
	if err := cad.WriteInteractiveHTML(viewerParts, p, htmlPath); err != nil {
		return fmt.Errorf("writing HTML viewer: %w", err)
	}
	fi, _ := os.Stat(htmlPath)
	fmt.Printf("  %-22s -> %-34s (%.1f KB) | %s\n", "interactive HTML", htmlPath, float64(fi.Size())/1024, time.Since(t0).Round(time.Millisecond))
	return nil
}

func serveViewer(docsDir, addr string) {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(docsDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(docsDir, "viewer.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listening on %s: %v", addr, err)
	}
	fmt.Printf("\nInteractive 3D CAD viewer running at http://%s  (Ctrl+C to stop)\n", ln.Addr().String())
	log.Fatal(http.Serve(ln, mux))
}
