package buildrecord

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func aa06Copy[T any](value T) T {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var copy T
	if err := json.Unmarshal(data, &copy); err != nil {
		panic(err)
	}
	return copy
}

// These synthetic declarations test record structure only. They describe no
// retained SDK tag, actual license review or supported product platform.
func aa06Fixture() (AdoptionProfile, AdoptionRecord) {
	build := syntheticRecord()
	profile := AdoptionProfile{Scope: aa06Copy(build.Scope)}
	for _, kind := range []string{"dependency", "matter.js-lineage", "generated-source", "generator-tool", "replacement", "native-requirement", "sqlite-redistribution"} {
		profile.Inventory = append(profile.Inventory, NoticeItem{
			ID: kind + "-a", Kind: kind, Component: "synthetic-" + kind, Version: "synthetic-version-a", Disposition: "included",
			Source: syntheticDigest(kind + "/source"), Licenses: []Digest{syntheticDigest(kind + "/license")}, Notices: []Digest{syntheticDigest(kind + "/notice")}, Rights: Pass, Evidence: []Digest{syntheticDigest(kind + "/evidence")},
		})
	}
	record := AdoptionRecord{Build: build}
	for _, role := range []Role{Producer, Verifier} {
		actor := "producer-role-a"
		if role == Verifier {
			actor = "verifier-role-b"
		}
		for _, target := range build.Scope.Targets {
			record.Executions = append(record.Executions, TargetExecution{Key: Key{"B5", target.ID, role}, Actor: actor, Identity: aa06Copy(Identity{build.Scope.Inputs, target.Target, target.Test}), Method: "executed-target", Evidence: []Digest{syntheticDigest("target-execution")}})
		}
		for _, item := range profile.Inventory {
			record.Notices = append(record.Notices, NoticeRecord{Key: Key{"B8", "", role}, Actor: actor, Identity: aa06Copy(Identity{Inputs: build.Scope.Inputs, Command: build.Scope.Ledger}), Item: aa06Copy(item)})
		}
	}
	return profile, record
}

func aa06Complete(t *testing.T, p AdoptionProfile, r AdoptionRecord) {
	t.Helper()
	if got := AssessAdoption(p, r); got.State != StructurallyComplete || len(got.Findings) != 0 {
		t.Fatalf("positive structural control: %+v", got)
	}
}

func aa06Rejected(t *testing.T, p AdoptionProfile, r AdoptionRecord) {
	t.Helper()
	if got := AssessAdoption(p, r); got.State == StructurallyComplete || len(got.Findings) == 0 {
		t.Fatalf("incomplete/conflicting structure accepted: %+v", got)
	}
}

func TestAA06Complete(t *testing.T) {
	p, r := aa06Fixture()
	beforeP, beforeR := aa06Copy(p), aa06Copy(r)
	aa06Complete(t, p, r)
	if !reflect.DeepEqual(p, beforeP) || !reflect.DeepEqual(r, beforeR) {
		t.Fatal("assessment mutated supplied evidence")
	}
}

// Each case catches removal of the corresponding typed identity comparison:
// release; Source/Dependencies/Toolchain/GeneratorInputs; OS/Architecture/ABI/
// CGO/Compiler/Linker/Tags/Features; CWD/Args/Environment/TestIDs/Fixtures/
// Repeats/TimeoutSeconds. All other values stay valid and unchanged.
func TestAA06Identity(t *testing.T) {
	for _, change := range identityChanges() {
		for _, role := range []Role{Producer, Verifier} {
			for _, row := range []string{"B4", "B5", "B6"} {
				for _, target := range []string{"target-a", "target-b"} {
					t.Run(change.name+"/build/"+string(role)+"/"+row+"/"+target, func(t *testing.T) {
						p, r := aa06Fixture()
						aa06Complete(t, p, r)
						change.apply(&r.Build.Results[resultIndex(r.Build, row, target, role)].Identity)
						aa06Rejected(t, p, r)
					})
				}
			}
		}
		for _, ledger := range []bool{false, true} {
			for _, role := range []Role{Producer, Verifier} {
				t.Run(fmt.Sprintf("%s/additional-join/%t/%s", change.name, ledger, role), func(t *testing.T) {
					p, r := aa06Fixture()
					aa06Complete(t, p, r)
					if ledger {
						index := 0
						if role == Verifier {
							index = len(p.Inventory)
						}
						change.apply(&r.Notices[index].Identity)
					} else {
						index := 0
						if role == Verifier {
							index = 2
						}
						change.apply(&r.Executions[index].Identity)
					}
					aa06Rejected(t, p, r)
				})
			}
		}
	}
}

