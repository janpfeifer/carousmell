# Carousmell — Programmatic CAD Design Plan

This document specifies the mechanical architecture, dimensional constraints, tolerances, and Go (`soypat/gsdf`) software structure for generating the 3D-printable parts of **Carousmell** (`github.com/janpfeifer/carousmell`).

It is intended both as a human-readable design reference and as a complete specification for programmatic CAD generation.

---

## 1. Design Goals & Constraints

1. **100% Parametric in Go:** All dimensions, clearances, and hardware cutouts live in a single parameter struct so tolerances can be tuned after caliper measurements and test prints.
2. **Support-Free (or Near Support-Free) FDM Printing:** Optimized for a **Creality K1C** (0.4 mm nozzle, 220 × 220 × 250 mm build volume, enclosed chamber).
3. **Zero-Friction, Oil-Resistant Mechanicals:**
   * Non-contact labyrinth air-gap seal between rotating carousel and stationary lid (no silicone drag, no grease).
   * Direct-drive 11-slot carousel mounted straight onto the 28BYJ-48 stepper shaft.
   * Upstream "push" airflow so essential oil vapors never pass through the fan bearings or blades.
4. **Material Separation:**
   * **Base (electronics):** Printable in **Hyper PLA** (isolated from essential oil vapors).
   * **Carousel & Lid/Duct (scent path):** Prototype in **Hyper PLA**; final print in **PETG** (resists citrus *limonene* and clove *eugenol* vapors).
5. **Output Format:** **`.3mf`** with explicit millimeter units (`go3mf.UnitMillimeter`), exported as individual files per printable part (plus an optional combined assembly `.3mf` for visual verification).

---

## 2. Coordinate System & Global Layout

All parts share a unified Z-up coordinate system (all units in **millimeters**):

* **Origin `(0, 0, Z)`:** Centered on the **28BYJ-48 stepper motor shaft axis** (which is also the rotational center of the carousel and the outer cylindrical housing).
  * **CRITICAL:** The 28BYJ-48 stepper's output shaft is **offset by 8.0 mm** from the center of its 28 mm cylindrical metal body. Therefore, the stepper's *shaft* sits at `(0, 0)`, and the stepper's *body center* sits at `(0, -8.0)` (or rotated to whichever angle best clears the Raspberry Pi).
* **`Z = 0`:** Bottom exterior face of the base resting on the nightstand.
* **Angular Convention ($\theta$):**
  * **Active Exhaust / Front:** $\theta = 0^\circ$ ($+X$ axis). The single lid aperture and exhaust nozzle face $+X$.
  * **Upstream Fan / Rear:** $\theta = 180^\circ$ ($-X$ axis), tilted upward at $40^\circ$ pushing air from $-X$ toward $+X$ across the active wick.
  * **Slot Spacing:** $N = 11$ slots evenly distributed at $\Delta\theta = \frac{360^\circ}{11} \approx 32.727^\circ$.
    * **Slot 0:** Empty "park" slot (carries the embedded 5 × 2 mm homing magnet underneath).
    * **Slots 1–10:** Essential oil vials.

---

## 3. Hardware Envelope & Reference Dimensions

> **Note:** Treat nominal values marked *(verify)* as defaults in `params.go` and update them with caliper measurements once physical parts arrive.

