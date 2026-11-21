# Carousmell — Programmatic CAD Design Plan

This document specifies the mechanical architecture, dimensional constraints, tolerances, and Go (`soypat/gsdf`) software structure for generating the 3D-printable parts of **Carousmell** (`github.com/janpfeifer/carousmell`).

It is intended both as a human-readable design reference and as a verified specification matching the implemented Go CAD generator under `cad/`.

---

## 1. Design Goals & Constraints

1. **100% Parametric in Go:** All dimensions, clearances, and hardware cutouts live in a single `cad.Params` struct (`cad/params.go`) so tolerances can be tuned after caliper measurements and calibration prints.
2. **100% Support-Free FDM Printing:** Optimized for a **Creality K1C** (0.4 mm nozzle, 220 × 220 × 250 mm build volume, enclosed chamber). Every printable `.stl` is exported flat on `Z = 0` in its optimal support-free print orientation (e.g., `lid_labyrinth` and `motor_deck` flipped upside-down).
3. **Zero-Friction, Oil-Resistant Mechanicals:**
   * Non-contact labyrinth air-gap seal (`0.60 mm` radial, `0.60 mm` vertical) between rotating carousel and stationary lid (no silicone drag, no grease).
   * Direct-drive 11-slot carousel mounted straight onto the 28BYJ-48 stepper shaft, with a blind double-D bore acting as a positive Z-stop on the shaft tip.
   * Upstream "push" airflow so essential oil vapors never pass through the fan bearings or blades.
4. **Material Separation:**
   * **Base & Motor Deck (electronics):** Printable in **Hyper PLA** (sealed below the vapor-barrier `motor_deck`).
   * **Carousel, Labyrinth Lid & Fan Duct Hood (scent path):** Prototype in **Hyper PLA**; final print in **PETG** (resists citrus *limonene* and clove *eugenol* vapors).
5. **Output Formats & Visualizations:**
   * Binary **`.stl`** files (millimeter units) exported per printable part via `gsdf/glrender.WriteBinarySTL` (`cad/out/*.stl`) plus a combined `assembly_preview.stl`.
   * Shaded 3D PNG renders (`docs/render_isometric.png`, `docs/render_cutaway.png`, `docs/render_exploded.png`), exact SDF 2D cross-section blueprint SVG (`docs/cross_section.svg`), and a standalone interactive 3D WebGL viewer (`docs/viewer.html`).

---

## 2. Coordinate System & Global Layout

All parts share a unified Z-up assembled coordinate system (all units in **millimeters**):

* **Origin `(0, 0, Z)`:** Centered on the **28BYJ-48 stepper motor shaft axis** (which is also the rotational center of the carousel and the unified $\varnothing 120\text{ mm}$ outer cylindrical housing).
  * **CRITICAL:** The 28BYJ-48 stepper's output shaft is **offset by 8.0 mm** from the center of its 28 mm cylindrical metal body. Therefore, the stepper's *shaft* sits at `(0, 0)`, and the stepper's *body center* sits at `(0, -8.0)` (blue plastic wire block pointing $-Y$ toward the ULN2003 driver board, keeping the entire $+Y$ half open for the Raspberry Pi Zero 2 WH).
* **`Z = 0`:** Bottom exterior face of `base` resting on the nightstand.
* **Angular Convention ($\theta$):**
  * **Active Exhaust / Front:** $\theta = 0^\circ$ ($+X$ axis). The single lid aperture and flared exhaust nozzle face $+X$.
  * **Upstream Fan / Rear:** $\theta = 180^\circ$ ($-X$ axis), tilted at **$45^\circ$** pushing clean room air from $-X$ toward $+X$ down through the lid aperture and horizontally across the active wick.
  * **Raspberry Pi Side:** $\theta = +90^\circ$ ($+Y$ axis), with the Micro-USB PWR IN wall cutout at $X = +21.5\text{ mm}$.
  * **Manual "Whiff Now" Button:** $\theta = -55^\circ$ (front-right, facing user, clear of the ULN2003 and perimeter bolts).
  * **Perimeter M3 Bolt Circle ($R = 53.0\text{ mm}$):** 4 symmetric positions at $\theta = \pm 25^\circ, \pm 140^\circ$ (verified clear of the Pi Zero, ULN2003, button, exhaust tunnel, carousel, and Noctua fan).
  * **Slot Spacing:** $N = 11$ slots evenly distributed at $\Delta\theta = \frac{360^\circ}{11} \approx 32.727^\circ$.
    * **Slot 0 ($\theta = 0^\circ$ when parked):** Empty "park" slot carrying the embedded $5 \times 2\text{ mm}$ homing magnet underneath directly above the KY-024 Hall sensor at `(+32.0, 0)`.
    * **Slots 1–10:** Essential oil vials.