// Scope is supplied independently of the record, including every target tuple
// and command field. Changing both a result and its result scope cannot rebind
// the profile. This catches a wrapper that trusts only record.Build.Scope.
func TestAA06ProfileJoin(t *testing.T) {
	for _, change := range identityChanges() {
		t.Run(change.name, func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			id := Identity{p.Scope.Inputs, p.Scope.Targets[0].Target, p.Scope.Targets[0].Test}
			change.apply(&id)
			p.Scope.Inputs, p.Scope.Targets[0].Target, p.Scope.Targets[0].Test = id.Inputs, id.Target, id.Command
			aa06Rejected(t, p, r)
		})
	}
}

func TestAA06RequiredEvidence(t *testing.T) {
	cases := []struct {
		name string
		edit func(*AdoptionProfile, *AdoptionRecord)
	}{
		{"arm64-not-declared", func(p *AdoptionProfile, r *AdoptionRecord) {
			p.Scope.Targets = p.Scope.Targets[:1]
			r.Build.Scope.Targets = r.Build.Scope.Targets[:1]
			results := []Result{}
			for _, v := range r.Build.Results {
				if v.Key.Target != "target-b" {
					results = append(results, v)
				}
			}
			r.Build.Results = results
			executions := []TargetExecution{}
			for _, v := range r.Executions {
				if v.Key.Target != "target-b" {
					executions = append(executions, v)
				}
			}
			r.Executions = executions
		}},
		{"cross-compile", func(_ *AdoptionProfile, r *AdoptionRecord) { r.Executions[1].Method = "cross-compile" }},
		{"zero-selected", func(_ *AdoptionProfile, r *AdoptionRecord) {
			r.Build.Results[resultIndex(r.Build, "B5", "target-b", Producer)].Runs[0].Counts = Counts{}
		}},
		{"zero-executed", func(_ *AdoptionProfile, r *AdoptionRecord) {
			r.Build.Results[resultIndex(r.Build, "B5", "target-b", Producer)].Runs[0].Counts = Counts{Selected: 2, Skipped: 2}
		}},
		{"required-skip", func(_ *AdoptionProfile, r *AdoptionRecord) {
			r.Build.Results[resultIndex(r.Build, "B5", "target-b", Producer)].Runs[0].Counts = Counts{Selected: 2, Executed: 1, Passed: 1, Skipped: 1}
		}},
		{"merged-actors", func(p *AdoptionProfile, r *AdoptionRecord) {
			p.Scope.Verifier = p.Scope.Producer
			r.Build.Scope.Verifier = r.Build.Scope.Producer
			for i := range r.Build.Results {
				r.Build.Results[i].Actor = p.Scope.Producer
			}
			for i := range r.Executions {
				r.Executions[i].Actor = p.Scope.Producer
			}
			for i := range r.Notices {
				r.Notices[i].Actor = p.Scope.Producer
			}
		}},
		{"missing-inputs", func(p *AdoptionProfile, r *AdoptionRecord) {
			p.Scope.Inputs = Inputs{}
			r.Build.Scope.Inputs = Inputs{}
		}},
		{"nil-inventory", func(p *AdoptionProfile, _ *AdoptionRecord) { p.Inventory = nil }},
		{"oversized-inventory", func(p *AdoptionProfile, _ *AdoptionRecord) {
			for len(p.Inventory) < 257 {
				p.Inventory = append(p.Inventory, p.Inventory[0])
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			c.edit(&p, &r)
			aa06Rejected(t, p, r)
		})
	}
	p, r := aa06Fixture()
	for i, result := range r.Build.Results {
		t.Run(fmt.Sprintf("missing-build/%d/%s", i, result.Key.Row), func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			r.Build.Results = append(r.Build.Results[:i], r.Build.Results[i+1:]...)
			aa06Rejected(t, p, r)
		})
	}
	for i := range r.Executions {
		t.Run(fmt.Sprintf("missing-execution/%d", i), func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			r.Executions = append(r.Executions[:i], r.Executions[i+1:]...)
			aa06Rejected(t, p, r)
		})
	}
	for i := range r.Notices {
		t.Run(fmt.Sprintf("missing-notice/%d", i), func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			r.Notices = append(r.Notices[:i], r.Notices[i+1:]...)
			aa06Rejected(t, p, r)
		})
	}
	for i := range p.Inventory {
		t.Run(fmt.Sprintf("missing-kind/%d", i), func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			id := p.Inventory[i].ID
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			notices := []NoticeRecord{}
			for _, n := range r.Notices {
				if n.Item.ID != id {
					notices = append(notices, n)
				}
			}
			r.Notices = notices
			aa06Rejected(t, p, r)
		})
	}
}

