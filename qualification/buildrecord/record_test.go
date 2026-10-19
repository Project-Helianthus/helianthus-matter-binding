package buildrecord

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"
)

func syntheticDigest(label string) Digest {
	return Digest(fmt.Sprintf("%x", sha256.Sum256([]byte("synthetic/"+label))))
}

// Expectations are constructed directly from public synthetic values; no
// production key builder or validation helper supplies the expected rows.
func syntheticRecord() Record {
	inputs := Inputs{UpstreamRelease: "v0.8.0", Source: syntheticDigest("source"), Dependencies: syntheticDigest("dependencies"), Toolchain: syntheticDigest("toolchain"), GeneratorInputs: syntheticDigest("generator-inputs")}
	command := func(name string) Command {
		return Command{CWD: "fixture", Args: []string{"synthetic-action", name}, Environment: map[string]string{"SYNTHETIC_MODE": "offline"}, TestIDs: []string{"case-a", "case-b"}, Fixtures: []Digest{syntheticDigest("fixture")}, Repeats: 2, TimeoutSeconds: 10}
	}
	r := Record{Scope: Scope{Inputs: inputs, Ledger: command("ledger"), Producer: "producer-role-a", Verifier: "verifier-role-b"}}
	for n, arch := range []string{"amd64", "arm64"} {
		name := []string{"target-a", "target-b"}[n]
		target := Target{OS: "linux", Architecture: arch, ABI: "synthetic-abi", CGO: "off", Compiler: "synthetic-compiler", Linker: "synthetic-linker", Tags: []string{"tag-a"}, Features: []string{"feature-a"}}
		r.Scope.Targets = append(r.Scope.Targets, TargetPlan{name, target, command("build"), command("test"), command("generate")})
	}
	for _, role := range []Role{Producer, Verifier} {
		actor := "producer-role-a"
		if role == Verifier {
			actor = "verifier-role-b"
		}
		for _, row := range []string{"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8"} {
			r.Results = append(r.Results, Result{Key: Key{row, "", role}, Actor: actor, Identity: Identity{Inputs: inputs, Command: command("ledger")}, Outcome: Pass, Evidence: []Digest{syntheticDigest("ledger-evidence")}})
		}
		for _, p := range r.Scope.Targets {
			for _, row := range []string{"B4", "B5", "B6"} {
				cmd := command("build")
				if row == "B5" {
					cmd = command("test")
				}
				if row == "B6" {
					cmd = command("generate")
				}
				result := Result{Key: Key{row, p.ID, role}, Actor: actor, Identity: Identity{inputs, p.Target, cmd}, Outcome: Pass, Evidence: []Digest{syntheticDigest("child-evidence")}}
				for i := 0; i < 2; i++ {
					result.Runs = append(result.Runs, Run{Completed: true, Counts: Counts{Selected: 2, Executed: 2, Passed: 2}, Log: syntheticDigest("log"), Output: syntheticDigest("output")})
				}
				r.Results = append(r.Results, result)
			}
		}
	}
	return r
}

func resultIndex(r Record, row, target string, role Role) int {
	for i, v := range r.Results {
		if v.Key == (Key{row, target, role}) {
			return i
		}
	}
	panic("fixture result absent")
}

func TestBuildRecordComplete(t *testing.T) {
	r := syntheticRecord()
	before := syntheticRecord()
	a := Validate(r)
	if a.State != StructurallyComplete || len(a.Findings) != 0 {
		t.Fatalf("complete synthetic structure: %+v", a)
	}
	if !reflect.DeepEqual(r, before) {
		t.Fatal("validator mutated supplied evidence")
	}
	// Only structure is returned; this result has no runtime/admission operation.
	if string(a.State) != "structurally-complete" {
		t.Fatal("unexpected decision vocabulary")
	}
}

type identityChange struct {
	name, field string
	apply       func(*Identity)
}

