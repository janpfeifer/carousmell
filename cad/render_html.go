package cad

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/soypat/geometry/ms3"
)

// ViewerPartData holds a tessellated part mesh ready to embed into the standalone WebGL HTML viewer.
type ViewerPartData struct {
	Spec      PartSpec
	Triangles []ms3.Triangle
}

type embeddedPartJSON struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Material    string     `json:"material"`
	PrintNote   string     `json:"printNote"`
	Color       [3]float32 `json:"color"`
	ExplodeZ    float32    `json:"explodeZ"`
	TriCount    int        `json:"triCount"`
	Min         [3]float32 `json:"min"`
	Scale       [3]float32 `json:"scale"`
	PosB64      string     `json:"posB64"` // uint16 quantized triangle vertices (3*3*triCount)
}

// WriteInteractiveHTML generates a zero-dependency standalone interactive 3D WebGL viewer
// containing all assembled parts with orbit/zoom, live exploded-view slider, live XZ cutaway
// slider, part visibility toggles, and a Creality K1C 220x220 mm build-plate reference grid.
func WriteInteractiveHTML(parts []ViewerPartData, p Params, outPath string) error {
	payload := make([]embeddedPartJSON, 0, len(parts))
	for _, vp := range parts {
		if len(vp.Triangles) == 0 {
			continue
		}
		bb := ms3.Box{Min: vp.Triangles[0][0], Max: vp.Triangles[0][0]}
		for _, t := range vp.Triangles {
			for _, v := range t {
				bb = bb.IncludePoint(v)
			}
		}
		sz := bb.Size()
		scale := [3]float32{
			max(sz.X, 1e-4) / 65535.0,
			max(sz.Y, 1e-4) / 65535.0,
			max(sz.Z, 1e-4) / 65535.0,
		}

		// Quantize vertex coordinates to uint16 (precision < 0.002 mm) for compact self-contained HTML.
		raw := make([]byte, len(vp.Triangles)*3*3*2)
		off := 0
		for _, t := range vp.Triangles {
			for _, v := range t {
				qx := uint16(math.Round(float64((v.X - bb.Min.X) / scale[0])))
				qy := uint16(math.Round(float64((v.Y - bb.Min.Y) / scale[1])))
				qz := uint16(math.Round(float64((v.Z - bb.Min.Z) / scale[2])))
				binary.LittleEndian.PutUint16(raw[off:], qx)
				binary.LittleEndian.PutUint16(raw[off+2:], qy)
				binary.LittleEndian.PutUint16(raw[off+4:], qz)
				off += 6
			}
		}

		// Gzip-compress the quantized vertex buffer so viewer.html stays lightweight (~1-2 MB).
		var gzBuf bytes.Buffer
		gz, err := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
		if err != nil {
			return err
		}
		if _, err := gz.Write(raw); err != nil {
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}

		payload = append(payload, embeddedPartJSON{
			Name:        vp.Spec.Name,
			Description: vp.Spec.Description,
			Material:    vp.Spec.Material,
			PrintNote:   vp.Spec.PrintNote,
			Color:       rgbaToVec3(vp.Spec.Color),
			ExplodeZ:    vp.Spec.ExplodeZ,
			TriCount:    len(vp.Triangles),
			Min:         [3]float32{bb.Min.X, bb.Min.Y, bb.Min.Z},
			Scale:       scale,
			PosB64:      base64.StdEncoding.EncodeToString(gzBuf.Bytes()),
		})
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, viewerHTMLTemplate, 2*p.OuterRadius, p.HoodTopZ(), string(jsonBytes))

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}

const viewerHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light dark">
  <title>Carousmell — Interactive 3D CAD Viewer</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg-light: #f4f7fb;
      --bg-dark: #0f172a;
      --panel-light: rgba(255, 255, 255, 0.92);
      --panel-dark: rgba(15, 23, 42, 0.90);
      --border-light: #cbd5e1;
      --border-dark: #334155;
      --text-light: #0f172a;
      --text-dark: #f1f5f9;
      --muted-light: #475569;
      --muted-dark: #94a3b8;
      --brand-light: #0284c7;
      --brand-dark: #38bdf8;

      --bg: var(--bg-light);
      --panel: var(--panel-light);
      --border: var(--border-light);
      --text: var(--text-light);
      --muted: var(--muted-light);
      --brand: var(--brand-light);

      @media (prefers-color-scheme: dark) {
        --bg: var(--bg-dark);
        --panel: var(--panel-dark);
        --border: var(--border-dark);
        --text: var(--text-dark);
        --muted: var(--muted-dark);
        --brand: var(--brand-dark);
      }

      @supports (color: light-dark(white, black)) {
        --bg: light-dark(var(--bg-light), var(--bg-dark));
        --panel: light-dark(var(--panel-light), var(--panel-dark));
        --border: light-dark(var(--border-light), var(--border-dark));
        --text: light-dark(var(--text-light), var(--text-dark));
        --muted: light-dark(var(--muted-light), var(--muted-dark));
        --brand: light-dark(var(--brand-light), var(--brand-dark));
      }

      accent-color: var(--brand);
    }

    * { box-sizing: border-box; }
    html, body {
      margin: 0;
      height: 100%%;
      overflow: hidden;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif;
    }

    #viewport {
      width: 100vw;
      height: 100vh;
      display: block;
      cursor: grab;
      touch-action: none;
    }
    #viewport:active { cursor: grabbing; }

    .panel {
      position: fixed;
      top: 14px;
      left: 14px;
      width: min(320px, calc(100vw - 28px));
      max-height: calc(100vh - 28px);
      overflow-y: auto;
      padding: 14px 16px;
      border-radius: 12px;
      border: 1px solid var(--border);
      background: var(--panel);
      backdrop-filter: blur(10px);
      box-shadow: 0 10px 28px rgba(0, 0, 0, 0.14);
      font-size: 13px;
    }

    h1 {
      margin: 0 0 2px;
      font-size: 16px;
      font-weight: 700;
    }
    .sub {
      margin: 0 0 12px;
      font-size: 12px;
      color: var(--muted);
    }
    .section {
      margin-top: 12px;
      padding-top: 10px;
      border-top: 1px solid var(--border);
    }
    .section-title {
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 8px;
    }
    .slider-row {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 4px 8px;
      align-items: center;
      margin-bottom: 8px;
    }
    .slider-row input[type="range"] {
      grid-column: 1 / -1;
      width: 100%%;
    }
    .part-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 4px 0;
      cursor: pointer;
    }
    .swatch {
      width: 12px;
      height: 12px;
      border-radius: 3px;
      border: 1px solid rgba(128, 128, 128, 0.4);
      flex: 0 0 auto;
    }
    .part-meta {
      font-size: 11px;
      color: var(--muted);
      margin-left: auto;
    }
    .btn-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 6px;
    }
    button {
      border: 1px solid var(--border);
      background: transparent;
      color: var(--text);
      padding: 5px 8px;
      border-radius: 7px;
      font-size: 12px;
      cursor: pointer;
    }
    button:hover {
      border-color: var(--brand);
      color: var(--brand);
    }
    .hint {
      position: fixed;
      right: 14px;
      bottom: 14px;
      padding: 8px 12px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--panel);
      font-size: 12px;
      color: var(--muted);
    }

    /* Fallback for older browsers without accent-color support */
    @supports not (accent-color: #0284c7) {
      input[type="checkbox"].visually-hidden {
        position: absolute !important;
        clip-path: inset(50%%) !important;
        overflow: hidden !important;
        width: 1px !important;
        height: 1px !important;
        margin: -1px !important;
        padding: 0 !important;
        border: 0 !important;
        white-space: nowrap !important;
      }
      input[type="range"] {
        -webkit-appearance: none;
        appearance: none;
        background: transparent;
      }
      input[type="range"]::-webkit-slider-runnable-track {
        height: 6px;
        background: linear-gradient(to right, var(--brand) var(--progress, 0%%), var(--border) var(--progress, 0%%));
        border-radius: 3px;
      }
      input[type="range"]::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 14px;
        height: 14px;
        border-radius: 50%%;
        background: var(--brand);
        margin-top: -4px;
      }
    }
  </style>
