// Package eem contains internal contribution-candidate logic only. It does not
// encode a Matter schema, establish library support, or admit production use.
package eem

import "math"

type Direction string

const (
	Import Direction = "import"
	Export Direction = "export"
)

type Kind string

const (
	Cumulative Kind = "cumulative"
	Periodic   Kind = "periodic"
)

// Identity uses opaque supplied evidence labels, never device or fabric IDs.
// Bounds are synthetic ordered ticks in a caller-defined common frame; they
// are not timestamps or a proposed target encoding. HasInterval distinguishes
// an evidenced zero bound from missing metadata.
type Identity struct {
	Owner, Profile, Fact, Scope string
	Direction                   Direction
	Kind                        Kind
	ResetEpoch                  string
	HasInterval                 bool
	Start, End                  int64
}

// Reading retains source evidence independently of candidate acceptance.
// Magnitude * WhPerUnit is an exact whole-Wh candidate convention only.
// Known, ContinuityKnown and Complete are supplied facts, not qualification.
type Reading struct {
	Identity        Identity
	Magnitude       int64
	WhPerUnit       uint64
	Known           bool
	ContinuityKnown bool
	Complete        bool
	Native          []byte
}

type Vector struct {
	Identity Identity
	Wh       int64
}

type Reason string

const (
	Valid                 Reason = "candidate_valid"
	InvalidIdentity       Reason = "identity_unknown_or_invalid"
	IdentityMismatch      Reason = "identity_mismatch"
	ValueUnknown          Reason = "value_unknown"
	ContinuityUnavailable Reason = "counter_continuity_unavailable"
	PartialPeriod         Reason = "partial_period"
	InvalidValue          Reason = "value_invalid"
	Overflow              Reason = "value_overflow"
)

type Result struct {
	Retained Reading
	Vector   *Vector
	Reason   Reason
}

// Validate performs no aggregation, differencing, state update or target
// mapping. A valid result is still only an internal vector. No outcome grants
// native qualification, semantic acceptance, target closure or admission.
// A withheld result preserves detached evidence; callers retain last-known-good
// values separately and must not mark them fresh on the strength of this call.
func Validate(expected Identity, observed Reading) Result {
	retained := observed
	if observed.Native != nil {
		retained.Native = make([]byte, len(observed.Native))
		copy(retained.Native, observed.Native)
	}
	r := Result{Retained: retained}
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
	case observed.Magnitude < 0 || observed.WhPerUnit == 0:
		r.Reason = InvalidValue
	case observed.WhPerUnit > math.MaxInt64 || uint64(observed.Magnitude) > uint64(math.MaxInt64)/observed.WhPerUnit:
		r.Reason = Overflow
	default:
		r.Reason = Valid
		r.Vector = &Vector{Identity: expected, Wh: observed.Magnitude * int64(observed.WhPerUnit)}
	}
	return r
}

func validIdentity(id Identity) bool {
	if id.Owner == "" || id.Profile == "" || id.Fact == "" || id.Scope == "" || id.ResetEpoch == "" {
		return false
	}
	if id.Direction != Import && id.Direction != Export {
		return false
	}
	switch id.Kind {
	case Cumulative:
		return !id.HasInterval && id.Start == 0 && id.End == 0
	case Periodic:
		return id.HasInterval && id.Start >= 0 && id.End > id.Start
	default:
		return false
	}
}