func identityChanges() []identityChange {
	return []identityChange{
		{"upstream-release", "Inputs.UpstreamRelease", func(i *Identity) { i.Inputs.UpstreamRelease = "v0.8.1" }},
		{"source", "Inputs.Source", func(i *Identity) { i.Inputs.Source = syntheticDigest("other-source") }},
		{"dependencies", "Inputs.Dependencies", func(i *Identity) { i.Inputs.Dependencies = syntheticDigest("other-dependencies") }},
		{"toolchain", "Inputs.Toolchain", func(i *Identity) { i.Inputs.Toolchain = syntheticDigest("other-toolchain") }},
		{"generator-inputs", "Inputs.GeneratorInputs", func(i *Identity) { i.Inputs.GeneratorInputs = syntheticDigest("other-generator") }},
		{"os", "Target.OS", func(i *Identity) { i.Target.OS = "other-os" }},
		{"architecture", "Target.Architecture", func(i *Identity) { i.Target.Architecture = "other-arch" }},
		{"abi", "Target.ABI", func(i *Identity) { i.Target.ABI = "other-abi" }},
		{"cgo", "Target.CGO", func(i *Identity) { i.Target.CGO = "on" }},
		{"compiler", "Target.Compiler", func(i *Identity) { i.Target.Compiler = "other-compiler" }},
		{"linker", "Target.Linker", func(i *Identity) { i.Target.Linker = "other-linker" }},
		{"tags", "Target.Tags", func(i *Identity) { i.Target.Tags = []string{"other-tag"} }},
		{"features", "Target.Features", func(i *Identity) { i.Target.Features = []string{"other-feature"} }},
		{"cwd", "Command.CWD", func(i *Identity) { i.Command.CWD = "other-fixture" }},
		{"args", "Command.Args", func(i *Identity) { i.Command.Args = []string{"synthetic-action", "other-action"} }},
		{"environment", "Command.Environment", func(i *Identity) { i.Command.Environment = map[string]string{"SYNTHETIC_MODE": "other"} }},
		{"test-ids", "Command.TestIDs", func(i *Identity) { i.Command.TestIDs = []string{"case-c", "case-d"} }},
		{"fixtures", "Command.Fixtures", func(i *Identity) { i.Command.Fixtures = []Digest{syntheticDigest("other-fixture")} }},
		{"repeats", "Command.Repeats", func(i *Identity) { i.Command.Repeats = 3 }},
		{"timeout", "Command.TimeoutSeconds", func(i *Identity) { i.Command.TimeoutSeconds = 11 }},
	}
}

func TestBuildRecordIdentity(t *testing.T) {
	for _, change := range identityChanges() {
		t.Run(change.name, func(t *testing.T) {
			for _, role := range []Role{Producer, Verifier} {
				for _, row := range []string{"B4", "B5", "B6"} {
					t.Run(string(role)+"/"+row, func(t *testing.T) {
						r := syntheticRecord()
						i := resultIndex(r, row, "target-a", role)
						change.apply(&r.Results[i].Identity)
						a := Validate(r)
						found := false
						for _, f := range a.Findings {
							if f.Key == r.Results[i].Key && f.Reason == "result identity mismatch" {
								found = true
							}
						}
						if a.State != Failed || !found {
							t.Fatalf("%s comparison lost: %+v", change.field, a)
						}
					})
				}
			}
		})
	}
}

func TestBuildRecordLedgerIdentity(t *testing.T) {
	for _, change := range identityChanges() {
		for _, role := range []Role{Producer, Verifier} {
			for _, row := range []string{"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8"} {
				t.Run(change.name+"/"+string(role)+"/"+row, func(t *testing.T) {
					r := syntheticRecord()
					change.apply(&r.Results[resultIndex(r, row, "", role)].Identity)
					if a := Validate(r); a.State != Failed {
						t.Fatalf("ledger identity mismatch accepted: %+v", a)
					}
				})
			}
		}
	}
}

func TestBuildRecordRequiredRows(t *testing.T) {
	fixture := syntheticRecord()
	for index, result := range fixture.Results {
		name := result.Key.Row + "/" + result.Key.Target + "/" + string(result.Key.Role)
		t.Run("missing/"+name, func(t *testing.T) {
			r := syntheticRecord()
			r.Results = append(r.Results[:index], r.Results[index+1:]...)
			if a := Validate(r); a.State != Unresolved {
				t.Fatalf("missing record: %+v", a)
			}
		})
		t.Run("duplicate/"+name, func(t *testing.T) {
			r := syntheticRecord()
			r.Results = append(r.Results, r.Results[index])
			if a := Validate(r); a.State != Failed {
				t.Fatalf("duplicate record: %+v", a)
			}
		})
	}
}