| Component | Key Dimensions (Nominal) | Mounting / Cutout Notes |
| :--- | :--- | :--- |
| **2 ml Amber Vial (ALWSCI)** *(verify)* | OD $\approx 11.6\text{–}12.0\text{ mm}$, body height $\approx 32.0\text{ mm}$, total height with open-top cap $\approx 33.5\text{ mm}$ | Drop-in pocket with $+0.5\text{ mm}$ radial clearance (e.g., $\varnothing 12.6\text{ mm}$ pocket) so vials lift out effortlessly. |
| **3 mm Fiber Wick** | $\varnothing 3.0\text{ mm}$, cut to $\sim 40\text{ mm}$ | Protrudes $7\text{–}8\text{ mm}$ above vial cap. |
| **28BYJ-48 Stepper Motor** | Body $\varnothing 28.0\text{ mm} \times 19.0\text{ mm}$ tall; blue plastic wire housing protrudes $\sim 17.5\text{ mm}$ radially from body center; mounting ears $35.0\text{ mm}$ center-to-center ($\varnothing 4.2\text{ mm}$ holes). | **Shaft:** $\varnothing 5.0\text{ mm}$ with double-D flats ($3.0\text{ mm}$ across flats, $\sim 6.0\text{ mm}$ flat length, $\sim 8.5\text{ mm}$ shaft height above boss). Mount with M3 screws + captive nuts. |
| **ULN2003 Driver Board** *(verify)* | $\sim 35.0 \times 32.0 \times 12.0\text{ mm}$ (incl. connector pins) | 4 mounting holes ($\varnothing 3.0\text{ mm}$) or vertical slide-in PCB slot in the base. |
| **Raspberry Pi Zero 2 WH** | PCB $65.0 \times 30.0 \times 1.5\text{ mm}$; M2.5 holes at $58.0 \times 23.0\text{ mm}$; 40-pin header $\sim 8.5\text{ mm}$ tall; optional heatsink $14.3 \times 14.0 \times 6.0\text{ mm}$. | **Do not use M3 screws** (holes are M2.5). Use 3D-printed standoffs ($3\text{–}4\text{ mm}$ high) + snap-fit retaining clips. Expose Micro-USB power port through base wall. Keep stepper body away from the PCB Wi-Fi antenna corner. |
| **Noctua NF-A4x10 5V PWM Fan** | $40.0 \times 40.0 \times 10.0\text{ mm}$; mounting holes $32.0 \times 32.0\text{ mm}$ spacing ($\varnothing 4.3\text{ mm}$). | Design for the included **silicone anti-vibration pins** (or M3 screws/nuts) on a $40^\circ$ angled intake face. Add a wire-routing notch for the 4-pin cable. |
| **KY-024 Hall-Effect Sensor** *(verify)* | PCB $\sim 35.0 \times 15.0\text{ mm}$ (sensor IC extends from front tip); 1 × $\varnothing 3.0\text{ mm}$ hole. | Mount underneath the carousel so the Hall IC sits directly below the magnet's pitch circle ($R_{\text{magnet}}$) with $1.5\text{–}2.5\text{ mm}$ vertical air gap. |
| **Neodymium Homing Magnet** | $\varnothing 5.0 \times 2.0\text{ mm}$ disc | Press-fit/glue pocket ($\varnothing 5.2\text{ mm} \times 2.2\text{ mm}$ deep) on the underside of Slot 0. |
| **12 mm ATX Power Button (HAAMNING7)** | M12 threaded barrel, hex nut, $\sim 15\text{ mm}$ bezel OD, $\sim 20\text{ mm}$ internal depth behind panel. | Mount on a **flat recessed boss** on the outer base wall: $\varnothing 12.3\text{ mm}$ hole, wall thickness $2.5\text{–}3.5\text{ mm}$ so the M12 hex nut threads cleanly inside. |
| **M3 Fasteners** | M3 socket-head screws (various lengths) & hex nuts ($5.5\text{ mm}$ across flats, $2.4\text{ mm}$ thick). | Clearance hole: $\varnothing 3.4\text{ mm}$. Captive hex-nut trap: $5.8\text{ mm}$ across flats, $2.6\text{ mm}$ deep. |

---

## 4. Mechanical Sub-Assemblies (Printable Parts)

To keep printing **support-free** and allow separate PLA/PETG material choices, split the enclosure into **4 modular printable parts** (plus 1 calibration coupon):

### Part 1: `base` (Electronics Housing — Hyper PLA)
* **Overall Footprint:** Cylinder $\varnothing 110\text{–}120\text{ mm}$ (exact OD derived parametrically from the carousel pitch radius + wall thickness), height $\sim 38\text{–}45\text{ mm}$ to comfortably house the Pi Zero 2 WH, Dupont cable bend radius ($\sim 15\text{ mm}$ above the 40-pin header), ULN2003 board, and coiled button wire.
* **Stepper Mount Bridge:** An internal horizontal shelf/boss supporting the 28BYJ-48 stepper motor at the top of the base, with its **shaft centered at `(0, 0)`** and two M3 captive nut traps spaced $35.0\text{ mm}$ apart.
* **KY-024 Sensor Mount:** A small bracket or slide-in cradle positioned directly beneath the carousel's magnet orbit radius ($R_{\text{magnet}}$), with a small window in the top floor of the base if needed so the Hall IC is within $2\text{ mm}$ of the passing magnet.
* **Raspberry Pi Zero 2 WH Cradle:** Snap-fit clips + $4\text{ mm}$ standoffs on the base floor, positioned with the Micro-USB **PWR IN** port aligned to a generous wall cutout ($14 \times 10\text{ mm}$ to accommodate molded USB plug boots).
* **Manual "Whiff Now" Button Flat:** A flattened exterior facet on the base cylinder (e.g., at $\theta = -30^\circ$ or $+30^\circ$, facing the user) with a $\varnothing 12.3\text{ mm}$ hole and $3.0\text{ mm}$ wall thickness.
* **Ventilation & Cable Routing:** Subtle bottom/side vent slots for passive cooling of the Pi, plus a recessed vertical wire channel along the inner wall for routing the Noctua fan's 4-pin cable down from the top hood into the base.
* **Top Interface:** A stepped alignment lip and 3 or 4 M3 mounting tabs/bosses around the perimeter to bolt the upper enclosure / stationary lid cleanly to the base.

