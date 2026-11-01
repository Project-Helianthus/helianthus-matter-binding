package epm

import "strings"

// Electrical Power Measurement target identity and units for the adopted Matter
// 1.6.1 model. These are review constants for a read-only DTO, not SDK symbols or
// a claim that the adopted SDK serves this composition. The bounds are the ones
// the reference host already applies to milliamperes, carried here as Hypothesis.
const (
	Matter161DataModel       = "1.6.1"
	PowerCluster             = "ElectricalPowerMeasurement"
	PowerClusterID           = uint32(0x0090)
	PowerRevision            = uint16(3)
	ActiveCurrentAttributeID = uint32(0x0005)
	PowerTargetUnit          = "mW"
	CurrentTargetUnit        = "mA"
	VoltageTargetUnit        = "mV"
	TargetOutputScale161     = -3
	TargetMin161             = -(int64(1) << 62)
	TargetMax161             = int64(1) << 62
)

// Slot161 is one target measurement. Only a supplied Binding161 fills a slot; no
// slot is ever chosen from the text of a Fact label.
type Slot161 string

const (
	ActivePower161   Slot161 = "ActivePower"
	ActiveCurrent161 Slot161 = "ActiveCurrent"
	Voltage161       Slot161 = "Voltage"
)

// Target fact defects, reported in this order in Result161.Reasons.
const (
	PowerModeInvalid161      = "power-mode-absent-unknown-or-unevidenced"
	CountInvalid161          = "measurement-count-absent-or-unevidenced"
	AccuracyAbsent161        = "accuracy-absent"
	AccuracyMalformed161     = "accuracy-malformed-or-unevidenced"
	MeasuredClaim161         = "measured-precision-not-accepted"
	AccuracyCountMismatch161 = "accuracy-count-mismatch"
	BindingInvalid161        = "binding-invalid"
	BindingConflict161       = "binding-conflict"
	ActivePowerUndeclared161 = "active-power-undeclared"
	AccuracySlotMissing161   = "accuracy-missing-for-bound-slot"
)

// Slot withholding reasons this adapter adds to the Validate reasons.
const (
	FactsIncomplete161 = "target-facts-incomplete"
	RowMissing161      = "bound-row-missing"
	TargetUnit161      = "target-unit-mismatch"
	TargetScale161     = "target-scale-mismatch"
	NativeMissing161   = "native-evidence-missing"
	TargetRange161     = "target-range-exceeded"
)

const (
	unknownOutcome161   = "Unknown"
	withheldSurface161  = "withheld"
	fixtureReference161 = "synthetic:"
	exactStatus161      = "exact"
	nullStatus161       = "unavailable"
)

// PowerMode161 is a supplied target fact. Only AC and DC are accepted; no mode
// follows from a unit, a phase or a value.
type PowerMode161 struct {
	Value, Reference string
}

// Count161 is the supplied NumberOfMeasurementTypes; it is never counted from
// rows, bindings or Accuracy entries.
type Count161 struct {
	Value     int
	Reference string
}

// Accuracy161 is one supplied fixed Accuracy entry. A nil bound is absent, never
// zero. Measured must stay false: this adapter accepts no measured precision. The
// bounds are copied, not converted or checked against any value.
type Accuracy161 struct {
	MeasurementType                 string
	Measured                        bool
	MinMeasured, MaxMeasured, Fixed *int64
	Reference                       string
}

// Binding161 asserts that one complete candidate key feeds one slot. It is a
// fixture fact and accepts no semantic mapping.
type Binding161 struct {
	Slot      Slot161
	Key       Key
	Reference string
}

// Facts161 is the only source of target facts. Every Reference starts with
// "synthetic:": these facts are fixture input, never native evidence. Opaque
// Input.Accuracy bytes are not read.
type Facts161 struct {
	PowerMode        PowerMode161
	MeasurementTypes Count161
	Accuracy         []Accuracy161
	Bindings         []Binding161
}

// Metadata161 is a detached copy of the accepted mandatory facts.
type Metadata161 struct {
	PowerMode        PowerMode161
	MeasurementTypes Count161
	Accuracy         []Accuracy161
}

// Target161 grades its own claims. DefinitionEvidence covers reported source
// presence of the cluster identity, the mandatory attributes, optional
// ActiveCurrent and the mA/mV/mW units. EncodingEvidence covers range and null
// representation, which were not inspected in the adopted SDK definitions; the
// presence of a Voltage attribute is not asserted by either grade.
type Target161 struct {
	DataModel, Cluster                   string
	ClusterID                            uint32
	Revision                             uint16
	ActiveCurrentAttributeID             uint32
	PowerUnit, CurrentUnit, VoltageUnit  string
	Min, Max                             int64
	DefinitionEvidence, EncodingEvidence string
}

