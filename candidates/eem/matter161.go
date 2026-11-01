package eem

import "math"

// Electrical Energy Measurement target identity and unit for the adopted Matter
// 1.6.1 model. These are review constants for a read-only DTO, not SDK symbols or
// a claim that the adopted SDK serves this composition. The whole-Wh Validate
// convention stays a separate internal one and never feeds this adapter.
const (
	Matter161DataModel = "1.6.1"
	EnergyCluster      = "ElectricalEnergyMeasurement"
	EnergyClusterID    = uint32(0x0091)
	EnergyRevision     = uint16(2)
	EnergyTargetUnit   = "mWh"
	EnergyMinMilliWh   = int64(0)
	EnergyMaxMilliWh   = int64(math.MaxInt64)
)

// Withholding reasons this adapter adds to the shared Reason set.
const (
	UnitUnknown161   Reason = "unit_unknown_or_invalid"
	PrecisionLoss161 Reason = "value_precision_loss"
	TimeInvalid161   Reason = "time_metadata_invalid"
	NativeMissing161 Reason = "native_evidence_missing"
)

const (
	unknownOutcome161  = "Unknown"
	withheldSurface161 = "withheld"
)

// EnergyUnit161 is matched exactly, so mWh and MWh can never be confused.
type EnergyUnit161 string

const (
	MilliWh161 EnergyUnit161 = "mWh"
	Wh161      EnergyUnit161 = "Wh"
	KiloWh161  EnergyUnit161 = "kWh"
	MegaWh161  EnergyUnit161 = "MWh"
)

// Decimal161 is the exact value Coefficient * 10^Exponent Unit. A negative
// coefficient has no encoding: a sign never chooses a direction.
type Decimal161 struct {
	Coefficient int64
	Exponent    int32
	Unit        EnergyUnit161
}

// Time161 is optional time metadata in one supplied frame. A nil bound is
// Unknown, never zero; nothing is converted or derived from Identity ticks.
type Time161 struct {
	Frame      string
	Start, End *int64
}

// ExactReading161 mirrors Reading with a decimal value in place of the whole-Wh
// product. Known, ContinuityKnown and Complete are supplied facts, not
// qualification.
type ExactReading161 struct {
	Identity        Identity
	Value           Decimal161
	Known           bool
	ContinuityKnown bool
	Complete        bool
	Time            *Time161
	Native          []byte
}

// Slot161 keeps each direction and kind apart. Assigning a slot to a target
// attribute needs an accepted equivalence that is not supplied here.
type Slot161 string

const (
	ImportedCumulative161 Slot161 = "imported-cumulative"
	ExportedCumulative161 Slot161 = "exported-cumulative"
	ImportedPeriodic161   Slot161 = "imported-periodic"
	ExportedPeriodic161   Slot161 = "exported-periodic"
)

// Target161 grades its own claims. DefinitionEvidence covers reported source
// presence of the cluster identity, its directions and kinds, and the mWh unit.
// EncodingEvidence covers range, nullability and time representation, which were
// not inspected in the adopted SDK definitions.
type Target161 struct {
	DataModel, Cluster                   string
	ClusterID                            uint32
	Revision                             uint16
	Unit                                 string
	MinMilliWh, MaxMilliWh               int64
	DefinitionEvidence, EncodingEvidence string
}

// Energy161 is one encoded value with its source decimal and native bytes beside
// it. Identity carries the reset epoch and the interval ticks unchanged.
type Energy161 struct {
	Identity Identity
	Slot     Slot161
	MilliWh  int64
	Time     *Time161
	Source   Decimal161
	Native   []byte
}

// Surface161 names a target surface this adapter never encodes.
type Surface161 struct {
	Name, Disposition, Evidence string
}

// Result161 always retains the detached reading. A withheld result supplies no
// replacement sample: callers keep last-known-good values separately and must not
// mark them fresh on the strength of this call. A populated Energy means only that
// supplied assertions were encodable; the Unknown outcomes and false admission
// are not changed by any encoding.
type Result161 struct {
	Target                                         Target161
	Retained                                       ExactReading161
	Energy                                         *Energy161
	Reason                                         Reason
	SemanticMapping, TargetClosure, LibrarySupport string
	ProductionAdmitted                             bool
	Surfaces                                       []Surface161
}