### Part 2: `carousel` (11-Slot Rotor — PETG)
* **Pitch Circle Radius ($R_{\text{vial}}$):**
  * With 11 vials of $\varnothing 12.6\text{ mm}$ pocket + $2.0\text{ mm}$ radial divider walls between them, minimum circumference at the pocket center is $\approx 11 \times (12.6 + 2.0) \approx 160.6\text{ mm} \implies R_{\text{vial}} \ge 25.6\text{ mm}$.
  * **Recommended $R_{\text{vial}} = 32.0\text{ mm}$** (gives a comfortable $\sim 18.3\text{ mm}$ center-to-center arc spacing between adjacent vials, plenty of room for sturdy radial fins and easy finger grip when refilling).
  * Outer radius of carousel pockets/fins: $R_{\text{carousel}} \approx 32.0 + 7.5 = 39.5\text{ mm}$ ($\varnothing \approx 79\text{ mm}$).
* **Central Direct-Drive Hub:**
  * Centered at `(0, 0)`, boss OD $\approx 14\text{ mm}$, height $\approx 10\text{ mm}$.
  * Internal **double-D bore** matching the 28BYJ-48 shaft ($\varnothing 5.0\text{ mm}$ + clearance, $3.0\text{ mm}$ across flats + clearance — tuned via the calibration coupon).
  * Optional small radial M3 set-screw hole (or a snug press-fit) to eliminate any vertical lift.
* **Lower Vial Cups:**
  * 10 cylindrical blind pockets ($\varnothing 12.6\text{ mm}$, depth $\approx 24\text{ mm}$) so the upper $\sim 9\text{ mm}$ of each vial + screw cap protrudes above the cup for easy removal and clean airflow.
  * **Slot 0 (Park Slot):** Solid floor (no vial pocket needed, or shallow dummy marker) with the **$\varnothing 5.2 \times 2.2\text{ mm}$ magnet pocket** recessed into the *bottom* face at $R_{\text{magnet}} = R_{\text{vial}}$.
* **11 Tall Radial Divider Fins (Key Airflow Feature):**
  * Placed midway between each pair of adjacent slots (at $\theta_k + \frac{\Delta\theta}{2}$).
  * Thickness: $2.0\text{ mm}$.
  * Height: Extends from the carousel deck up to **$2.5\text{ mm}$ above the exposed wick tips** (i.e., if vial cap top is at $+33.5\text{ mm}$ and wick protrudes $+8.0\text{ mm} \to +41.5\text{ mm}$, fins reach $+44.0\text{ mm}$ relative to vial bottom).
  * **No tall outer rim above the vial caps!** Leaving the outer radial edge open between the fins allows the fan's airstream to sweep straight horizontally across the active wick and directly out the lid's exhaust nozzle.

### Part 3: `lid_labyrinth` (Stationary Seal Plate & Outer Shell — PETG)
*(Splitting the stationary lid from the upper fan duct hood makes both parts 100% support-free on the K1C and lets you lift off the top cleanly to refill vials.)*

* **Outer Cylindrical Wall:** Rises from the base interface flange up past the carousel to enclose the rotating assembly with a $1.5\text{ mm}$ radial clearance around the carousel base.
* **Labyrinth Seal Rings (Top Underside):**
  * Two concentric downward-pointing rings, **$2.0\text{ mm}$ thick**, hanging from the flat lid ceiling:
    * **Inner Ring:** Sits inside the inner edge of the carousel's radial fins with a **$0.6\text{ mm}$ radial air gap**.
    * **Outer Ring:** Sits outside the outer edge of the carousel's radial fins with a **$0.6\text{ mm}$ radial air gap**.
  * **Vertical Air Gap:** The flat ceiling sits **$0.5\text{ mm}$ above the top edge of the 11 radial fins** (zero contact, zero friction).
* **Active Slot Aperture & Exhaust Cutout (at $\theta = 0^\circ$):**
  * **Top Inlet Aperture:** A single sector/slot cutout in the lid ceiling directly above the active vial pocket (angular span $\approx 26^\circ\text{–}28^\circ$, narrower than the $32.7^\circ$ slot pitch so the adjacent fins remain covered).
  * **Outer Exhaust Window:** A matching cutout through the lid's outer $2.0\text{ mm}$ labyrinth ring and outer shell at $\theta = 0^\circ$, opening directly into the exhaust nozzle ($\sim 22\text{ mm}$ wide $\times 14\text{ mm}$ tall).