// Attribute161 is one encoded slot with its source row beside it. Null marks an
// evidenced unavailable row; Value is nil only then, and an exact zero is non-nil.
type Attribute161 struct {
	Slot      Slot161
	Unit      string
	Value     *int64
	Null      bool
	Source    Input
	Reference string
}

// Withheld161 explains an empty slot. Key is zero when no single binding applies.
type Withheld161 struct {
	Slot   Slot161
	Key    Key
	Reason string
}

// Surface161 names a target surface this adapter never encodes.
type Surface161 struct {
	Name, Disposition, Evidence string
}

// Result161 keeps the unmodified Validate output, so a row without a slot is never
// lost. A nil slot supplies no replacement sample: callers keep last-known-good
// values separately and must not mark them fresh on the strength of this call. A
// populated slot means only that supplied assertions were encodable; the Unknown
// outcomes and false admission are not changed by any encoding.
type Result161 struct {
	Target                                         Target161
	Validation                                     []Result
	Metadata                                       *Metadata161
	ActivePower, ActiveCurrent, Voltage            *Attribute161
	Withheld                                       []Withheld161
	Reasons                                        []string
	SemanticMapping, TargetClosure, LibrarySupport string
	ProductionAdmitted                             bool
	Surfaces                                       []Surface161
}

// EncodePower161 is pure and detached. It reads candidate output and supplied
// facts and has no aggregation, derived power, command, state update or SDK path.
// Any fact defect withholds every slot; a defect in one bound row withholds only
// that slot.
func EncodePower161(rows []Input, facts Facts161) Result161 {
	r := Result161{
		Target: Target161{
			DataModel: Matter161DataModel, Cluster: PowerCluster, ClusterID: PowerClusterID, Revision: PowerRevision,
			ActiveCurrentAttributeID: ActiveCurrentAttributeID,
			PowerUnit:                PowerTargetUnit, CurrentUnit: CurrentTargetUnit, VoltageUnit: VoltageTargetUnit,
			Min: TargetMin161, Max: TargetMax161,
			DefinitionEvidence: "Proven", EncodingEvidence: "Hypothesis",
		},
		Validation: Validate(rows), Withheld: []Withheld161{}, Reasons: factDefects161(facts),
		SemanticMapping: unknownOutcome161, TargetClosure: unknownOutcome161, LibrarySupport: unknownOutcome161,
		Surfaces: surfaces161(),
	}
	accepted := len(r.Reasons) == 0
	if accepted {
		r.Metadata = &Metadata161{facts.PowerMode, facts.MeasurementTypes, accuracy161(facts.Accuracy)}
	}
	for _, slot := range []Slot161{ActivePower161, ActiveCurrent161, Voltage161} {
		var bound []Binding161
		for _, b := range facts.Bindings {
			if b.Slot == slot {
				bound = append(bound, b)
			}
		}
		switch {
		case len(bound) == 0:
		case len(bound) > 1:
			r.Withheld = append(r.Withheld, Withheld161{Slot: slot, Reason: BindingConflict161})
		case !accepted:
			r.Withheld = append(r.Withheld, Withheld161{slot, bound[0].Key, FactsIncomplete161})
		default:
			a, reason := attribute161(bound[0], r.Validation)
			if reason != "" {
				r.Withheld = append(r.Withheld, Withheld161{slot, bound[0].Key, reason})
				continue
			}
			switch slot {
			case ActivePower161:
				r.ActivePower = a
			case ActiveCurrent161:
				r.ActiveCurrent = a
			case Voltage161:
				r.Voltage = a
			}
		}
	}
	return r
}

// attribute161 is called only for accepted facts. Guards run in a fixed order, so
// the first failure gives a deterministic reason and no value. Duplicate rows all
// carry the Validate withholding, so the first row of a key speaks for them.
func attribute161(b Binding161, rows []Result) (*Attribute161, string) {
	unit, target, _ := slotUnits161(b.Slot)
	for _, row := range rows {
		if row.Native.Key != b.Key {
			continue
		}
		switch {
		case row.Status != exactStatus161 && row.Status != nullStatus161:
			return nil, row.Reason
		case row.Native.Unit != unit: // Validate already holds OutputUnit equal to Unit.
			return nil, TargetUnit161
		case row.Native.OutputScale != TargetOutputScale161:
			return nil, TargetScale161
		case len(row.Native.Evidence) == 0:
			return nil, NativeMissing161
		}
		a := &Attribute161{Slot: b.Slot, Unit: target, Source: row.Native, Reference: b.Reference}
		a.Source.Evidence, a.Source.Accuracy = clone(row.Native.Evidence), clone(row.Native.Accuracy)
		if row.Status == nullStatus161 {
			a.Null = true
			return a, ""
		}
		if *row.Value < TargetMin161 || *row.Value > TargetMax161 {
			return nil, TargetRange161
		}
		value := *row.Value
		a.Value = &value
		return a, ""
	}
	return nil, RowMissing161
}

