package tagtriage

import (
	"reflect"
	"testing"
)

func fixture() Record {
	return Record{
		PreviousRelease: "synthetic-previous", NewRelease: "synthetic-next",
		Evidence: []Evidence{{ID: "notes", Kind: ReleaseNotes, Reference: "fixtures/notes.txt", Excerpt: "Synthetic comparison: no changes on the six reviewed axes."}},
		Classifications: []Classification{
			{Binding, NotRelevant, Verified, []string{"notes"}},
			{APIBreaks, NotRelevant, Verified, []string{"notes"}},
			{Commissioning, NotRelevant, Verified, []string{"notes"}},
			{BLE, NotRelevant, Verified, []string{"notes"}},
			{Target, NotRelevant, Verified, []string{"notes"}},
			{Toolchain, NotRelevant, Verified, []string{"notes"}},
		},
	}
}

func TestSyntheticComparisons(t *testing.T) {
	for _, name := range []string{"unchanged", "breaking", "commissioning", "ble-unknown", "missing-input"} {
		t.Run(name, func(t *testing.T) {
			r := fixture()
			want := []Relevance{NotRelevant, NotRelevant, NotRelevant, NotRelevant, NotRelevant, NotRelevant}
			switch name {
			case "breaking":
				r.Evidence[0].Excerpt = "Synthetic API removal affects binding."
				r.Classifications[0].Relevance = Relevant
				r.Classifications[1].Relevance = Relevant
				want[0] = Relevant
				want[1] = Relevant
			case "commissioning":
				r.Evidence[0].Excerpt = "Synthetic new commissioning API; no other changes."
				r.Classifications[0].Relevance = Relevant
				r.Classifications[2].Relevance = Relevant
				want[0] = Relevant
				want[2] = Relevant
			case "ble-unknown":
				r.Evidence[0].Excerpt = "Synthetic BLE timing expectation, delivery unconfirmed."
				for _, i := range []int{0, 3} {
					r.Classifications[i].Relevance = Unknown
					r.Classifications[i].Verification = Unverified
				}
				want[0] = Unknown
				want[3] = Unknown
			case "missing-input":
				r.Evidence = nil
				for i := range r.Classifications {
					r.Classifications[i].Relevance = Unknown
					r.Classifications[i].Verification = Unverified
					r.Classifications[i].EvidenceIDs = nil
					want[i] = Unknown
				}
			}
			got, err := Validate(r)
			if err != nil {
				t.Fatal(err)
			}
			for i, c := range got.Classifications {
				if c.Relevance != want[i] || c.Verification != r.Classifications[i].Verification {
					t.Fatalf("axis %d changed: %+v", i, c)
				}
			}
		})
	}
}

func TestRejectInvalidRecords(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Record)
	}{
		{"missing-previous", func(r *Record) { r.PreviousRelease = "" }},
		{"missing-new", func(r *Record) { r.NewRelease = "" }},
		{"duplicate-release", func(r *Record) { r.NewRelease = r.PreviousRelease }},
		{"padded-previous", func(r *Record) { r.PreviousRelease = " synthetic-previous" }},
		{"padded-new", func(r *Record) { r.NewRelease = "synthetic-next " }},
		{"api-contradiction", func(r *Record) { r.Classifications[1].Relevance = Relevant }},
		{"unknown-binding-contradiction", func(r *Record) {
			r.Classifications[3].Relevance = Unknown
			r.Classifications[3].Verification = Unverified
		}},
		{"duplicate-axis", func(r *Record) { r.Classifications[1].Axis = Binding }},
		{"invalid-axis", func(r *Record) { r.Classifications[1].Axis = "other" }},
		{"invalid-relevance", func(r *Record) { r.Classifications[1].Relevance = "" }},
		{"invalid-verification", func(r *Record) { r.Classifications[1].Verification = "" }},
		{"verified-no-reference", func(r *Record) { r.Classifications[1].EvidenceIDs = nil }},
		{"verified-no-excerpt", func(r *Record) { r.Evidence[0].Excerpt = " " }},
		{"verified-unknown", func(r *Record) { r.Classifications[0].Relevance = Unknown }},
		{"dangling-reference", func(r *Record) { r.Classifications[1].EvidenceIDs = []string{"absent"} }},
		{"duplicate-reference", func(r *Record) { r.Classifications[1].EvidenceIDs = []string{"notes", "notes"} }},
		{"duplicate-evidence", func(r *Record) { r.Evidence = append(r.Evidence, r.Evidence[0]) }},
		{"missing-evidence-id", func(r *Record) { r.Evidence[0].ID = "" }},
		{"missing-evidence-location", func(r *Record) { r.Evidence[0].Reference = "" }},
		{"invalid-evidence-kind", func(r *Record) { r.Evidence[0].Kind = "other" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := fixture()
			tc.mutate(&r)
			got, err := Validate(r)
			if err == nil || !reflect.DeepEqual(got, Record{}) {
				t.Fatalf("accepted invalid record: %+v, %v", got, err)
			}
		})
	}
}