### Part 4: `duct_hood` (Upstream Fan Mount & Curved Exhaust Duct — PETG)
* **Mounts Directly on Top of `lid_labyrinth`** (aligned with locating pins and secured with M3 screws, or optionally fused with `lid_labyrinth` if printed with minimal supports).
* **40 mm Noctua Fan Intake (Rear, $\theta = 180^\circ$):**
  * Flat mounting face tilted at **$40^\circ$** from horizontal.
  * Circular intake bore $\varnothing 38.0\text{ mm}$ + 4 corner mounting holes ($\varnothing 4.4\text{ mm}$ at $32.0 \times 32.0\text{ mm}$ spacing) for the Noctua silicone pins.
  * Ensure $\ge 5\text{ mm}$ clearance behind the intake face so the fan draws unobstructed room air.
* **Smooth Curved Duct:**
  * Transitions smoothly from the $40^\circ$ fan face down into the single lid aperture at $\theta = 0^\circ$.
  * Uses a **gently curving roof baffle** right above the aperture to direct air down across the full height of the exposed 3 mm wick, then sweeps straight out the **wide front exhaust nozzle** ($\sim 22\text{ mm}$ wide, slight flare or rounded lip, zero sharp $90^\circ$ dead-ends).
* **Fan Cable Channel:** Small $5 \times 3\text{ mm}$ notch exiting near the rear/side perimeter to drop the 4-pin Noctua wire down into the base.

### Part 5: `calibration_coupon` (10-Minute Fit-Check Print)
A small flat test plate ($\sim 50 \times 30 \times 10\text{ mm}$) generated from the **exact same `Params` struct** containing:
1. 3 variations of the **28BYJ-48 double-D shaft bore** (e.g., nominal $+0.10$, $+0.20$, $+0.30\text{ mm}$ clearance).
2. 1 **2 ml vial pocket** ($\varnothing 12.6\text{ mm}$).
3. 1 **5 × 2 mm magnet recess** ($\varnothing 5.2\text{ mm}$).
4. 1 **M3 clearance hole** ($\varnothing 3.4\text{ mm}$) + **M3 captive hex nut trap** ($5.8\text{ mm}$ AF).
5. 1 **M12 button hole** ($\varnothing 12.3\text{ mm}$).

---

## 5. Creality K1C Printing & DFM Guidelines

| Parameter | Creality K1C Guideline | CAD Modeling Rule |
| :--- | :--- | :--- |
| **Build Volume** | $220 \times 220 \times 250\text{ mm}$ | Total footprint ($\varnothing \sim 115\text{ mm}$, total height $\sim 115\text{ mm}$) fits easily with room to spare. |
| **Nozzle & Wall Width** | $0.4\text{ mm}$ nozzle ($0.42\text{ mm}$ line width) | Design all thin walls (fins, labyrinth rings, shell) to be $\ge 2.0\text{ mm}$ ($\approx 4$ perimeter walls) so PETG surfaces are vapor-tight and rigid. |
| **Overhangs** | Clean up to $45^\circ\text{–}50^\circ$ with K1C part cooling | Avoid $90^\circ$ unsupported overhangs. Use **$45^\circ$ chamfers** on bottom edges instead of fillets (fillets start nearly horizontal and droop). |
| **Horizontal Holes** | Slight top sag on circles | Add a small top teardrop relief (or $+0.2\text{ mm}$ clearance) for horizontal M3 and M12 holes. |
| **XY Tolerance** | K1C typically prints internal holes $0.1\text{–}0.2\text{ mm}$ undersized | Keep all clearances in `params.go` (`ShaftClearance`, `VialClearance`, `HoleClearance`) and verify with `calibration_coupon` in both Hyper PLA and PETG. |
| **PETG Slicer Settings** | Enclosed chamber (keep lid slightly cracked or chamber fan on low for PLA; closed for PETG) | 3–4 wall loops, 15–20% gyroid infill, $0.20\text{ mm}$ layer height (or $0.16\text{ mm}$ for the curved duct and shaft hub). |

---

## 6. Go (`soypat/gsdf`) Software Architecture