---

## 3. Hardware Envelope & Reference Dimensions

> **Note:** Treat nominal values marked *(verify)* as defaults in `cad/params.go` and update them with caliper measurements once physical parts arrive.

| Component | Key Dimensions (Nominal) | Mounting / Cutout Notes |
| :--- | :--- | :--- |
| **2 ml Amber Vial (ALWSCI)** *(verify)* | OD $\approx 11.6\text{–}12.0\text{ mm}$, body height $\approx 32.0\text{ mm}$, total height with open-top cap $\approx 33.5\text{ mm}$ | Drop-in pocket $\varnothing 12.6\text{ mm}$ ($+0.6\text{ mm}$ diametral clearance) with $\ge 2.2\text{ mm}$ PETG walls all around. |
| **3 mm Fiber Wick** | $\varnothing 3.0\text{ mm}$, cut to $\sim 40\text{ mm}$ | Protrudes $8.0\text{ mm}$ above vial cap ($3.1\text{ mm}$ clearance below stationary lid ceiling). |
| **28BYJ-48 Stepper Motor** | Body $\varnothing 28.0\text{ mm} \times 19.0\text{ mm}$ tall; blue wire housing protrudes $\sim 17.5\text{ mm}$ from body center; mounting ears $35.0\text{ mm}$ c-to-c ($\varnothing 4.2\text{ mm}$ holes). | **Shaft:** $\varnothing 5.0\text{ mm}$ with double-D flats ($3.0\text{ mm}$ AF, $10.0\text{ mm}$ total height above flange). Bolts to underside of `motor_deck` with countersunk M3 screws flush below `Z = 42.0 mm`. |
| **ULN2003 Driver Board** *(verify)* | $\sim 35.0 \times 32.0 \times 12.0\text{ mm}$ | Centered at `(-8.0, -33.0)` on $3.0\text{ mm}$ standoffs ($29 \times 26\text{ mm}$ holes), directly adjacent to the stepper wire block. |
| **Raspberry Pi Zero 2 WH** | PCB $65.0 \times 30.0 \times 1.5\text{ mm}$; M2.5 holes at $58.0 \times 23.0\text{ mm}$; 40-pin header $\sim 8.5\text{ mm}$ tall. | Centered at `(0, +29.5)` on $4.0\text{ mm}$ standoffs with M2.5 alignment pins + edge guide rail. Generous $18 \times 12\text{ mm}$ chamfered-arch wall cutout at $X = +21.5\text{ mm}$ accommodates molded Micro-USB plug boots. |
| **Noctua NF-A4x10 5V PWM Fan** | $40.0 \times 40.0 \times 10.0\text{ mm}$; mounting holes $32.0 \times 32.0\text{ mm}$ ($\varnothing 4.4\text{ mm}$). | Mounted on a **$45^\circ$ tilted face** with open rear intake cradle so the included **silicone anti-vibration pins** can be pulled cleanly from behind and room air enters unobstructed. |
| **KY-024 Hall-Effect Sensor** *(verify)* | PCB $\sim 35.0 \times 15.0\text{ mm}$; 1 × $\varnothing 3.0\text{ mm}$ hole. | Slides into underside cradle on `motor_deck` beneath Slot 0 `(+32.0, 0)`, under a $0.8\text{ mm}$ thin sensing diaphragm ($\sim 2.0\text{ mm}$ air gap to magnet while maintaining the vapor seal). |
| **Neodymium Homing Magnet** | $\varnothing 5.0 \times 2.0\text{ mm}$ disc | Press-fit/glue pocket ($\varnothing 5.2\text{ mm} \times 2.2\text{ mm}$ deep) on underside of Slot 0 (`carousel` floor is $3.5\text{ mm}$ thick so pocket never punches through). |
| **12 mm ATX Power Button** | M12 threaded barrel, hex nut, $\sim 15\text{ mm}$ bezel OD, $\sim 20\text{ mm}$ internal depth. | Mounted on a **flat recessed boss** at $\theta = -55^\circ$ ($3.0\text{ mm}$ wall, $\varnothing 12.3\text{ mm}$ hole with $45^\circ$ top teardrop relief so it prints support-free). |
| **M3 Fasteners** | Standard M3×10–12 mm screws & hex nuts ($5.5\text{ mm}$ AF, $2.4\text{ mm}$ thick). | Clearance hole: $\varnothing 3.4\text{ mm}$. Side-accessible captive hex-nut slots ($5.8\text{ mm}$ wide × $2.6\text{ mm}$ tall) avoid vertical overhangs. |