func TestEveryAxisRequiredAndMissingInputStaysUnknown(t *testing.T) {
	// Explicit independent axis list: do not derive expectations from production.
	for _, axis := range []Axis{Binding, APIBreaks, Commissioning, BLE, Target, Toolchain} {
		t.Run(string(axis), func(t *testing.T) {
			r := fixture()
			for i, c := range r.Classifications {
				if c.Axis == axis {
					r.Classifications = append(r.Classifications[:i], r.Classifications[i+1:]...)
					break
				}
			}
			if _, err := Validate(r); err == nil {
				t.Fatal("omitted axis accepted")
			}
			r = fixture()
			r.Evidence = nil
			for i := range r.Classifications {
				r.Classifications[i].Relevance = Unknown
				r.Classifications[i].Verification = Unverified
				r.Classifications[i].EvidenceIDs = nil
			}
			for i := range r.Classifications {
				if r.Classifications[i].Axis == axis {
					r.Classifications[i].Relevance = NotRelevant
				}
			}
			if _, err := Validate(r); err == nil {
				t.Fatal("missing input upgraded to not-relevant")
			}
		})
	}
}

func TestDetachedSlices(t *testing.T) {
	for _, axis := range []Axis{Binding, APIBreaks, Commissioning, BLE, Target, Toolchain} {
		t.Run(string(axis), func(t *testing.T) {
			input := fixture()
			got, err := Validate(input)
			if err != nil {
				t.Fatal(err)
			}
			got.Evidence[0].Excerpt = "changed output"
			for i := range got.Classifications {
				if got.Classifications[i].Axis == axis {
					got.Classifications[i].EvidenceIDs[0] = "changed output"
					got.Classifications[i].Relevance = Unknown
				}
			}
			if !reflect.DeepEqual(input, fixture()) {
				t.Fatal("returned slices alias input")
			}
			got, err = Validate(input)
			if err != nil {
				t.Fatal(err)
			}
			input.Evidence[0].Reference = "changed input"
			for i := range input.Classifications {
				if input.Classifications[i].Axis == axis {
					input.Classifications[i].EvidenceIDs[0] = "changed input"
					input.Classifications[i].Relevance = Unknown
				}
			}
			if !reflect.DeepEqual(got, fixture()) {
				t.Fatal("input slices alias output")
			}
		})
	}
}

func TestUnverifiedDeterminateAndSourceEvidence(t *testing.T) {
	r := fixture()
	r.Evidence[0].Kind = Source
	for i := range r.Classifications {
		r.Classifications[i].Verification = Unverified
	}
	if _, err := Validate(r); err != nil {
		t.Fatal(err)
	}
	r.Evidence[0].Excerpt = ""
	if _, err := Validate(r); err == nil {
		t.Fatal("unverified missing input upgraded to determinate")
	}
}