func slotUnits161(slot Slot161) (string, string, bool) {
	switch slot {
	case ActivePower161:
		return "W", PowerTargetUnit, true
	case ActiveCurrent161:
		return "A", CurrentTargetUnit, true
	case Voltage161:
		return "V", VoltageTargetUnit, true
	}
	return "", "", false
}

// factDefects161 reports every defect once in the fixed order above and defaults
// nothing: an absent fact stays a defect whatever the rows or other facts say.
func factDefects161(f Facts161) []string {
	reasons := []string{}
	if (f.PowerMode.Value != "AC" && f.PowerMode.Value != "DC") || !fixtureFact161(f.PowerMode.Reference) {
		reasons = append(reasons, PowerModeInvalid161)
	}
	countKnown := f.MeasurementTypes.Value >= 1 && fixtureFact161(f.MeasurementTypes.Reference)
	if !countKnown {
		reasons = append(reasons, CountInvalid161)
	}
	covered := make(map[string]bool, len(f.Accuracy))
	malformed, measured := false, false
	for _, a := range f.Accuracy {
		if a.MeasurementType == "" || covered[a.MeasurementType] || !fixtureFact161(a.Reference) ||
			a.MinMeasured == nil || a.MaxMeasured == nil || a.Fixed == nil ||
			*a.MinMeasured > *a.MaxMeasured || *a.Fixed < 0 {
			malformed = true
		}
		measured = measured || a.Measured
		covered[a.MeasurementType] = true
	}
	if len(f.Accuracy) == 0 {
		reasons = append(reasons, AccuracyAbsent161)
	}
	if malformed {
		reasons = append(reasons, AccuracyMalformed161)
	}
	if measured {
		reasons = append(reasons, MeasuredClaim161)
	}
	if countKnown && len(f.Accuracy) != 0 && len(f.Accuracy) != f.MeasurementTypes.Value {
		reasons = append(reasons, AccuracyCountMismatch161)
	}
	slots := make(map[Slot161]bool, len(f.Bindings))
	keys := make(map[Key]bool, len(f.Bindings))
	invalid, conflict, uncovered := false, false, false
	for _, b := range f.Bindings {
		if _, _, known := slotUnits161(b.Slot); !known || !fixtureFact161(b.Reference) {
			invalid = true
		}
		conflict = conflict || slots[b.Slot] || keys[b.Key]
		uncovered = uncovered || !covered[string(b.Slot)]
		slots[b.Slot], keys[b.Key] = true, true
	}
	if invalid {
		reasons = append(reasons, BindingInvalid161)
	}
	if conflict {
		reasons = append(reasons, BindingConflict161)
	}
	if !slots[ActivePower161] {
		reasons = append(reasons, ActivePowerUndeclared161)
	}
	if uncovered {
		reasons = append(reasons, AccuracySlotMissing161)
	}
	return reasons
}

func fixtureFact161(reference string) bool {
	return strings.HasPrefix(reference, fixtureReference161) && strings.TrimSpace(reference[len(fixtureReference161):]) != ""
}

func accuracy161(entries []Accuracy161) []Accuracy161 {
	out := make([]Accuracy161, len(entries))
	for i, a := range entries {
		a.MinMeasured, a.MaxMeasured, a.Fixed = bound161(a.MinMeasured), bound161(a.MaxMeasured), bound161(a.Fixed)
		out[i] = a
	}
	return out
}

func bound161(v *int64) *int64 {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}

// surfaces161 lists what stays out of the DTO. Their support and every mandatory
// closure question stay Unknown until the real SDK definitions are inspected.
func surfaces161() []Surface161 {
	names := []string{"apparent-power", "events", "feature-map", "frequency", "harmonics", "native-control", "polyphase", "power-factor", "power-quality", "reactive-power"}
	out := make([]Surface161, 0, len(names))
	for _, name := range names {
		out = append(out, Surface161{name, withheldSurface161, unknownOutcome161})
	}
	return out
}