// Each independent edit catches omission of its dependency/license join axis,
// even when all evidence digests remain syntactically valid.
func TestAA06NoticeJoin(t *testing.T) {
	edits := []struct {
		name string
		edit func(*NoticeRecord)
	}{
		{"row", func(n *NoticeRecord) { n.Key.Row = "B7" }}, {"target", func(n *NoticeRecord) { n.Key.Target = "target-a" }}, {"role", func(n *NoticeRecord) { n.Key.Role = "other" }}, {"actor", func(n *NoticeRecord) { n.Actor = "other-role" }},
		{"id", func(n *NoticeRecord) { n.Item.ID = "other-id" }}, {"kind", func(n *NoticeRecord) { n.Item.Kind = "dependency" }}, {"component", func(n *NoticeRecord) { n.Item.Component = "other-component" }}, {"version", func(n *NoticeRecord) { n.Item.Version = "other-version" }},
		{"source", func(n *NoticeRecord) { n.Item.Source = syntheticDigest("other-source") }}, {"license", func(n *NoticeRecord) { n.Item.Licenses = []Digest{syntheticDigest("other-license")} }}, {"notice", func(n *NoticeRecord) { n.Item.Notices = []Digest{syntheticDigest("other-notice")} }},
		{"rights", func(n *NoticeRecord) { n.Item.Rights = Unverified }}, {"disposition", func(n *NoticeRecord) { n.Item.Disposition = "absent" }}, {"evidence", func(n *NoticeRecord) { n.Item.Evidence = []Digest{syntheticDigest("other-evidence")} }},
	}
	for _, edit := range edits {
		for _, index := range []int{1, 8} {
			t.Run(fmt.Sprintf("%s/%d", edit.name, index), func(t *testing.T) {
				p, r := aa06Fixture()
				aa06Complete(t, p, r)
				edit.edit(&r.Notices[index])
				aa06Rejected(t, p, r)
			})
		}
	}
	for _, field := range []string{"row", "target", "role", "actor", "evidence", "duplicate"} {
		t.Run("execution/"+field, func(t *testing.T) {
			p, r := aa06Fixture()
			aa06Complete(t, p, r)
			switch field {
			case "row":
				r.Executions[0].Key.Row = "B4"
			case "target":
				r.Executions[0].Key.Target = "outside"
			case "role":
				r.Executions[0].Key.Role = "other"
			case "actor":
				r.Executions[0].Actor = p.Scope.Verifier
			case "evidence":
				r.Executions[0].Evidence = nil
			case "duplicate":
				r.Executions = append(r.Executions, r.Executions[0])
			}
			aa06Rejected(t, p, r)
		})
	}
}

func TestAA06Closure(t *testing.T) {
	for index := 0; index < 7; index++ {
		for _, field := range []string{"license", "notice", "rights", "evidence", "source", "kind", "disposition", "component", "version", "duplicate", "malformed-license", "malformed-notice"} {
			t.Run(fmt.Sprintf("%d/%s", index, field), func(t *testing.T) {
				p, r := aa06Fixture()
				aa06Complete(t, p, r)
				item := &p.Inventory[index]
				switch field {
				case "license":
					item.Licenses = nil
				case "notice":
					item.Notices = nil
				case "rights":
					item.Rights = Unverified
				case "evidence":
					item.Evidence = nil
				case "source":
					item.Source = ""
				case "kind":
					item.Kind = "unknown"
				case "disposition":
					item.Disposition = "unknown"
				case "component":
					item.Component = ""
				case "version":
					item.Version = ""
				case "duplicate":
					p.Inventory = append(p.Inventory, *item)
				case "malformed-license":
					item.Licenses = []Digest{"malformed"}
				case "malformed-notice":
					item.Notices = []Digest{"malformed"}
				}
				// Rejoin the changed declaration so failure must come from closure rules,
				// rather than merely a mismatched result item.
				for i := range r.Notices {
					if r.Notices[i].Item.ID == item.ID {
						r.Notices[i].Item = aa06Copy(*item)
					}
				}
				aa06Rejected(t, p, r)
			})
		}
	}
	for _, kind := range []string{"replacement", "native-requirement", "sqlite-redistribution"} {
		t.Run("evidenced-exclusion/"+kind, func(t *testing.T) {
			p, r := aa06Fixture()
			for i := range p.Inventory {
				item := &p.Inventory[i]
				if item.Kind == kind {
					item.Disposition = "absent"
					if kind == "sqlite-redistribution" {
						item.Disposition = "not-redistributed"
					}
					item.Licenses = []Digest{}
					item.Notices = []Digest{}
					for j := range r.Notices {
						if r.Notices[j].Item.ID == item.ID {
							r.Notices[j].Item = aa06Copy(*item)
						}
					}
				}
			}
			aa06Complete(t, p, r)
			for i := range p.Inventory {
				if p.Inventory[i].Kind == kind {
					p.Inventory[i].Evidence = nil
					for j := range r.Notices {
						if r.Notices[j].Item.ID == p.Inventory[i].ID {
							r.Notices[j].Item = aa06Copy(p.Inventory[i])
						}
					}
				}
			}
			aa06Rejected(t, p, r)
		})
	}
}
