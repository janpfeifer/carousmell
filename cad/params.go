// Package cad defines the parametric 3D SDF models, exporters, and visualizers
// for the Carousmell overnight olfactory carousel.
//
// Coordinate System (all units in millimeters, Z-up):
//   - Origin (0, 0, Z): 28BYJ-48 stepper motor output shaft axis (rotational axis of carousel).
//   - Z = 0: Bottom exterior face of the base resting on the nightstand.
//   - +X (theta = 0 deg): Active slot aperture & front exhaust nozzle facing the user.
//   - -X (theta = 180 deg): Upstream 40mm Noctua fan intake & fan wire channel.
//   - +Y (theta = +90 deg): Raspberry Pi Zero 2 WH side (Micro-USB PWR IN on outer wall).
//   - -Y (theta = -90 deg): Stepper motor body offset (-8mm) and ULN2003 driver board side.
package cad

import "github.com/chewxy/math32"

// Params holds every parametric dimension and tolerance (in millimeters) for
// all Carousmell printable parts. Update values marked "(verify)" after caliper
// measurements and a calibration_coupon test print on the Creality K1C.
type Params struct {
	// --- Global Enclosure ---
	OuterRadius    float32 // Outer housing radius (60.0 mm -> Ø120 mm)
	ShellWall      float32 // Outer cylindrical shell wall thickness (3.6 mm)
	LipHeight      float32 // Interlocking stepped alignment lip height (3.0 mm)
	LipClearance   float32 // Radial slip-fit clearance on stepped lips (0.25 mm)
	BottomChamfer  float32 // 45-degree bottom edge chamfer to prevent elephant's foot (1.0 mm)
	PerimeterBolts int     // Number of perimeter M3 fasteners (4)
	BoltCircleR    float32 // Radius of perimeter M3 bolt circle (53.0 mm)
	BoltBossR      float32 // Radius of internal M3 boss columns (4.4 mm)

	// --- M3 Fastener Tolerances ---
	M3ClearanceD float32 // M3 through-hole diameter (3.4 mm)
	M3NutAF      float32 // M3 captive hex nut across-flats (5.8 mm)
	M3NutDepth   float32 // M3 captive hex nut thickness trap (2.6 mm)

	// --- Base (Part 1: Electronics Tub) ---
	BaseHeight     float32 // Top of base / motor_deck top surface (42.0 mm)
	BaseFloor      float32 // Base bottom floor thickness (3.0 mm)
	MotorDeckThick float32 // Divider plate thickness (3.0 mm)
	MotorDeckGap   float32 // Radial slip clearance for motor_deck inside base (0.30 mm)

	// --- Raspberry Pi Zero 2 WH ---
	PiLength     float32 // PCB length along X (65.0 mm)
	PiWidth      float32 // PCB width along Y (30.0 mm)
	PiCenterY    float32 // PCB center Y offset (+29.5 mm)
	PiHoleSpanX  float32 // M2.5 mounting hole spacing X (58.0 mm)
	PiHoleSpanY  float32 // M2.5 mounting hole spacing Y (23.0 mm)
	PiStandoffH  float32 // Standoff height above base floor (4.0 mm)
	PiUSBCenterX float32 // Micro-USB PWR IN center X (+21.5 mm from PCB center)
	PiUSBWidth   float32 // Wall cutout width for molded USB boot (18.0 mm)
	PiUSBHeight  float32 // Wall cutout height for molded USB boot (12.0 mm)

	// --- ULN2003 Driver Board (verify) ---
	ULNLength    float32 // PCB length along X (35.0 mm)
	ULNWidth     float32 // PCB width along Y (32.0 mm)
	ULNCenterX   float32 // PCB center X (-8.0 mm)
	ULNCenterY   float32 // PCB center Y (-33.0 mm)
	ULNHoleSpanX float32 // Mounting hole spacing X (29.0 mm)
	ULNHoleSpanY float32 // Mounting hole spacing Y (26.0 mm)
	ULNStandoffH float32 // Standoff height above base floor (3.0 mm)

	// --- Manual "Whiff Now" Push Button (M12 ATX) ---
	ButtonAngleDeg float32 // Angular position on base wall (-55.0 deg)
	ButtonHoleD    float32 // M12 barrel clearance hole diameter (12.3 mm)
	ButtonWall     float32 // Flat boss wall thickness for M12 hex nut (3.0 mm)
	ButtonCenterZ  float32 // Vertical center of button on base wall (21.0 mm)

	// --- 28BYJ-48 Stepper Motor ---
	StepperBodyD        float32 // Metal body OD (28.0 mm)
	StepperBodyH        float32 // Metal body height below mounting flange (19.0 mm)
	StepperShaftOffsetY float32 // Body center Y offset from shaft at (0,0) (-8.0 mm)
	StepperEarSpacing   float32 // Mounting ear center-to-center distance (35.0 mm)
	StepperBossHoleD    float32 // Clearance bore in motor_deck for Ø9mm boss (10.6 mm)
	StepperShaftD       float32 // Nominal shaft diameter (5.0 mm)
	StepperShaftFlats   float32 // Nominal double-D across-flats (3.0 mm)
	StepperShaftHeight  float32 // Total shaft height above mounting flange (10.0 mm)
	ShaftClearance      float32 // Diametral clearance on double-D bore (0.20 mm, tune with coupon)

	// --- KY-024 Hall-Effect Sensor & Homing Magnet ---
	MagnetD        float32 // Nominal magnet disc diameter (5.0 mm)
	MagnetH        float32 // Nominal magnet disc height (2.0 mm)
	MagnetClearD   float32 // Pocket diameter clearance (+0.20 mm -> 5.2 mm)
	MagnetClearH   float32 // Pocket depth clearance (+0.20 mm -> 2.2 mm)
	HallWindowSize float32 // Thin sensing window size in motor_deck at Slot 0 (9.0 mm)
	HallWindowWall float32 // Remaining floor thickness over Hall IC (0.8 mm)

	// --- Carousel (Part 3: 11-Slot Rotor) ---
	NumSlots           int     // Total slots: 10 oil vials + 1 empty park slot (11)
	VialPitchR         float32 // Pitch circle radius of vial centers (32.0 mm)
	VialOD             float32 // Nominal 2 ml amber glass vial OD (12.0 mm, verify)
	VialClearance      float32 // Diametral clearance for drop-in vial pockets (0.6 mm -> Ø12.6 mm)
	VialTotalH         float32 // Total height of vial with open-top cap (33.5 mm, verify)
	WickProtrusion     float32 // Exposed 3 mm fiber wick above vial cap (8.0 mm)
	CarouselFloor      float32 // Carousel base disc floor thickness under vials (3.5 mm)
	CarouselCupH       float32 // Lower vial cup height above vial bottom (24.0 mm)
	CarouselInnerR     float32 // Inner radius of vial ring & radial fins (23.5 mm)
	CarouselOuterR     float32 // Outer radius of vial ring & radial fins (40.5 mm)
	CarouselBottomGap  float32 // Vertical air gap between motor_deck top and carousel bottom (1.5 mm)
	CarouselHubOD      float32 // Central direct-drive hub OD (14.0 mm)
	CarouselHubH       float32 // Central hub height above carousel bottom (10.0 mm)
	FinThickness       float32 // Thickness of the 11 radial divider fins (2.2 mm)
	FinClearanceOverWk float32 // Radial fin extension above exposed wick tips (2.5 mm)

	// --- Labyrinth Seal Lid (Part 4) ---
	LabyrinthRadialGap  float32 // Non-contact radial air gap to carousel fins (0.60 mm)
	LabyrinthVertGap    float32 // Non-contact vertical air gap above fin tips (0.60 mm)
	LabyrinthRingThick  float32 // Thickness of inner & outer labyrinth seal rings (2.2 mm)
	LabyrinthOverlap    float32 // Downward overlap past carousel cup top (2.0 mm)
	LidCeilingThick     float32 // Top ceiling plate thickness (3.0 mm)
	ApertureSpanDeg     float32 // Angular span of top inlet aperture over active slot (25.0 deg)
	ExhaustInnerW       float32 // Exhaust opening width at outer labyrinth ring (18.0 mm)
	ExhaustExitW        float32 // Flared exhaust nozzle exit width (22.0 mm)
	ExhaustFloorZOffset float32 // Exhaust floor height above carousel cup top (3.0 mm)
	ExhaustProtrusion   float32 // Nozzle extension past OuterRadius (8.0 mm)
	ExhaustWall         float32 // Exhaust tunnel & nozzle wall thickness (2.4 mm)

	// --- Upstream Fan Duct Hood (Part 5) ---
	FanSize       float32 // Noctua NF-A4x10 square side (40.0 mm)
	FanThick      float32 // Noctua NF-A4x10 thickness (10.0 mm)
	FanBoreD      float32 // Circular intake bore diameter (38.0 mm)
	FanHoleSpan   float32 // Mounting hole center spacing (32.0 mm)
	FanHoleD      float32 // Silicone anti-vibration pin hole diameter (4.4 mm)
	FanTiltDeg    float32 // Duct axis & fan face tilt from horizontal (45.0 deg, support-free!)
	FanPlateThick float32 // Fan mounting plate thickness (3.5 mm)
	FanCenterX    float32 // Fan intake face center X (-16.0 mm)
	FanCenterZOff float32 // Fan intake face center Z above lid top (26.0 mm)
	DuctWall      float32 // Curved turning duct wall thickness (2.6 mm)
	HoodHeight    float32 // Overall duct_hood height above lid top (50.4 mm)

	// --- Wire Channel ---
	WireChannelW float32 // Fan cable routing channel width (6.0 mm)
	WireChannelD float32 // Fan cable routing channel depth (4.0 mm)
}

