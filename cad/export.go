package cad

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/soypat/geometry/ms3"
	"github.com/soypat/gsdf/gleval"
	"github.com/soypat/gsdf/glrender"
	"github.com/soypat/gsdf/gpusdf/glbuild"
)

// MeshStats summarizes a generated triangle mesh for reporting and verification.
type MeshStats struct {
	Name      string
	Path      string
	Triangles int
	Bounds    ms3.Box
	Duration  time.Duration
}

// MeshSDF tessellates a 3D SDF into triangles on the CPU at the requested voxel resolution (mm).
// It uses parallel FlatRenderer for draft/medium resolutions (>= 0.45 mm) and memory-efficient
// pruned Octree marching cubes for fine resolutions (< 0.45 mm).
func MeshSDF(s glbuild.Shader3D, resolution float32) ([]ms3.Triangle, ms3.Box, error) {
	if resolution <= 0 {
		return nil, ms3.Box{}, fmt.Errorf("resolution must be positive, got %g", resolution)
	}
	cpusdf, err := gleval.NewCPUSDF3(s)
	if err != nil {
		return nil, ms3.Box{}, fmt.Errorf("initializing CPU SDF: %w", err)
	}
	const evalBufSize = 4096
	cpusdf.VecPool().SetMinAllocationLen(evalBufSize)

	var renderer glrender.Renderer
	if resolution >= 0.45 {
		numCPU := max(1, runtime.GOMAXPROCS(0)-1)
		renderer, err = glrender.NewFlatRenderer(cpusdf, resolution, evalBufSize, numCPU)
	} else {
		renderer, err = glrender.NewOctreeRenderer(cpusdf, resolution, evalBufSize)
	}
	if err != nil {
		return nil, ms3.Box{}, fmt.Errorf("creating renderer: %w", err)
	}

	tris, err := glrender.RenderAll(renderer, cpusdf.VecPool())
	if err != nil {
		return nil, ms3.Box{}, fmt.Errorf("marching cubes render: %w", err)
	}
	if len(tris) == 0 {
		return nil, ms3.Box{}, fmt.Errorf("empty mesh produced at resolution %g mm", resolution)
	}

	// Compute actual mesh bounding box from emitted triangles.
	bb := ms3.Box{Min: tris[0][0], Max: tris[0][0]}
	for _, t := range tris {
		for _, v := range t {
			bb = bb.IncludePoint(v)
		}
	}
	return tris, bb, nil
}

// WriteSTLFile writes triangles as a binary STL file (millimeter units).
func WriteSTLFile(path string, tris []ms3.Triangle) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 1<<16)
	_, werr := glrender.WriteBinarySTL(bw, tris)
	ferr := bw.Flush()
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	if ferr != nil {
		return ferr
	}
	return cerr
}

// ExportPartSTL meshes a single 3D SDF and writes it to outPath as binary STL.
func ExportPartSTL(name string, s glbuild.Shader3D, resolution float32, outPath string) (MeshStats, []ms3.Triangle, error) {
	start := time.Now()
	tris, bb, err := MeshSDF(s, resolution)
	if err != nil {
		return MeshStats{}, nil, fmt.Errorf("meshing %s: %w", name, err)
	}
	if err := WriteSTLFile(outPath, tris); err != nil {
		return MeshStats{}, nil, fmt.Errorf("writing %s: %w", outPath, err)
	}
	return MeshStats{
		Name:      name,
		Path:      outPath,
		Triangles: len(tris),
		Bounds:    bb,
		Duration:  time.Since(start),
	}, tris, nil
}