// EncodeEnergy161 is pure and detached. It reads one reading and has no
// differencing, aggregation, reset, state update or SDK path. Guards run in a
// fixed order, so the first failure gives a deterministic reason and no value.
func EncodeEnergy161(expected Identity, observed ExactReading161) Result161 {
	r := Result161{
		Target: Target161{
			DataModel: Matter161DataModel, Cluster: EnergyCluster, ClusterID: EnergyClusterID, Revision: EnergyRevision,
			Unit: EnergyTargetUnit, MinMilliWh: EnergyMinMilliWh, MaxMilliWh: EnergyMaxMilliWh,
			DefinitionEvidence: "Proven", EncodingEvidence: "Hypothesis",
		},
		Retained: observed, SemanticMapping: unknownOutcome161, TargetClosure: unknownOutcome161, LibrarySupport: unknownOutcome161,
		Surfaces: surfaces161(),
	}
	r.Retained.Native = bytes161(observed.Native)
	r.Retained.Time = time161(observed.Time)
	switch {
	case !validIdentity(expected) || !validIdentity(observed.Identity):
		r.Reason = InvalidIdentity
	case observed.Identity != expected:
		r.Reason = IdentityMismatch
	case !observed.Known:
		r.Reason = ValueUnknown
	case !observed.ContinuityKnown:
		r.Reason = ContinuityUnavailable
	case !observed.Complete:
		r.Reason = PartialPeriod
	case len(observed.Native) == 0:
		r.Reason = NativeMissing161
	case !validTime161(observed.Identity.Kind, observed.Time):
		r.Reason = TimeInvalid161
	default:
		var milliWh int64
		if milliWh, r.Reason = milliWh161(observed.Value); r.Reason == Valid {
			r.Energy = &Energy161{
				Identity: expected, Slot: slot161(expected), MilliWh: milliWh, Time: time161(observed.Time),
				Source: observed.Value, Native: bytes161(observed.Native),
			}
		}
	}
	return r
}

// milliWh161 never rounds: the result must equal the supplied decimal exactly, so
// a value finer than 1 mWh has no encoding. The loops end within 19 steps for any
// nonzero int64 whatever the exponent, and zero is exact at every scale.
func milliWh161(v Decimal161) (int64, Reason) {
	scale, ok := unitScale161(v.Unit)
	if !ok {
		return 0, UnitUnknown161
	}
	c := v.Coefficient
	if c < 0 {
		return 0, InvalidValue
	}
	if c == 0 {
		return 0, Valid
	}
	for e := int64(v.Exponent) + scale; e != 0; {
		if e < 0 {
			if c%10 != 0 {
				return 0, PrecisionLoss161
			}
			c /= 10
			e++
			continue
		}
		if c > EnergyMaxMilliWh/10 {
			return 0, Overflow
		}
		c *= 10
		e--
	}
	return c, Valid
}

func unitScale161(unit EnergyUnit161) (int64, bool) {
	switch unit {
	case MilliWh161:
		return 0, true
	case Wh161:
		return 3, true
	case KiloWh161:
		return 6, true
	case MegaWh161:
		return 9, true
	}
	return 0, false
}

// validTime161 accepts absent metadata. A present record names its frame, holds at
// least one nonnegative bound and orders two bounds; a periodic value also needs
// both, because a half-known period is not a complete one.
func validTime161(kind Kind, t *Time161) bool {
	if t == nil {
		return true
	}
	if t.Frame == "" || (t.Start == nil && t.End == nil) {
		return false
	}
	if (t.Start != nil && *t.Start < 0) || (t.End != nil && *t.End < 0) {
		return false
	}
	if t.Start != nil && t.End != nil {
		return *t.End > *t.Start
	}
	return kind != Periodic
}

// slot161 is called only for a validated identity.
func slot161(id Identity) Slot161 {
	switch {
	case id.Direction == Import && id.Kind == Cumulative:
		return ImportedCumulative161
	case id.Direction == Export && id.Kind == Cumulative:
		return ExportedCumulative161
	case id.Direction == Import && id.Kind == Periodic:
		return ImportedPeriodic161
	case id.Direction == Export && id.Kind == Periodic:
		return ExportedPeriodic161
	}
	panic("unsupported internal energy identity")
}

func bytes161(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func time161(t *Time161) *Time161 {
	if t == nil {
		return nil
	}
	return &Time161{Frame: t.Frame, Start: bound161(t.Start), End: bound161(t.End)}
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
	names := []string{"aggregation", "differencing", "events", "feature-map", "measurement-accuracy", "reset-operation", "reset-structure"}
	out := make([]Surface161, 0, len(names))
	for _, name := range names {
		out = append(out, Surface161{name, withheldSurface161, unknownOutcome161})
	}
	return out
}
