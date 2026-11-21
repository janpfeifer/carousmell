//go:build !webgpu

// Command viewer launches a zero-dependency local interactive 3D WebGL CAD UI in your browser.
// For the native GLFW/WebGPU raymarching window (requires libwayland-dev on Linux), run:
//
//	go run -tags webgpu ./cad/cmd/viewer
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/janpfeifer/carousmell/cad"
	"github.com/soypat/gsdf"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "Local HTTP address for interactive 3D viewer")
	res := flag.Float64("res", 1.15, "Mesh resolution in mm for the browser viewer")
	out := flag.String("out", "docs/viewer.html", "Path to write/serve viewer.html")
	rebuild := flag.Bool("rebuild", false, "Force regenerating viewer.html even if it already exists")
	flag.Parse()

	if *rebuild || fileMissing(*out) {
		fmt.Printf("Building interactive 3D viewer (%s, res=%.2f mm)...\n", *out, *res)
		p := cad.DefaultParams()
		var parts []cad.ViewerPartData
		for _, sp := range cad.AssemblyParts() {
			var bld gsdf.Builder
			s := sp.Build(&bld, p)
			if err := bld.Err(); err != nil {
				log.Fatalf("building %s: %v", sp.Name, err)
			}
			tris, _, err := cad.MeshSDF(s, float32(*res))
			if err != nil {
				log.Fatalf("meshing %s: %v", sp.Name, err)
			}
			parts = append(parts, cad.ViewerPartData{Spec: sp, Triangles: tris})
		}
		if err := cad.WriteInteractiveHTML(parts, p, *out); err != nil {
			log.Fatalf("writing %s: %v", *out, err)
		}
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listening on %s: %v", *addr, err)
	}
	fmt.Printf("Carousmell interactive 3D CAD UI running at http://%s  (Ctrl+C to stop)\n", ln.Addr().String())
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, *out)
	})
	log.Fatal(http.Serve(ln, mux))
}

func fileMissing(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err != nil
}