---

## 4. Mechanical Sub-Assemblies (5 Printable Parts + Calibration Coupon)

> **Key DFM Revisions Verified During CAD Design:**
> 1. **Split `motor_deck` out of `base`:** Combining a solid bottom floor and an upper stepper bridge into one part would create an unprintable $110\text{ mm}$ horizontal overhang and block wiring access. Splitting `motor_deck` into a flat removable divider plate makes both parts 100% support-free, seals electronics from oil vapors, and gives open-top wiring access.
> 2. **Unified $\varnothing 120\text{ mm}$ outer shell:** Matching `base`, `lid_labyrinth`, and `duct_hood` to $\varnothing 120\text{ mm}$ resolves the OD mismatch between the electronics base and the $\varnothing 81\text{ mm}$ carousel, providing annular room for M3 bosses, an isolated fan wire channel outside the vapor seal, and the full 40 mm Noctua fan silhouette.
> 3. **Strengthened carousel outer cup wall ($2.2\text{ mm}$):** Increased $R_{\text{carousel}}$ from $39.5\text{ mm}$ ($1.2\text{ mm}$ outer wall) to $40.5\text{ mm}$ ($2.2\text{ mm}$ outer wall) to meet the $\ge 2.0\text{ mm}$ vapor-tight PETG rule.
> 4. **Exhaust nozzle consolidated into `lid_labyrinth`:** The sealed, flared `18 → 22 mm` exhaust tunnel belongs at wick height on `lid_labyrinth` (rather than above the ceiling in `duct_hood`). Narrowing the inner labyrinth window to `18.0 mm` preserves $1.8^\circ$ of seal overlap on both flanking radial fins while flaring to `22.0 mm` at the exit.
> 5. **$45^\circ$ fan tilt & open rear cradle in `duct_hood`:** Changing from $40^\circ$ / closed dome to $45^\circ$ with an open rear cradle guarantees $\le 45^\circ$ overhangs (support-free on K1C), eliminates intake restriction, and gives finger access behind the fan plate to install Noctua's silicone pins.

### Part 1: `base.stl` (Electronics Housing Tub — Hyper PLA)
* **Print Orientation:** Right-side-up on `Z = 0` (`120.0 × 120.0 × 45.0 mm`). **100% support-free.**
* **Geometry:** Open-top $\varnothing 120\text{ mm}$ cylinder (`Z = 0..42.0 mm`, $3.6\text{ mm}$ wall, $1.0\text{ mm}$ bottom anti-elephant-foot chamfer) with a $45^\circ$-chamfered internal support ledge at `Z = 39.0 mm` for `motor_deck`, stepped upper alignment rim (`Z = 42.0..45.0 mm`), 4 perimeter M3 columns with side-inserted captive nut slots, Pi Zero 2 WH standoffs + $18 \times 12\text{ mm}$ arched Micro-USB cutout, ULN2003 standoffs, $\theta = -55^\circ$ M12 teardrop button boss, passive lower vent slots, and a rear vertical fan-wire channel.

### Part 2: `motor_deck.stl` (Stepper & Hall-Sensor Divider Plate — Hyper PLA / PETG)
* **Print Orientation:** Exported **upside-down** on `Z = 0` (`112.2 × 112.2 × 8.0 mm`) so the Hall-sensor cradle rails print straight up. **100% support-free.**
* **Geometry:** $3.0\text{ mm}$ circular plate (`Z = 39.0..42.0 mm`) bolting down onto `base` with 4 countersunk M3×10 mm screws. Mounts the 28BYJ-48 stepper underneath with its shaft centered at `(0, 0)` (countersunk M3 ear holes at `±17.5, -8.0`), holds the KY-024 Hall sensor under a $0.8\text{ mm}$ sensing window at `(+32.0, 0)`, and includes a rear fan-wire pass-through notch.