func TestBuildRecordRejection(t *testing.T) {
	cases := []struct {
		name string
		want State
		edit func(*Record)
	}{
		{"empty", Failed, func(r *Record) { *r = Record{} }},
		{"empty-targets", Failed, func(r *Record) { r.Scope.Targets = nil }},
		{"duplicate-target", Failed, func(r *Record) { r.Scope.Targets = append(r.Scope.Targets, r.Scope.Targets[0]) }},
		{"target-alias", Failed, func(r *Record) { r.Scope.Targets[1].Target = r.Scope.Targets[0].Target }},
		{"undeclared-row", Failed, func(r *Record) { r.Results[0].Key.Row = "B9" }},
		{"undeclared-target", Failed, func(r *Record) { r.Results[0].Key.Target = "outside" }},
		{"unknown-role", Failed, func(r *Record) { r.Results[0].Key.Role = "other" }},
		{"same-actor", Failed, func(r *Record) { r.Scope.Verifier = r.Scope.Producer }},
		{"actor-conflict", Failed, func(r *Record) { r.Results[0].Actor = r.Scope.Verifier }},
		{"failed-row", Failed, func(r *Record) { r.Results[0].Outcome = Failure }},
		{"missing-claim", Unresolved, func(r *Record) { r.Results[0].Outcome = Missing }},
		{"unverified", Unresolved, func(r *Record) { r.Results[0].Outcome = Unverified }},
		{"unknown-outcome", Failed, func(r *Record) { r.Results[0].Outcome = "qualified" }},
		{"lost-evidence", Unresolved, func(r *Record) { r.Results[0].Evidence = nil }},
		{"bad-digest", Failed, func(r *Record) { r.Results[0].Evidence = []Digest{"malformed"} }},
		{"bad-input", Failed, func(r *Record) { r.Scope.Inputs.Source = "malformed" }},
		{"parent-execution", Failed, func(r *Record) { r.Results[0].Runs = []Run{{Completed: true}} }},
		{"relative-escape", Failed, func(r *Record) { r.Scope.Ledger.CWD = "../outside" }},
		{"absolute-cwd", Failed, func(r *Record) { r.Scope.Ledger.CWD = "/outside" }},
		{"implicit-environment", Failed, func(r *Record) { r.Scope.Ledger.Environment = nil }},
		{"empty-test-selection", Failed, func(r *Record) { r.Scope.Targets[0].Test.TestIDs = []string{} }},
		{"duplicate-tests", Failed, func(r *Record) { r.Scope.Targets[0].Test.TestIDs = []string{"case-a", "case-a"} }},
		{"generator-one-repeat", Failed, func(r *Record) { r.Scope.Targets[0].Generate.Repeats = 1 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := syntheticRecord()
			c.edit(&r)
			if a := Validate(r); a.State != c.want {
				t.Fatalf("want %s: %+v", c.want, a)
			}
		})
	}
}

func TestBuildRecordExecutions(t *testing.T) {
	cases := []struct {
		name, row string
		want      State
		edit      func(*Result)
	}{
		{"no-runs", "B4", Unresolved, func(r *Result) { r.Runs = nil }},
		{"missing-repeat", "B5", Unresolved, func(r *Result) { r.Runs = r.Runs[:1] }},
		{"extra-repeat", "B4", Unresolved, func(r *Result) { r.Runs = append(r.Runs, r.Runs[0]) }},
		{"exit", "B4", Failed, func(r *Result) { r.Runs[0].Exit = 2 }},
		{"incomplete", "B4", Unresolved, func(r *Result) { r.Runs[0].Completed = false }},
		{"timeout", "B5", Unresolved, func(r *Result) { r.Runs[0].TimedOut = true }},
		{"blocked", "B4", Unresolved, func(r *Result) { r.Runs[0].Blocked = true }},
		{"lost-log", "B4", Unresolved, func(r *Result) { r.Runs[0].Log = "" }},
		{"lost-output", "B6", Unresolved, func(r *Result) { r.Runs[0].Output = "" }},
		{"malformed-log", "B4", Failed, func(r *Result) { r.Runs[0].Log = "bad" }},
		{"zero-selected", "B5", Failed, func(r *Result) { r.Runs[0].Counts = Counts{} }},
		{"zero-executed", "B5", Unresolved, func(r *Result) { r.Runs[0].Counts = Counts{Selected: 2, Skipped: 2} }},
		{"required-skip", "B5", Unresolved, func(r *Result) { r.Runs[0].Counts = Counts{Selected: 2, Executed: 1, Passed: 1, Skipped: 1} }},
		{"failed-tests", "B5", Failed, func(r *Result) { r.Runs[0].Counts = Counts{Selected: 2, Executed: 2, Passed: 1, Failed: 1} }},
		{"contradictory-counts", "B5", Failed, func(r *Result) { r.Runs[0].Counts.Passed = 3 }},
		{"negative-count", "B5", Failed, func(r *Result) { r.Runs[0].Counts.Skipped = -1 }},
		{"generator-drift", "B6", Failed, func(r *Result) { r.Runs[1].Output = syntheticDigest("drift") }},
	}
	for _, c := range cases {
		for _, role := range []Role{Producer, Verifier} {
			t.Run(c.name+"/"+string(role), func(t *testing.T) {
				r := syntheticRecord()
				c.edit(&r.Results[resultIndex(r, c.row, "target-a", role)])
				if a := Validate(r); a.State != c.want {
					t.Fatalf("want %s: %+v", c.want, a)
				}
			})
		}
	}
}