</head>
<body>
  <canvas id="viewport" aria-label="Interactive 3D view of the Carousmell olfactory carousel"></canvas>

  <aside class="panel">
    <h1>Carousmell 3D CAD</h1>
    <p class="sub">Ø%.0f × %.1f mm · Creality K1C Support-Free</p>

    <div class="slider-row">
      <label for="explode">Exploded View</label>
      <span id="explodeVal">0%%</span>
      <input id="explode" type="range" min="0" max="100" value="0">
    </div>

    <div class="slider-row">
      <label for="cutaway">XZ Cutaway Slice (Y clip)</label>
      <span id="cutawayVal">Off</span>
      <input id="cutaway" type="range" min="-65" max="65" value="-65">
    </div>

    <div class="section">
      <div class="section-title">Printable Parts</div>
      <div id="partList"></div>
      <label class="part-item" style="margin-top:4px;">
        <input id="showBed" type="checkbox" checked>
        <span>Creality K1C Bed (220×220 mm)</span>
      </label>
    </div>

    <div class="section">
      <div class="section-title">Camera Views</div>
      <div class="btn-grid">
        <button type="button" data-view="iso">Isometric</button>
        <button type="button" data-view="cut">Cutaway</button>
        <button type="button" data-view="front">Exhaust +X</button>
        <button type="button" data-view="side">Airflow -Y</button>
        <button type="button" data-view="rear">Fan -X</button>
        <button type="button" data-view="top">Top Plan</button>
      </div>
    </div>
  </aside>

  <div class="hint">Drag: Orbit · Shift+Drag / Right-Drag: Pan · Scroll: Zoom</div>

  <script>
    const PARTS_DATA = %s;

    const canvas = document.getElementById("viewport");
    const gl = canvas.getContext("webgl", { antialias: true, alpha: false });
    if (!gl) {
      document.body.innerHTML = "<p style='padding:24px'>WebGL is required for the interactive 3D viewer.</p>";
      throw new Error("WebGL unavailable");
    }

    const vsSrc = ` + "`" + `
      attribute vec3 aPos;
      attribute vec3 aNorm;
      uniform mat4 uMVP;
      uniform float uExplodeZ;
      varying vec3 vWorldPos;
      varying vec3 vNorm;
      void main() {
        vec3 p = aPos + vec3(0.0, 0.0, uExplodeZ);
        vWorldPos = p;
        vNorm = aNorm;
        gl_Position = uMVP * vec4(p, 1.0);
      }
    ` + "`" + `;

    const fsSrc = ` + "`" + `
      precision mediump float;
      uniform vec3 uColor;
      uniform float uClipY;
      uniform vec3 uCamPos;
      varying vec3 vWorldPos;
      varying vec3 vNorm;
      void main() {
        if (vWorldPos.y < uClipY) discard;
        vec3 n = normalize(vNorm);
        if (!gl_FrontFacing) n = -n;
        vec3 l1 = normalize(vec3(0.55, -0.70, 0.85));
        vec3 l2 = normalize(vec3(-0.65, 0.55, 0.45));
        vec3 v = normalize(uCamPos - vWorldPos);
        float d1 = max(dot(n, l1), 0.0);
        float d2 = max(dot(n, l2), 0.0);
        float hemi = 0.5 + 0.5 * n.z;
        vec3 h = normalize(l1 + v);
        float spec = pow(max(dot(n, h), 0.0), 32.0) * 0.22;
        vec3 col = uColor * (0.28 * hemi + 0.62 * d1 + 0.24 * d2) + vec3(spec);
        if (!gl_FrontFacing) col *= 0.82;
        gl_FragColor = vec4(pow(clamp(col, 0.0, 1.0), vec3(1.0 / 1.25)), 1.0);
      }
    ` + "`" + `;

    const lineVS = ` + "`" + `
      attribute vec3 aPos;
      uniform mat4 uMVP;
      void main() { gl_Position = uMVP * vec4(aPos, 1.0); }
    ` + "`" + `;
    const lineFS = ` + "`" + `
      precision mediump float;
      uniform vec4 uColor;
      void main() { gl_FragColor = uColor; }
    ` + "`" + `;

    function compile(type, src) {
      const sh = gl.createShader(type);
      gl.shaderSource(sh, src);
      gl.compileShader(sh);
      if (!gl.getShaderParameter(sh, gl.COMPILE_STATUS)) throw new Error(gl.getShaderInfoLog(sh));
      return sh;
    }
    function link(vs, fs) {
      const pr = gl.createProgram();
      gl.attachShader(pr, compile(gl.VERTEX_SHADER, vs));
      gl.attachShader(pr, compile(gl.FRAGMENT_SHADER, fs));
      gl.linkProgram(pr);
      if (!gl.getProgramParameter(pr, gl.LINK_STATUS)) throw new Error(gl.getProgramInfoLog(pr));
      return pr;
    }

    const meshProg = link(vsSrc, fsSrc);
    const lineProg = link(lineVS, lineFS);

    async function decodeGzipBase64Bytes(b64) {
      const bin = atob(b64);
      const compressed = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) compressed[i] = bin.charCodeAt(i);
      const stream = new Blob([compressed]).stream().pipeThrough(new DecompressionStream("gzip"));
      return new Uint8Array(await new Response(stream).arrayBuffer());
    }

    let gpuParts = [];
    async function initParts() {
      gpuParts = await Promise.all(PARTS_DATA.map(async (p) => {
        const bytes = await decodeGzipBase64Bytes(p.posB64);
        const u16 = new Uint16Array(bytes.buffer);
        const vertCount = p.triCount * 3;
        const pos = new Float32Array(vertCount * 3);
        const norm = new Float32Array(vertCount * 3);
        for (let t = 0; t < p.triCount; t++) {
          const base = t * 9;
          for (let k = 0; k < 3; k++) {
            const i = base + k * 3;
            pos[i]     = p.min[0] + u16[i]     * p.scale[0];
            pos[i + 1] = p.min[1] + u16[i + 1] * p.scale[1];
            pos[i + 2] = p.min[2] + u16[i + 2] * p.scale[2];
          }
          const ax = pos[base + 3] - pos[base], ay = pos[base + 4] - pos[base + 1], az = pos[base + 5] - pos[base + 2];
          const bx = pos[base + 6] - pos[base], by = pos[base + 7] - pos[base + 1], bz = pos[base + 8] - pos[base + 2];
          let nx = ay * bz - az * by, ny = az * bx - ax * bz, nz = ax * by - ay * bx;
          const len = Math.hypot(nx, ny, nz) || 1;
          nx /= len; ny /= len; nz /= len;
          for (let k = 0; k < 3; k++) {
            const i = base + k * 3;
            norm[i] = nx; norm[i + 1] = ny; norm[i + 2] = nz;
          }
        }
        const posBuf = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, posBuf);
        gl.bufferData(gl.ARRAY_BUFFER, pos, gl.STATIC_DRAW);
        const normBuf = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, normBuf);
        gl.bufferData(gl.ARRAY_BUFFER, norm, gl.STATIC_DRAW);
        return { ...p, visible: true, vertCount, posBuf, normBuf };
      }));

      // Populate UI part checkboxes
      const partList = document.getElementById("partList");
      gpuParts.forEach((p) => {
        const label = document.createElement("label");
        label.className = "part-item";
        const cb = document.createElement("input");
        cb.type = "checkbox";
        cb.checked = true;
        cb.addEventListener("change", () => { p.visible = cb.checked; requestRender(); });
        const sw = document.createElement("span");
        sw.className = "swatch";
        sw.style.background = "rgb(" + p.color.map(c => Math.round(c * 255)).join(",") + ")";
        const txt = document.createElement("span");
        txt.textContent = p.name;
        const meta = document.createElement("span");
        meta.className = "part-meta";
        meta.textContent = p.material.split(" ")[0];
        label.title = p.description + " — " + p.printNote;
        label.append(cb, sw, txt, meta);
        partList.appendChild(label);
      });
      requestRender();
    }

    // Creality K1C 220x220 mm build plate grid on Z = 0
    const gridVerts = [];
    for (let g = -110; g <= 110; g += 10) {
      gridVerts.push(g, -110, 0, g, 110, 0);
      gridVerts.push(-110, g, 0, 110, g, 0);
    }
    const gridBuf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, gridBuf);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(gridVerts), gl.STATIC_DRAW);

    // Camera & interactive state
    let yaw = -0.82, pitch = 0.44, dist = 310;
    let target = [0, 0, 74];
    let explode = 0.0, clipY = -65.0;
    let showBed = true;

    const explodeInput = document.getElementById("explode");
    const explodeVal = document.getElementById("explodeVal");
    const cutawayInput = document.getElementById("cutaway");
    const cutawayVal = document.getElementById("cutawayVal");
    const showBedInput = document.getElementById("showBed");

    explodeInput.addEventListener("input", () => {
      explode = Number(explodeInput.value) / 100;
      explodeVal.textContent = explodeInput.value + "%%";
      explodeInput.style.setProperty("--progress", explodeInput.value + "%%");
      requestRender();
    });
    cutawayInput.addEventListener("input", () => {
      clipY = Number(cutawayInput.value);
      cutawayVal.textContent = clipY <= -64 ? "Off" : ("Y ≥ " + clipY + " mm");
      const pct = ((clipY + 65) / 130) * 100;
      cutawayInput.style.setProperty("--progress", pct + "%%");
      requestRender();
    });
    showBedInput.addEventListener("change", () => { showBed = showBedInput.checked; requestRender(); });

    document.querySelectorAll("button[data-view]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const v = btn.dataset.view;
        target = [0, 0, 74 + explode * 65];
        if (v === "iso")   { yaw = -0.82; pitch = 0.44; dist = 310; clipY = -65; cutawayInput.value = -65; cutawayVal.textContent = "Off"; }
        if (v === "cut")   { yaw = -1.57; pitch = 0.16; dist = 280; clipY = 0;   cutawayInput.value = 0;   cutawayVal.textContent = "Y ≥ 0 mm"; }
        if (v === "front") { yaw = 0.0;   pitch = 0.15; dist = 290; }
        if (v === "side")  { yaw = -1.57; pitch = 0.08; dist = 290; }
        if (v === "rear")  { yaw = 3.14;  pitch = 0.30; dist = 290; }
        if (v === "top")   { yaw = -1.57; pitch = 1.48; dist = 320; }
        requestRender();
      });
    });

    // Pointer orbit / pan / zoom
    let dragging = false, panMode = false, lastX = 0, lastY = 0;
    canvas.addEventListener("contextmenu", (e) => e.preventDefault());
    canvas.addEventListener("pointerdown", (e) => {
      dragging = true;
      panMode = e.button === 2 || e.shiftKey;
      lastX = e.clientX; lastY = e.clientY;
      canvas.setPointerCapture(e.pointerId);
    });
    canvas.addEventListener("pointerup", () => { dragging = false; });
    canvas.addEventListener("pointermove", (e) => {
      if (!dragging) return;
      const dx = e.clientX - lastX, dy = e.clientY - lastY;
      lastX = e.clientX; lastY = e.clientY;
      if (panMode) {
        const scale = dist * 0.0014;
        const rx = -Math.sin(yaw), ry = Math.cos(yaw);
        target[0] -= rx * dx * scale;
        target[1] -= ry * dx * scale;
        target[2] += dy * scale;
      } else {
        yaw -= dx * 0.007;
        pitch = Math.max(-1.45, Math.min(1.52, pitch + dy * 0.007));
      }
      requestRender();
    });
    canvas.addEventListener("wheel", (e) => {
      e.preventDefault();
      dist = Math.max(80, Math.min(900, dist * (1 + Math.sign(e.deltaY) * 0.09)));
      requestRender();
    }, { passive: false });

    // Minimal 4x4 matrix math (column-major for WebGL)
    function perspective(fovRad, aspect, near, far) {
      const f = 1 / Math.tan(fovRad / 2), nf = 1 / (near - far);
      return new Float32Array([
        f / aspect, 0, 0, 0,
        0, f, 0, 0,
        0, 0, (far + near) * nf, -1,
        0, 0, 2 * far * near * nf, 0
      ]);
    }
    function lookAt(eye, ctr, up) {
      let zx = eye[0] - ctr[0], zy = eye[1] - ctr[1], zz = eye[2] - ctr[2];
      let zl = Math.hypot(zx, zy, zz) || 1; zx /= zl; zy /= zl; zz /= zl;
      let xx = up[1] * zz - up[2] * zy, xy = up[2] * zx - up[0] * zz, xz = up[0] * zy - up[1] * zx;
      let xl = Math.hypot(xx, xy, xz) || 1; xx /= xl; xy /= xl; xz /= xl;
      const yx = zy * xz - zz * xy, yy = zz * xx - zx * xz, yz = zx * xy - zy * xx;
      return new Float32Array([
        xx, yx, zx, 0,
        xy, yy, zy, 0,
        xz, yz, zz, 0,
        -(xx*eye[0] + xy*eye[1] + xz*eye[2]),
        -(yx*eye[0] + yy*eye[1] + yz*eye[2]),
        -(zx*eye[0] + zy*eye[1] + zz*eye[2]),
        1
      ]);
    }
    function mulMat4(a, b) {
      const out = new Float32Array(16);
      for (let c = 0; c < 4; c++) {
        for (let r = 0; r < 4; r++) {
          out[c*4 + r] = a[r]*b[c*4] + a[4+r]*b[c*4+1] + a[8+r]*b[c*4+2] + a[12+r]*b[c*4+3];
        }
      }
      return out;
    }

    const darkQuery = window.matchMedia("(prefers-color-scheme: dark)");
    let isDark = darkQuery.matches;
    darkQuery.addEventListener("change", (e) => { isDark = e.matches; requestRender(); });

    let pending = false;
    function requestRender() {
      if (!pending) { pending = true; requestAnimationFrame(render); }
    }

    function render() {
      pending = false;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const w = Math.floor(canvas.clientWidth * dpr);
      const h = Math.floor(canvas.clientHeight * dpr);
      if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; }

      gl.viewport(0, 0, w, h);
      if (isDark) gl.clearColor(0.06, 0.09, 0.16, 1.0);
      else gl.clearColor(0.95, 0.97, 0.99, 1.0);
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
      gl.enable(gl.DEPTH_TEST);

      const camPos = [
        target[0] + dist * Math.cos(pitch) * Math.cos(yaw),
        target[1] + dist * Math.cos(pitch) * Math.sin(yaw),
        target[2] + dist * Math.sin(pitch)
      ];
      const mvp = mulMat4(perspective(0.62, w / Math.max(h, 1), 1.0, 2000.0), lookAt(camPos, target, [0, 0, 1]));

      if (showBed) {
        gl.useProgram(lineProg);
        gl.uniformMatrix4fv(gl.getUniformLocation(lineProg, "uMVP"), false, mvp);
        gl.uniform4f(gl.getUniformLocation(lineProg, "uColor"), 0.5, 0.58, 0.68, 0.35);
        gl.bindBuffer(gl.ARRAY_BUFFER, gridBuf);
        const loc = gl.getAttribLocation(lineProg, "aPos");
        gl.enableVertexAttribArray(loc);
        gl.vertexAttribPointer(loc, 3, gl.FLOAT, false, 0, 0);
        gl.drawArrays(gl.LINES, 0, gridVerts.length / 3);
      }

      gl.useProgram(meshProg);
      gl.uniformMatrix4fv(gl.getUniformLocation(meshProg, "uMVP"), false, mvp);
      gl.uniform1f(gl.getUniformLocation(meshProg, "uClipY"), clipY <= -64 ? -1e6 : clipY);
      gl.uniform3fv(gl.getUniformLocation(meshProg, "uCamPos"), new Float32Array(camPos));
      const aPos = gl.getAttribLocation(meshProg, "aPos");
      const aNorm = gl.getAttribLocation(meshProg, "aNorm");
      const uColor = gl.getUniformLocation(meshProg, "uColor");
      const uExplodeZ = gl.getUniformLocation(meshProg, "uExplodeZ");

      for (const p of gpuParts) {
        if (!p.visible) continue;
        gl.uniform3fv(uColor, new Float32Array(p.color));
        gl.uniform1f(uExplodeZ, p.explodeZ * explode);

        gl.bindBuffer(gl.ARRAY_BUFFER, p.posBuf);
        gl.enableVertexAttribArray(aPos);
        gl.vertexAttribPointer(aPos, 3, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ARRAY_BUFFER, p.normBuf);
        gl.enableVertexAttribArray(aNorm);
        gl.vertexAttribPointer(aNorm, 3, gl.FLOAT, false, 0, 0);

        gl.drawArrays(gl.TRIANGLES, 0, p.vertCount);
      }
    }

    window.addEventListener("resize", requestRender);
    initParts();
  </script>
</body>
</html>
`