// DefaultParams returns the verified, Creality K1C support-free default parameters.
func DefaultParams() Params {
	return Params{
		OuterRadius:    60.0,
		ShellWall:      3.6,
		LipHeight:      3.0,
		LipClearance:   0.25,
		BottomChamfer:  1.0,
		PerimeterBolts: 4,
		BoltCircleR:    53.0,
		BoltBossR:      4.4,

		M3ClearanceD: 3.4,
		M3NutAF:      5.8,
		M3NutDepth:   2.6,

		BaseHeight:     42.0,
		BaseFloor:      3.0,
		MotorDeckThick: 3.0,
		MotorDeckGap:   0.30,

		PiLength:     65.0,
		PiWidth:      30.0,
		PiCenterY:    29.5,
		PiHoleSpanX:  58.0,
		PiHoleSpanY:  23.0,
		PiStandoffH:  4.0,
		PiUSBCenterX: 21.5,
		PiUSBWidth:   18.0,
		PiUSBHeight:  12.0,

		ULNLength:    35.0,
		ULNWidth:     32.0,
		ULNCenterX:   -8.0,
		ULNCenterY:   -33.0,
		ULNHoleSpanX: 29.0,
		ULNHoleSpanY: 26.0,
		ULNStandoffH: 3.0,

		ButtonAngleDeg: -55.0,
		ButtonHoleD:    12.3,
		ButtonWall:     3.0,
		ButtonCenterZ:  21.0,

		StepperBodyD:        28.0,
		StepperBodyH:        19.0,
		StepperShaftOffsetY: -8.0,
		StepperEarSpacing:   35.0,
		StepperBossHoleD:    10.6,
		StepperShaftD:       5.0,
		StepperShaftFlats:   3.0,
		StepperShaftHeight:  10.0,
		ShaftClearance:      0.20,

		MagnetD:        5.0,
		MagnetH:        2.0,
		MagnetClearD:   0.20,
		MagnetClearH:   0.20,
		HallWindowSize: 9.0,
		HallWindowWall: 0.8,

		NumSlots:           11,
		VialPitchR:         32.0,
		VialOD:             12.0,
		VialClearance:      0.60,
		VialTotalH:         33.5,
		WickProtrusion:     8.0,
		CarouselFloor:      3.5,
		CarouselCupH:       24.0,
		CarouselInnerR:     23.5,
		CarouselOuterR:     40.5,
		CarouselBottomGap:  1.5,
		CarouselHubOD:      14.0,
		CarouselHubH:       10.0,
		FinThickness:       2.2,
		FinClearanceOverWk: 2.5,

		LabyrinthRadialGap:  0.60,
		LabyrinthVertGap:    0.60,
		LabyrinthRingThick:  2.2,
		LabyrinthOverlap:    2.0,
		LidCeilingThick:     3.0,
		ApertureSpanDeg:     25.0,
		ExhaustInnerW:       18.0,
		ExhaustExitW:        22.0,
		ExhaustFloorZOffset: 3.0,
		ExhaustProtrusion:   8.0,
		ExhaustWall:         2.4,

		FanSize:       40.0,
		FanThick:      10.0,
		FanBoreD:      38.0,
		FanHoleSpan:   32.0,
		FanHoleD:      4.4,
		FanTiltDeg:    45.0,
		FanPlateThick: 3.5,
		FanCenterX:    -16.0,
		FanCenterZOff: 28.5,
		DuctWall:      2.6,
		HoodHeight:    52.0,

		WireChannelW: 6.0,
		WireChannelD: 4.0,
	}
}