func TestBuildRecordEvidenceLossAndConflict(t *testing.T) {
	r := syntheticRecord()
	if Validate(r).State != StructurallyComplete {
		t.Fatal("bad fixture")
	}
	i := resultIndex(r, "B6", "target-a", Verifier)
	r.Results[i].Runs[0].Output = syntheticDigest("verifier-disagreement")
	r.Results[0].Evidence = nil
	a := Validate(r)
	if a.State != Failed {
		t.Fatalf("failure must precede gap: %+v", a)
	}
	gap, conflict := false, false
	for _, f := range a.Findings {
		gap = gap || f.State == Unresolved
		conflict = conflict || f.Reason == "generator output disagreement"
	}
	if !gap || !conflict {
		t.Fatal("underlying reasons lost")
	}
	r = syntheticRecord()
	r.Results[resultIndex(r, "B4", "target-a", Producer)].Runs[0].Log = ""
	if Validate(r).State != Unresolved {
		t.Fatal("prior completeness survived evidence loss")
	}
}

// setSyntheticRelease updates the whole declaration and every result, so invalid
// release tests cannot pass merely because the fixture has mixed identities.
func setSyntheticRelease(r *Record, release string) {
	r.Scope.Inputs.UpstreamRelease = release
	for n := range r.Results {
		r.Results[n].Identity.Inputs.UpstreamRelease = release
	}
}

func TestBuildRecordReleaseSelection(t *testing.T) {
	for _, release := range []string{"", "v0.7.9", "v0.0.0", "latest", "v0.8.0+", ">=v0.8.0", "v0.8.x", "0.8.0", "v00.8.0", "v0.08.0", "v0.8.00", "v0.8.0-rc.1", "v0.8.0+metadata", " v0.8.0", "v0.8.0 ", "v0.8", "v0.8.0.1", "v0.-8.0", "v0.8.+1", "v0.8.18446744073709551616"} {
		t.Run("reject/"+release, func(t *testing.T) {
			r := syntheticRecord()
			setSyntheticRelease(&r, release)
			if a := Validate(r); a.State != Failed {
				t.Fatalf("invalid release accepted: %+v", a)
			}
		})
	}
	for _, release := range []string{"v0.8.0", "v0.8.1", "v0.9.0", "v0.11.0", "v1.0.0"} {
		t.Run("exact/"+release, func(t *testing.T) {
			r := syntheticRecord()
			setSyntheticRelease(&r, release)
			if a := Validate(r); a.State != StructurallyComplete || len(a.Findings) != 0 {
				t.Fatalf("exact selection is structure only: %+v", a)
			}
		})
	}
	for _, role := range []Role{Producer, Verifier} {
		for _, release := range []string{"", "v0.7.9", "v0.8.1"} {
			t.Run("mixed/"+string(role)+"/"+release, func(t *testing.T) {
				r := syntheticRecord()
				for n := range r.Results {
					if r.Results[n].Key.Role == role {
						r.Results[n].Identity.Inputs.UpstreamRelease = release
					}
				}
				if a := Validate(r); a.State != Failed {
					t.Fatalf("mixed role release accepted: %+v", a)
				}
			})
		}
	}
}

func TestBuildRecordDeclaredTarget(t *testing.T) {
	for _, change := range identityChanges() {
		if len(change.field) < len("Target.") || change.field[:len("Target.")] != "Target." {
			continue
		}
		t.Run("changed/"+change.name, func(t *testing.T) {
			r := syntheticRecord()
			id := Identity{Target: r.Scope.Targets[0].Target}
			change.apply(&id)
			r.Scope.Targets[0].Target = id.Target
			if a := Validate(r); a.State != Failed {
				t.Fatalf("changed declaration reused results: %+v", a)
			}
		})
		t.Run("missing/"+change.name, func(t *testing.T) {
			r := syntheticRecord()
			// Clear one field in the declaration and its children independently of
			// validation helpers; malformed declarations cannot hide behind exact joins.
			clear := func(target *Target) {
				reflect.ValueOf(target).Elem().FieldByName(change.field[len("Target."):]).SetZero()
			}
			clear(&r.Scope.Targets[0].Target)
			for n := range r.Results {
				if r.Results[n].Key.Target == "target-a" {
					clear(&r.Results[n].Identity.Target)
				}
			}
			if a := Validate(r); a.State != Failed {
				t.Fatalf("implicit target field accepted: %+v", a)
			}
		})
	}
}

func TestBuildRecordVersionOnly(t *testing.T) {
	r := Record{Scope: Scope{Inputs: Inputs{UpstreamRelease: "v0.8.0"}}}
	if a := Validate(r); a.State != Failed {
		t.Fatalf("release alone accepted: %+v", a)
	}
	r = syntheticRecord()
	r.Results = nil
	if a := Validate(r); a.State != Unresolved {
		t.Fatalf("declared release/targets replaced evidence: %+v", a)
	}
}