### Recommended Repository Layout
```text
cad/
├── go.mod                   # github.com/soypat/gsdf + github.com/hpinc/go3mf
├── params.go                # Single source of truth for all dimensions & tolerances (mm)
├── helpers.go               # Reusable SDF primitives (double-D shaft, M3 nut trap, sector cutout, CircArray wrappers)
├── base.go                  # Part 1: Electronics base
├── carousel.go              # Part 2: 11-slot rotor with radial fins & magnet recess
├── lid.go                   # Part 3: Stationary labyrinth lid & outer shell
├── duct.go                  # Part 4: Angled 40mm Noctua fan mount & curved exhaust hood
├── calibration.go           # Part 5: Quick fit-check coupon
├── export.go                # CPU octree meshing (glrender) -> .3mf (hpinc/go3mf) & optional .stl
└── cmd/
    └── generate/
        └── main.go          # CLI: flags for --part, --resolution, --format (3mf|stl), --out
```

### Key `soypat/gsdf` Implementation Notes
1. **Pure-Go CPU Meshing by Default:**
   * Use `gsdf`, `gleval`, and `glrender` for meshing so `go run ./cad/cmd/generate` builds cleanly anywhere with zero CGO/Vulkan prerequisites.
   * *(Optional)* Keep any `gpusdf` interactive WebGPU preview behind a Go build tag (`//go:build preview`) so CI and headless CLI builds stay lightweight.
2. **`.3mf` Export with Explicit Millimeter Units (`github.com/hpinc/go3mf`):**
   * `glrender` outputs a triangle slice (`[]ms3.Triangle`).
   * In `export.go`, deduplicate vertices (quantized to e.g. $10^{-4}\text{ mm}$ or exact `float32` map) into a `go3mf.Mesh`, set `model.Units = go3mf.UnitMillimeter`, name the object (e.g., `"carousmell_carousel"`), and write the `.3mf` archive.
   * Support exporting both individual part files (`out/base.3mf`, `out/carousel.3mf`, `out/lid_labyrinth.3mf`, `out/duct_hood.3mf`, `out/calibration.3mf`) and a stacked `out/assembly_preview.3mf` (multi-object 3MF) for visual interference checks in OrcaSlicer / Creality Print.
3. **SDF Resolution & Feature Crispness:**
   * Use a CLI `--res` flag:
     * **Fast draft preview:** `0.5 mm` voxel/leaf size (renders in seconds while iterating).
     * **Final print export:** `0.15–0.20 mm` leaf size (captures the $3.0\text{ mm}$ double-D flats, $0.5\text{ mm}$ air gaps, and smooth curved duct cleanly).
4. **Smooth Duct Modeling in SDFs:**
   * Model the curved duct interior as a swept/blended series of rounded-rectangle or elliptical cross-sections (or a `gsdf` loft/spline union with smooth `UnionSmooth` blending $k \approx 3\text{–}5\text{ mm}$) from the $40^\circ$ fan circle ($\varnothing 38\text{ mm}$) into the lid aperture and out the front nozzle.
   * Construct the duct wall by taking the outer offset shell (thickness $2.4\text{ mm}$) and subtracting the inner airflow cavity (`Difference(outerShell, innerCavity)`).
5. **Automated Geometry Sanity Checks (Unit Tests):**
   * Add a `cad/geometry_test.go` that evaluates the SDFs at critical points to assert:
     * **Zero interference:** Verify `carousel` and `lid_labyrinth` have strictly positive separation ($\ge 0.4\text{ mm}$) everywhere along the radial fin tips and labyrinth rings across all rotation angles.
     * **Shaft alignment:** Verify the double-D bore center aligns with the stepper mounting boss shaft axis at `(0, 0)`.
     * **Bounding box:** Verify every part fits within the Creality K1C build volume ($220 \times 220 \times 250\text{ mm}$).

---

## 7. Recommended Build & Iteration Workflow

1. **Bootstrap `cad/`** with `params.go`, `export.go` (`.3mf` writer), and `calibration.go`.
2. **Measure real hardware** with calipers (vial OD/height, 28BYJ-48 shaft flats, KY-024 PCB, ULN2003 PCB) and update `params.go`.
3. **Print `calibration.3mf`** on the Creality K1C (Hyper PLA first, then PETG) and dial in `ShaftClearance`, `VialClearance`, and `NutTrapAF`.
4. **Generate & visually inspect `assembly_preview.3mf`** in Creality Print / OrcaSlicer (use the slicer's layer-slider cross-section view to verify the $0.5\text{–}0.6\text{ mm}$ labyrinth air gaps and the curved duct path).
5. **Print full prototype in Hyper PLA** → bench-test homing, rotation, airflow, and button mounting.
6. **Reprint `carousel`, `lid_labyrinth`, and `duct_hood` in PETG** for the final oil-resistant unit.