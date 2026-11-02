//go:build gofabric

package sdkfabric

import (
	"errors"
	"fmt"

	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/eem"
	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/epm"
	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/hvac"
)

// This file imports no SDK package: it links already encoded candidate DTOs to
// the qualified reference only. It adds no encoding, serves no cluster and
// admits nothing to production.

var (
	ErrReferenceUnqualified = errors.New("sdkfabric: the mandatory reference is not qualified")
	ErrStageOrder           = errors.New("sdkfabric: candidate is not next in the HVAC, EEM, EPM order")
	ErrStageActive          = errors.New("sdkfabric: another candidate is still staged")
	ErrCandidateTarget      = errors.New("sdkfabric: candidate DTO differs from its encoded target")
	ErrCandidateEmpty       = errors.New("sdkfabric: candidate DTO encodes no value")
	ErrCandidateAdmitted    = errors.New("sdkfabric: candidate DTO claims production admission")
)

// Family names one candidate family in its fixed staging order.
type Family string

const (
	FamilyHVAC Family = "hvac"
	FamilyEEM  Family = "eem"
	FamilyEPM  Family = "epm"
)

// StageOrder is the only order in which candidates may follow the reference.
var StageOrder = []Family{FamilyHVAC, FamilyEEM, FamilyEPM}

// ReferenceQualification is the evidence that the mandatory one-sensor
// reference ran on the real node: an observed CommissioningComplete whose
// fabric the node held with no window open, and at least one controller
// application receipt. Only the gofabric-tagged qualification builds a nonzero
// value; it qualifies the fixture run, not a target, conformance or a device.
type ReferenceQualification struct {
	generation uint64
	sensor     string
	fabric     uint8
	receipts   int
}

func newReferenceQualification(generation uint64, sensor string, fabric uint8, receipts int) (ReferenceQualification, error) {
	if generation == 0 || sensor == "" || fabric == 0 || receipts <= 0 {
		return ReferenceQualification{}, ErrReferenceUnqualified
	}
	return ReferenceQualification{generation, sensor, fabric, receipts}, nil
}

func (q ReferenceQualification) valid() bool {
	return q.generation != 0 && q.sensor != "" && q.fabric != 0 && q.receipts > 0
}

// Staged is one candidate DTO linked after the reference. It carries the DTO
// unchanged; ProductionAdmitted is always false.
type Staged struct {
	Family             Family
	DataModel          string
	ClusterID          uint32
	Revision           uint16
	Encoded            int
	Reference          ReferenceQualification
	HVAC               *hvac.Result161
	EEM                *eem.Result161
	EPM                *epm.Result161
	ProductionAdmitted bool
}

// Stager selects one candidate at a time, in StageOrder, after the reference.
// It is serial and instance-local; a new Stager starts again at HVAC.
type Stager struct {
	reference ReferenceQualification
	next      int
	active    *Family
}

// NewStager refuses every reference that the tagged qualification did not build.
func NewStager(reference ReferenceQualification) (*Stager, error) {
	if !reference.valid() {
		return nil, ErrReferenceUnqualified
	}
	return &Stager{reference: reference}, nil
}

func (s *Stager) admit(f Family) error {
	if s == nil || !s.reference.valid() {
		return ErrReferenceUnqualified
	}
	if s.active != nil {
		return fmt.Errorf("%w: %s", ErrStageActive, *s.active)
	}
	if s.next >= len(StageOrder) || StageOrder[s.next] != f {
		return fmt.Errorf("%w: %s", ErrStageOrder, f)
	}
	return nil
}

func (s *Stager) stage(st Staged) Staged {
	f := st.Family
	s.active = &f
	st.Reference, st.ProductionAdmitted = s.reference, false
	return st
}

// StageHVAC links an encoded Thermostat DTO. At least one thermostat must
// carry an encoded temperature or SystemMode; the DTO is not re-encoded.
func (s *Stager) StageHVAC(r hvac.Result161) (Staged, error) {
	if err := s.admit(FamilyHVAC); err != nil {
		return Staged{}, err
	}
	t := r.Target
	if t.DataModel != hvac.Matter161DataModel || t.ClusterID != hvac.ThermostatClusterID || t.Revision != hvac.ThermostatRevision {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateTarget, FamilyHVAC)
	}
	if r.Validation.ProductionAdmitted {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateAdmitted, FamilyHVAC)
	}
	encoded := 0
	for _, th := range r.Thermostats {
		ts := th.Temperatures
		for _, v := range []*hvac.Temperature161{ts.MeasuredRoom, ts.HeatingConfigured, ts.HeatingManual, ts.HeatingEffective,
			ts.CoolingConfigured, ts.CoolingManual, ts.CoolingEffective} {
			if v != nil {
				encoded++
			}
		}
		if th.SystemMode != nil {
			encoded++
		}
	}
	if encoded == 0 {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateEmpty, FamilyHVAC)
	}
	return s.stage(Staged{Family: FamilyHVAC, DataModel: t.DataModel, ClusterID: t.ClusterID, Revision: t.Revision, Encoded: encoded, HVAC: &r}), nil
}

// StageEEM links an encoded Electrical Energy Measurement DTO.
func (s *Stager) StageEEM(r eem.Result161) (Staged, error) {
	if err := s.admit(FamilyEEM); err != nil {
		return Staged{}, err
	}
	t := r.Target
	if t.DataModel != eem.Matter161DataModel || t.ClusterID != eem.EnergyClusterID || t.Revision != eem.EnergyRevision || t.Unit != eem.EnergyTargetUnit {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateTarget, FamilyEEM)
	}
	if r.ProductionAdmitted {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateAdmitted, FamilyEEM)
	}
	if r.Energy == nil {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateEmpty, FamilyEEM)
	}
	return s.stage(Staged{Family: FamilyEEM, DataModel: t.DataModel, ClusterID: t.ClusterID, Revision: t.Revision, Encoded: 1, EEM: &r}), nil
}

// StageEPM links an encoded Electrical Power Measurement DTO. Its mandatory
// metadata must have been accepted and at least one slot encoded.
func (s *Stager) StageEPM(r epm.Result161) (Staged, error) {
	if err := s.admit(FamilyEPM); err != nil {
		return Staged{}, err
	}
	t := r.Target
	if t.DataModel != epm.Matter161DataModel || t.ClusterID != epm.PowerClusterID || t.Revision != epm.PowerRevision {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateTarget, FamilyEPM)
	}
	if r.ProductionAdmitted {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateAdmitted, FamilyEPM)
	}
	encoded := 0
	for _, a := range []*epm.Attribute161{r.ActivePower, r.ActiveCurrent, r.Voltage} {
		if a != nil {
			encoded++
		}
	}
	if r.Metadata == nil || encoded == 0 {
		return Staged{}, fmt.Errorf("%w: %s", ErrCandidateEmpty, FamilyEPM)
	}
	return s.stage(Staged{Family: FamilyEPM, DataModel: t.DataModel, ClusterID: t.ClusterID, Revision: t.Revision, Encoded: encoded, EPM: &r}), nil
}

// Release ends the staged candidate f, so the next family may be selected.
func (s *Stager) Release(f Family) error {
	if s == nil || s.active == nil || *s.active != f {
		return fmt.Errorf("%w: %s is not staged", ErrStageOrder, f)
	}
	s.active = nil
	s.next++
	return nil
}

// Next names the family that may be staged next, or "" when all were linked.
func (s *Stager) Next() Family {
	if s == nil || s.active != nil || s.next >= len(StageOrder) {
		return ""
	}
	return StageOrder[s.next]
}