### Part 3: `carousel.stl` (11-Slot Rotor — PETG)
* **Print Orientation:** Right-side-up on `Z = 0` (`81.0 × 81.0 × 47.5 mm`). **100% support-free.**
* **Geometry:** $R_{\text{vial}} = 32.0\text{ mm}$, $R_{\text{in}} = 23.5\text{ mm}$, $R_{\text{out}} = 40.5\text{ mm}$, $3.5\text{ mm}$ base floor, $24.0\text{ mm}$ deep lower vial pockets ($\varnothing 12.6\text{ mm}$, $2.2\text{ mm}$ inner/outer radial walls, $5.43\text{ mm}$ inter-vial wall).
* **Central Hub (`(0, 0)`):** $\varnothing 14.0 \times 10.0\text{ mm}$ boss with a **$5.5\text{ mm}$ deep blind double-D bore** (`Ø5.20 mm`, `3.20 mm` AF at default `ShaftClearance = 0.20 mm`). Resting positively on the stepper shaft tip (`Z = 49.0 mm`) locks the carousel bottom at `Z = 43.5 mm` ($1.5\text{ mm}$ gap above `motor_deck`).
* **Slot 0 & Radial Fins:** Slot 0 ($\theta = 0^\circ$) is solid with an underside $\varnothing 5.2 \times 2.2\text{ mm}$ magnet pocket and top marker dimple. Eleven $2.2\text{ mm}$ thick radial divider fins rise to `Z_local = 47.5 mm` ($2.5\text{ mm}$ above exposed wick tips) with an open outer radial edge for horizontal cross-wick airflow.

### Part 4: `lid_labyrinth.stl` (Stationary Seal Plate, Outer Shell & Flared Exhaust Nozzle — PETG)
* **Print Orientation:** Exported **upside-down** on `Z = 0` (`128.0 × 120.0 × 52.6 mm`, flat ceiling resting directly on the K1C build plate). **100% support-free.**
* **Geometry:** Outer $\varnothing 120\text{ mm}$ shell (`Z = 42.0..94.6 mm`) with a bottom slip-fit alignment skirt + rear indexing lug so the lid + hood lifts off cleanly for refilling and always re-seats at exact $\theta = 0^\circ$.
* **Labyrinth Seal:** Ceiling underside (`Z = 91.6 mm`) clears the carousel fin tops (`Z = 91.0 mm`) by **`0.60 mm`**. Concentric inner (`R = 20.7..22.9 mm`, full $360^\circ$) and outer (`R = 41.1..43.3 mm`) $2.2\text{ mm}$ rings hang down to `Z = 69.0 mm` ($2.0\text{ mm}$ overlap past cup top) with **`0.60 mm`** radial air gaps.
* **Airflow Path:** Single $25^\circ$ sector Top Inlet Aperture (`R = 24.0..39.5 mm`) over $\theta = 0^\circ$ + sealed exhaust tunnel flaring from `18.0 mm` at the outer labyrinth ring to `22.0 mm` wide at the protruding front nozzle (`X = 68.0 mm`, `Z = 74.0..91.6 mm`, centered on the exposed wick).
* **Upper Bosses:** Four $45^\circ$-chamfered M3 bosses with captive nut slots bolt `duct_hood` down using M3×12 mm screws.

### Part 5: `duct_hood.stl` (Upstream 45° Noctua Fan Cradle & Curved Duct — PETG)
* **Print Orientation:** Right-side-up on `Z = 0` (`120.0 × 120.0 × ~45.8 mm`). **100% support-free.**
* **Geometry:** Base mounting flange (`Z = 94.6..97.8 mm`) with 4 M3 clearance holes, front/side trim rim (open rear intake), and a $45^\circ$ tilted `44 × 44 mm` Noctua NF-A4x10 mounting plate ($\varnothing 38.0\text{ mm}$ bore + 4 × $\varnothing 4.4\text{ mm}$ silicone-pin holes at `32 × 32 mm`) supported by dual side struts.
* **Smooth Curved Duct:** Cubic-Bezier tapered inner cavity offset uniformly by `DuctWall = 2.6 mm` smoothly turns fan airflow downward into the single lid aperture over the active wick.

### Part 6: `calibration_coupon.stl` (10-Minute Fit-Check Print)
* Compact `66 × 32 × 9.0 mm` flat plate generated from `cad.Params` containing:
  1. 3 notched **double-D shaft bores** (`ShaftClearance - 0.10`, `+0.00`, `+0.10 mm`),
  2. 1 **2 ml vial pocket** ($\varnothing 12.6\text{ mm}$),
  3. 1 **5 × 2 mm magnet recess** ($\varnothing 5.2 \times 2.2\text{ mm}$),
  4. 1 **M3 clearance hole** ($\varnothing 3.4\text{ mm}$) + underside **M3 hex nut trap** (`5.8 mm` AF),
  5. 1 **M12 button hole** ($\varnothing 12.3\text{ mm}$).

---

## 5. Creality K1C Printing & DFM Guidelines