// --- Derived Geometric Coordinates (Single Source of Truth) ---

// InnerRadius is the interior cylindrical radius of base and lid_labyrinth.
func (p Params) InnerRadius() float32 { return p.OuterRadius - p.ShellWall }

// BoltAnglesRad returns the 4 symmetric perimeter bolt angles (radians)
// verified clear of the Pi Zero, ULN2003, button, exhaust tunnel, and fan.
func (p Params) BoltAnglesRad() []float32 {
	const deg = math32.Pi / 180
	return []float32{25 * deg, 140 * deg, -140 * deg, -25 * deg}
}

// MotorDeckBottomZ is the underside of the motor_deck divider plate (39.0 mm).
func (p Params) MotorDeckBottomZ() float32 { return p.BaseHeight - p.MotorDeckThick }

// CarouselBottomZ is the global Z of the carousel bottom face (43.5 mm).
func (p Params) CarouselBottomZ() float32 { return p.BaseHeight + p.CarouselBottomGap }

// CarouselCupTopLocalZ is the local Z of the top of the lower vial cups (27.5 mm).
func (p Params) CarouselCupTopLocalZ() float32 { return p.CarouselFloor + p.CarouselCupH }

// WickTipLocalZ is the local Z of the exposed 3 mm wick tips (45.0 mm).
func (p Params) WickTipLocalZ() float32 {
	return p.CarouselFloor + p.VialTotalH + p.WickProtrusion
}

