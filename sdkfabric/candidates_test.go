//go:build gofabric

package sdkfabric

import (
	"errors"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/eem"
	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/epm"
	"github.com/Project-Helianthus/helianthus-matter-binding/candidates/hvac"
)

// The candidate linkage tests use only this package's stager and the merged
// encoders: no SDK call, node or network is reached.

func aa30Reference(t *testing.T) ReferenceQualification {
	t.Helper()
	q, err := newReferenceQualification(1, "helianthus.reference/sensor-a", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func aa30Celsius(v float64) *float64 { return &v }

func aa30HVAC() hvac.Result161 {
	yes := hvac.Support{State: "Yes", Reference: "synthetic-capability"}
	return hvac.EncodeThermostat161([]hvac.Vector{{
		Key:      hvac.Key{Domain: "Thermal", Owner: "synthetic-native", Profile: "family-a/profile-a", Scope: "room-a", Fact: "fact-room", Direction: hvac.Observation, Target: "slot-room"},
		Evidence: hvac.Evidence{NativeContract: "synthetic-contract", SourceReference: "synthetic-source", Confidence: "Proven", Qualification: "Unknown"},
		Raw:      []byte("room"), Kind: "temperature", Role: "measured-room", Unit: "C", Celsius: aa30Celsius(21.5),
		Heating: yes, Cooling: yes, Loss: []string{},
	}}, nil)
}

func aa30EEM() eem.Result161 {
	r := eem.ExactReading161{
		Identity: eem.Identity{Owner: "synthetic-native", Profile: "meter-family", Fact: "energy", Scope: "service-a", Direction: eem.Import, Kind: eem.Cumulative, ResetEpoch: "epoch-a"},
		Value:    eem.Decimal161{Coefficient: 15, Exponent: -1, Unit: eem.Wh161},
		Known:    true, ContinuityKnown: true, Complete: true, Native: []byte{15, 0},
	}
	return eem.EncodeEnergy161(r.Identity, r)
}

func aa30Bound(v int64) *int64 { return &v }

func aa30EPM() epm.Result161 {
	row := func(fact, unit, decimal string) epm.Input {
		return epm.Input{
			Key:          epm.Key{Owner: "synthetic-native", Profile: "meter-family", Source: "source-a", Scope: "phase", Phase: "a", Fact: fact},
			Availability: "available", Decimal: decimal, Unit: unit, OutputUnit: unit, OutputScale: -3,
			Evidence: []byte("synthetic " + fact), Accuracy: []byte("opaque " + fact),
		}
	}
	power, current := row("active-power", "W", "-250"), row("active-current", "A", "1.5")
	accuracy := func(kind string) epm.Accuracy161 {
		return epm.Accuracy161{MeasurementType: kind, MinMeasured: aa30Bound(-1000), MaxMeasured: aa30Bound(1000), Fixed: aa30Bound(50), Reference: "synthetic:accuracy-" + kind}
	}
	return epm.EncodePower161([]epm.Input{power, current}, epm.Facts161{
		PowerMode:        epm.PowerMode161{Value: "AC", Reference: "synthetic:power-mode"},
		MeasurementTypes: epm.Count161{Value: 2, Reference: "synthetic:count"},
		Accuracy:         []epm.Accuracy161{accuracy("ActivePower"), accuracy("ActiveCurrent")},
		Bindings: []epm.Binding161{
			{Slot: epm.ActivePower161, Key: power.Key, Reference: "synthetic:power-binding"},
			{Slot: epm.ActiveCurrent161, Key: current.Key, Reference: "synthetic:current-binding"},
		},
	})
}

func aa30Stager(t *testing.T) *Stager {
	t.Helper()
	s, err := NewStager(aa30Reference(t))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestAA30CandidateLinkageOrder links all three encoded DTOs, one at a time,
// in HVAC, EEM, EPM order. Expected identities are literal, independent of
// the encoder constants. Catches: a reordered StageOrder, a dropped Release
// step, a re-encoded or copied-out DTO, and ProductionAdmitted set true.
func TestAA30CandidateLinkageOrder(t *testing.T) {
	ref := aa30Reference(t)
	s := aa30Stager(t)
	if s.Next() != FamilyHVAC {
		t.Fatalf("first family %q", s.Next())
	}
	h := aa30HVAC()
	got, err := s.StageHVAC(h)
	if err != nil {
		t.Fatalf("HVAC: %v", err)
	}
	if got.Family != FamilyHVAC || got.DataModel != "1.6.1" || got.ClusterID != 0x0201 || got.Revision != 11 ||
		got.Encoded != 1 || got.Reference != ref || got.ProductionAdmitted || got.HVAC == nil || got.EEM != nil || got.EPM != nil ||
		got.HVAC.Thermostats[0].Temperatures.MeasuredRoom.Wire != 2150 {
		t.Fatalf("HVAC staged %+v", got)
	}
	if s.Next() != "" {
		t.Fatalf("a family is offered while HVAC is staged: %q", s.Next())
	}
	if err := s.Release(FamilyHVAC); err != nil {
		t.Fatal(err)
	}

	got, err = s.StageEEM(aa30EEM())
	if err != nil {
		t.Fatalf("EEM: %v", err)
	}
	if got.Family != FamilyEEM || got.DataModel != "1.6.1" || got.ClusterID != 0x0091 || got.Revision != 2 ||
		got.Encoded != 1 || got.Reference != ref || got.ProductionAdmitted || got.EEM == nil || got.EEM.Energy.MilliWh != 1500 {
		t.Fatalf("EEM staged %+v", got)
	}
	if err := s.Release(FamilyEEM); err != nil {
		t.Fatal(err)
	}

	got, err = s.StageEPM(aa30EPM())
	if err != nil {
		t.Fatalf("EPM: %v", err)
	}
	if got.Family != FamilyEPM || got.DataModel != "1.6.1" || got.ClusterID != 0x0090 || got.Revision != 3 ||
		got.Encoded != 2 || got.Reference != ref || got.ProductionAdmitted || got.EPM == nil ||
		*got.EPM.ActivePower.Value != -250000 || *got.EPM.ActiveCurrent.Value != 1500 {
		t.Fatalf("EPM staged %+v", got)
	}
	if err := s.Release(FamilyEPM); err != nil {
		t.Fatal(err)
	}
	if s.Next() != "" {
		t.Fatalf("a family follows EPM: %q", s.Next())
	}
	if _, err := s.StageHVAC(h); !errors.Is(err, ErrStageOrder) {
		t.Fatalf("HVAC after EPM: %v", err)
	}
}

// TestAA30CandidateBeforeReference varies each reference field alone to its
// zero value and requires refusal before any candidate is staged. Fields and
// the mutation each catches: generation, sensor, fabric and receipts (each
// dropped from newReferenceQualification or valid); a zero value, a nil
// Stager and a Stager built without NewStager (admit's reference check).
func TestAA30CandidateBeforeReference(t *testing.T) {
	cases := []struct {
		name       string
		generation uint64
		sensor     string
		fabric     uint8
		receipts   int
	}{
		{"generation", 0, "helianthus.reference/sensor-a", 1, 1},
		{"sensor", 1, "", 1, 1},
		{"fabric", 1, "helianthus.reference/sensor-a", 0, 1},
		{"receipts", 1, "helianthus.reference/sensor-a", 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := newReferenceQualification(tc.generation, tc.sensor, tc.fabric, tc.receipts)
			if !errors.Is(err, ErrReferenceUnqualified) || q != (ReferenceQualification{}) {
				t.Fatalf("qualification %+v, %v", q, err)
			}
			forged := ReferenceQualification{tc.generation, tc.sensor, tc.fabric, tc.receipts}
			if s, err := NewStager(forged); !errors.Is(err, ErrReferenceUnqualified) || s != nil {
				t.Fatalf("stager from a forged reference: %v", err)
			}
		})
	}
	if s, err := NewStager(ReferenceQualification{}); !errors.Is(err, ErrReferenceUnqualified) || s != nil {
		t.Fatalf("zero reference: %v", err)
	}
	var none *Stager
	if _, err := none.StageHVAC(aa30HVAC()); !errors.Is(err, ErrReferenceUnqualified) {
		t.Fatalf("nil stager: %v", err)
	}
	if _, err := (&Stager{}).StageHVAC(aa30HVAC()); !errors.Is(err, ErrReferenceUnqualified) {
		t.Fatalf("stager without a reference: %v", err)
	}
	// Positive control: the same DTO stages once the reference is qualified.
	if _, err := aa30Stager(t).StageHVAC(aa30HVAC()); err != nil {
		t.Fatalf("qualified reference: %v", err)
	}
}

// TestAA30CandidateOrderRefusals catches: an order check that accepts any
// family, a missing active-stage check, Release accepting another family, and
// Release advancing without a staged candidate.
func TestAA30CandidateOrderRefusals(t *testing.T) {
	s := aa30Stager(t)
	if _, err := s.StageEEM(aa30EEM()); !errors.Is(err, ErrStageOrder) {
		t.Fatalf("EEM first: %v", err)
	}
	if _, err := s.StageEPM(aa30EPM()); !errors.Is(err, ErrStageOrder) {
		t.Fatalf("EPM first: %v", err)
	}
	if err := s.Release(FamilyHVAC); !errors.Is(err, ErrStageOrder) || s.Next() != FamilyHVAC {
		t.Fatalf("release with nothing staged: %v, next %q", err, s.Next())
	}
	if _, err := s.StageHVAC(aa30HVAC()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StageHVAC(aa30HVAC()); !errors.Is(err, ErrStageActive) {
		t.Fatalf("HVAC twice: %v", err)
	}
	if _, err := s.StageEEM(aa30EEM()); !errors.Is(err, ErrStageActive) {
		t.Fatalf("EEM while HVAC is staged: %v", err)
	}
	if err := s.Release(FamilyEEM); !errors.Is(err, ErrStageOrder) {
		t.Fatalf("release of another family: %v", err)
	}
	if err := s.Release(FamilyHVAC); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StageEPM(aa30EPM()); !errors.Is(err, ErrStageOrder) {
		t.Fatalf("EPM before EEM: %v", err)
	}
	if _, err := s.StageEEM(aa30EEM()); err != nil {
		t.Fatalf("EEM after HVAC: %v", err)
	}
}

// TestAA30CandidateDTORefusals varies each checked DTO field alone, all others
// valid, and requires the refusal not to consume the stage: the unmutated DTO
// stages next. Each case catches the omission of its own comparison.
func TestAA30CandidateDTORefusals(t *testing.T) {
	hvacCases := []struct {
		name   string
		mutate func(*hvac.Result161)
		want   error
	}{
		{"data model", func(r *hvac.Result161) { r.Target.DataModel = "1.7" }, ErrCandidateTarget},
		{"cluster", func(r *hvac.Result161) { r.Target.ClusterID = 0x0202 }, ErrCandidateTarget},
		{"revision", func(r *hvac.Result161) { r.Target.Revision = 10 }, ErrCandidateTarget},
		{"admitted", func(r *hvac.Result161) { r.Validation.ProductionAdmitted = true }, ErrCandidateAdmitted},
		{"no value", func(r *hvac.Result161) { r.Thermostats[0].Temperatures = hvac.Temperatures161{} }, ErrCandidateEmpty},
		{"no thermostat", func(r *hvac.Result161) { r.Thermostats = nil }, ErrCandidateEmpty},
	}
	for _, tc := range hvacCases {
		t.Run("hvac "+tc.name, func(t *testing.T) {
			s, r := aa30Stager(t), aa30HVAC()
			tc.mutate(&r)
			if _, err := s.StageHVAC(r); !errors.Is(err, tc.want) {
				t.Fatalf("%v, want %v", err, tc.want)
			}
			if _, err := s.StageHVAC(aa30HVAC()); err != nil {
				t.Fatalf("control after refusal: %v", err)
			}
		})
	}
	// A SystemMode alone is an encoded value.
	mode := aa30HVAC()
	mode.Thermostats[0].Temperatures = hvac.Temperatures161{}
	mode.Thermostats[0].SystemMode = &hvac.SystemMode161{Target: "HEAT", Wire: 4}
	if st, err := aa30Stager(t).StageHVAC(mode); err != nil || st.Encoded != 1 {
		t.Fatalf("SystemMode only: %+v, %v", st, err)
	}

	eemCases := []struct {
		name   string
		mutate func(*eem.Result161)
		want   error
	}{
		{"data model", func(r *eem.Result161) { r.Target.DataModel = "1.7" }, ErrCandidateTarget},
		{"cluster", func(r *eem.Result161) { r.Target.ClusterID = 0x0090 }, ErrCandidateTarget},
		{"revision", func(r *eem.Result161) { r.Target.Revision = 1 }, ErrCandidateTarget},
		{"unit", func(r *eem.Result161) { r.Target.Unit = "MWh" }, ErrCandidateTarget},
		{"admitted", func(r *eem.Result161) { r.ProductionAdmitted = true }, ErrCandidateAdmitted},
		{"no value", func(r *eem.Result161) { r.Energy = nil }, ErrCandidateEmpty},
	}
	for _, tc := range eemCases {
		t.Run("eem "+tc.name, func(t *testing.T) {
			s := aa30Stager(t)
			if _, err := s.StageHVAC(aa30HVAC()); err != nil {
				t.Fatal(err)
			}
			if err := s.Release(FamilyHVAC); err != nil {
				t.Fatal(err)
			}
			r := aa30EEM()
			tc.mutate(&r)
			if _, err := s.StageEEM(r); !errors.Is(err, tc.want) {
				t.Fatalf("%v, want %v", err, tc.want)
			}
			if _, err := s.StageEEM(aa30EEM()); err != nil {
				t.Fatalf("control after refusal: %v", err)
			}
		})
	}

	epmCases := []struct {
		name   string
		mutate func(*epm.Result161)
		want   error
	}{
		{"data model", func(r *epm.Result161) { r.Target.DataModel = "1.7" }, ErrCandidateTarget},
		{"cluster", func(r *epm.Result161) { r.Target.ClusterID = 0x0091 }, ErrCandidateTarget},
		{"revision", func(r *epm.Result161) { r.Target.Revision = 2 }, ErrCandidateTarget},
		{"admitted", func(r *epm.Result161) { r.ProductionAdmitted = true }, ErrCandidateAdmitted},
		{"no metadata", func(r *epm.Result161) { r.Metadata = nil }, ErrCandidateEmpty},
		{"no slot", func(r *epm.Result161) { r.ActivePower, r.ActiveCurrent, r.Voltage = nil, nil, nil }, ErrCandidateEmpty},
	}
	for _, tc := range epmCases {
		t.Run("epm "+tc.name, func(t *testing.T) {
			s := aa30Stager(t)
			for _, step := range []func() error{
				func() error { _, err := s.StageHVAC(aa30HVAC()); return err },
				func() error { return s.Release(FamilyHVAC) },
				func() error { _, err := s.StageEEM(aa30EEM()); return err },
				func() error { return s.Release(FamilyEEM) },
			} {
				if err := step(); err != nil {
					t.Fatal(err)
				}
			}
			r := aa30EPM()
			tc.mutate(&r)
			if _, err := s.StageEPM(r); !errors.Is(err, tc.want) {
				t.Fatalf("%v, want %v", err, tc.want)
			}
			if _, err := s.StageEPM(aa30EPM()); err != nil {
				t.Fatalf("control after refusal: %v", err)
			}
		})
	}
}

// TestAA30CandidateStaticBoundary keeps the linkage SDK-free: candidates.go
// carries the package's build tag and imports only the standard library and
// the three merged candidate encoders.
func TestAA30CandidateStaticBoundary(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "candidates.go", nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Comments) == 0 || f.Comments[0].List[0].Text != "//go:build gofabric" {
		t.Fatal("candidates.go lacks the gofabric build constraint")
	}
	allowed := map[string]bool{
		"errors": true, "fmt": true,
		"github.com/Project-Helianthus/helianthus-matter-binding/candidates/eem":  true,
		"github.com/Project-Helianthus/helianthus-matter-binding/candidates/epm":  true,
		"github.com/Project-Helianthus/helianthus-matter-binding/candidates/hvac": true,
	}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if !allowed[path] || strings.Contains(path, "go-fabric") {
			t.Errorf("candidates.go imports %q", path)
		}
	}
}