| Parameter | Creality K1C Guideline | Implemented CAD Rule |
| :--- | :--- | :--- |
| **Build Volume** | $220 \times 220 \times 250\text{ mm}$ | Largest part is `lid_labyrinth` (`128 × 120 × 52.6 mm`); assembled device is `128 × 120 × 140 mm`. |
| **Nozzle & Wall Width** | $0.4\text{ mm}$ nozzle ($0.42\text{ mm}$ line width) | All structural & vapor-exposed walls are $\ge 2.2\text{ mm}$ ($\approx 5$ perimeter lines). |
| **Overhangs** | Clean up to $45^\circ\text{–}50^\circ$ with K1C cooling | All ledges, boss undersides, and the fan duct use $\le 45^\circ$ angles; `lid_labyrinth` & `motor_deck` export upside-down. |
| **Horizontal Holes** | Slight top sag on circles | M12 button hole has a $45^\circ$ top teardrop peak; Micro-USB & vent slots have $45^\circ$ chamfered arches. |
| **XY Tolerance** | K1C prints internal holes $0.1\text{–}0.2\text{ mm}$ undersized | All clearances live in `cad/params.go`; verify first with `calibration_coupon.stl` in PLA and PETG. |
| **PETG Slicer Settings** | Enclosed chamber (cracked lid for PLA; closed for PETG) | 3–4 wall loops, 15–20% gyroid infill, $0.20\text{ mm}$ layer height ($0.16\text{ mm}$ recommended for `carousel` hub and `duct_hood`). |

---

## 6. Go (`soypat/gsdf`) Software Architecture

```text
cad/
├── params.go             # Single source of truth for all dimensions & tolerances (mm)
├── helpers.go            # Reusable gsdf primitives (double-D bore, hex trap, sector, teardrop, tapered tunnel)
├── base.go               # Part 1: Open-top Hyper PLA electronics tub
├── motor_deck.go         # Part 2: Removable stepper & KY-024 Hall-sensor divider plate
├── carousel.go           # Part 3: 11-slot PETG rotor with radial fins & magnet recess
├── lid.go                # Part 4: Stationary PETG labyrinth lid, outer shell & flared exhaust nozzle
├── duct.go               # Part 5: 45° Noctua 40 mm fan cradle & smooth cubic-Bezier turning duct
├── calibration.go        # Part 6: Quick K1C fit-check coupon
├── parts.go              # Part registry, print-orientation transforms, hardware silhouettes & assembly
├── export.go             # Parallel CPU marching-cubes STL exporter (FlatRenderer / Octree -> binary STL)
├── render_png.go         # Pure-Go parallel CPU SDF raymarcher (isometric, cutaway, exploded PNGs)
├── render_svg.go         # Pure-Go exact SDF 2D cross-section & floorplan SVG blueprint
├── render_html.go        # Standalone interactive 3D WebGL HTML viewer (gzip-compressed embedded meshes)
├── geometry_test.go      # Automated SDF verification tests (clearances, airflow, K1C build volume)
└── cmd/
    ├── generate/main.go  # CLI: generates STLs, PNGs, SVG, viewer.html, and optional --serve UI
    └── viewer/           # Local browser 3D UI (default) or native WebGPU UI (-tags webgpu)
```

### Usage Commands

```bash
# 1. Run automated SDF geometric verification (clearances, airflow, K1C fit)
go test -v ./cad/...

# 2. Generate fast preview STLs (0.50 mm) + README PNG/SVG renders + interactive HTML viewer
go run ./cad/cmd/generate

# 3. Export high-resolution print-ready STLs (0.20 mm) for Creality Print / OrcaSlicer
go run ./cad/cmd/generate --res 0.20 --no-visuals

# 4. Launch the local interactive 3D CAD viewer UI in your browser
go run ./cad/cmd/viewer
# (Or on systems with GLFW/Wayland headers, native GPU raymarching via gsdfaux.UI:)
# go run -tags webgpu ./cad/cmd/viewer --part assembly --cutaway
```

---

## 7. Recommended Build & Iteration Workflow

1. **Measure real hardware** with calipers (vial OD/height, 28BYJ-48 shaft flats, KY-024 PCB, ULN2003 PCB) and update `cad/params.go`.
2. **Run `go test -v ./cad/...`** to confirm zero interference and full airflow clearance after any parameter tweak.
3. **Print `cad/out/calibration_coupon.stl`** on the Creality K1C (Hyper PLA first, then PETG) and dial in `ShaftClearance`, `VialClearance`, and `M3NutAF`.
4. **Inspect `docs/viewer.html`** (or `cad/out/assembly_preview.stl` in Creality Print / OrcaSlicer) using the live XZ cutaway and exploded sliders.
5. **Print full prototype in Hyper PLA** → bench-test homing, rotation, airflow, and button mounting.
6. **Reprint `carousel.stl`, `lid_labyrinth.stl`, and `duct_hood.stl` in PETG** (at `--res 0.20`) for the final oil-resistant unit.