// CarouselHeight is the total height of the carousel (to top of radial fins, 47.5 mm).
func (p Params) CarouselHeight() float32 {
	return p.WickTipLocalZ() + p.FinClearanceOverWk
}

// CarouselFinTopZ is the global Z of the top edge of the 11 radial fins (91.0 mm).
func (p Params) CarouselFinTopZ() float32 {
	return p.CarouselBottomZ() + p.CarouselHeight()
}

// ShaftTipZ is the global Z of the top tip of the 28BYJ-48 stepper shaft (49.0 mm).
func (p Params) ShaftTipZ() float32 {
	return p.MotorDeckBottomZ() + p.StepperShaftHeight
}

// ShaftBoreDepth is the blind double-D bore depth inside the carousel hub so the
// carousel rests positively on the stepper shaft tip at CarouselBottomZ (5.5 mm).
func (p Params) ShaftBoreDepth() float32 {
	return p.ShaftTipZ() - p.CarouselBottomZ()
}

// VialPocketD is the drop-in vial pocket diameter (12.6 mm).
func (p Params) VialPocketD() float32 { return p.VialOD + p.VialClearance }

// SlotAngleRad is the angular spacing between adjacent carousel slots (2π/11).
func (p Params) SlotAngleRad() float32 { return 2 * math32.Pi / float32(p.NumSlots) }

// LidBottomZ is the global Z where lid_labyrinth seats on the base (42.0 mm).
func (p Params) LidBottomZ() float32 { return p.BaseHeight }

// LidCeilingInZ is the global Z of the underside of the stationary lid ceiling (91.6 mm).
func (p Params) LidCeilingInZ() float32 { return p.CarouselFinTopZ() + p.LabyrinthVertGap }

// LidTopZ is the global Z of the top face of lid_labyrinth (94.6 mm).
func (p Params) LidTopZ() float32 { return p.LidCeilingInZ() + p.LidCeilingThick }

// LabyrinthRingBottomZ is the global Z of the bottom edge of both labyrinth seal rings (69.0 mm).
func (p Params) LabyrinthRingBottomZ() float32 {
	return p.CarouselBottomZ() + p.CarouselCupTopLocalZ() - p.LabyrinthOverlap
}

// InnerRingOuterR is the outer radius of the inner labyrinth seal ring (22.9 mm).
func (p Params) InnerRingOuterR() float32 { return p.CarouselInnerR - p.LabyrinthRadialGap }

// InnerRingInnerR is the inner radius of the inner labyrinth seal ring (20.7 mm).
func (p Params) InnerRingInnerR() float32 { return p.InnerRingOuterR() - p.LabyrinthRingThick }

// OuterRingInnerR is the inner radius of the outer labyrinth seal ring (41.1 mm).
func (p Params) OuterRingInnerR() float32 { return p.CarouselOuterR + p.LabyrinthRadialGap }

// OuterRingOuterR is the outer radius of the outer labyrinth seal ring (43.3 mm).
func (p Params) OuterRingOuterR() float32 { return p.OuterRingInnerR() + p.LabyrinthRingThick }

// ExhaustFloorZ is the global Z of the interior floor of the exhaust tunnel (74.0 mm).
func (p Params) ExhaustFloorZ() float32 {
	return p.CarouselBottomZ() + p.CarouselCupTopLocalZ() + p.ExhaustFloorZOffset
}

// ExhaustExitX is the global X coordinate of the flared exhaust nozzle tip (68.0 mm).
func (p Params) ExhaustExitX() float32 { return p.OuterRadius + p.ExhaustProtrusion }

// LipMidR is the radial boundary between the base alignment step and lid alignment skirt.
func (p Params) LipMidR() float32 { return p.InnerRadius() + p.ShellWall/2 }

// HoodBottomZ is the global Z where duct_hood mounts onto lid_labyrinth (94.6 mm).
func (p Params) HoodBottomZ() float32 { return p.LidTopZ() }

// HoodTopZ is the global Z of the top of duct_hood (~145.0 mm).
func (p Params) HoodTopZ() float32 { return p.HoodBottomZ() + p.HoodHeight }
