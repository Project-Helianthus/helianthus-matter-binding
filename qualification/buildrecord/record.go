// Package buildrecord checks supplied offline evidence structure. It does not
// acquire, authenticate, execute, qualify, or admit anything.
package buildrecord

import (
	"encoding/hex"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
)

type State string

const (
	StructurallyComplete State = "structurally-complete"
	Unresolved           State = "unresolved"
	Failed               State = "failed"
)

type Outcome string

const (
	Pass       Outcome = "pass"
	Missing    Outcome = "missing"
	Unverified Outcome = "unverified"
	Failure    Outcome = "failed"
)

type Role string

const (
	Producer Role = "producer"
	Verifier Role = "verifier"
)

// Digest addresses supplied evidence; its syntax does not authenticate content.
type Digest string

type Inputs struct {
	// UpstreamRelease selects an exact release; it does not authenticate source.
	UpstreamRelease                                  string
	Source, Dependencies, Toolchain, GeneratorInputs Digest
}

type Target struct {
	OS, Architecture, ABI, CGO, Compiler, Linker string
	Tags, Features                               []string
}

// Command is a selection identity, never an instruction executed by this package.
// Empty optional collections must be explicit non-nil collections.
type Command struct {
	CWD            string
	Args           []string
	Environment    map[string]string
	TestIDs        []string
	Fixtures       []Digest
	Repeats        int
	TimeoutSeconds int
}

type Identity struct {
	Inputs  Inputs
	Target  Target // Zero for ledger parents; populated only for target children.
	Command Command
}

type TargetPlan struct {
	ID                    string
	Target                Target
	Build, Test, Generate Command
}

// Scope is declared before results. Actor tokens label separate supplied
// responsibilities, not authenticated people or authorization principals.
type Scope struct {
	Inputs             Inputs
	Ledger             Command
	Targets            []TargetPlan
	Producer, Verifier string
}

type Key struct {
	Row    string
	Target string // Empty for each of the eight ledger parents.
	Role   Role
}

type Counts struct {
	Selected, Executed, Passed, Failed, Skipped int
}

type Run struct {
	Completed         bool
	Exit              int
	TimedOut, Blocked bool
	Counts            Counts
	Log, Output       Digest
}

type Result struct {
	Key      Key
	Actor    string
	Identity Identity
	Outcome  Outcome
	Evidence []Digest
	Runs     []Run
}

type Record struct {
	Scope   Scope
	Results []Result
}

type Finding struct {
	Key    Key
	State  State
	Reason string
}

// Assessment has no admission or qualification field. Findings retain every gap
// and contradiction, even when a failure takes aggregate precedence.
type Assessment struct {
	State    State
	Findings []Finding
}

func validDigest(d Digest) bool {
	if len(d) != 64 {
		return false
	}
	b, err := hex.DecodeString(string(d))
	return err == nil && len(b) == 32 && string(d) == strings.ToLower(string(d))
}

func validRelease(release string) bool {
	if !strings.HasPrefix(release, "v") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(release, "v"), ".")
	if len(parts) != 3 {
		return false
	}
	var version [3]uint64
	for n, part := range parts {
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil || strconv.FormatUint(value, 10) != part {
			return false
		}
		version[n] = value
	}
	return version[0] > 0 || version[1] >= 8
}

func validInputs(i Inputs) bool {
	return validRelease(i.UpstreamRelease) && validDigest(i.Source) && validDigest(i.Dependencies) && validDigest(i.Toolchain) && validDigest(i.GeneratorInputs)
}

func uniqueText(values []string, nonempty bool) bool {
	if values == nil || (nonempty && len(values) == 0) {
		return false
	}
	seen := map[string]bool{}
	for _, v := range values {
		if strings.TrimSpace(v) == "" || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func validTarget(t Target) bool {
	for _, v := range []string{t.OS, t.Architecture, t.ABI, t.Compiler, t.Linker} {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return (t.CGO == "on" || t.CGO == "off") && uniqueText(t.Tags, false) && uniqueText(t.Features, false)
}

func validCommand(c Command) bool {
	if c.CWD == "" || path.IsAbs(c.CWD) || path.Clean(c.CWD) != c.CWD || c.CWD == ".." || strings.HasPrefix(c.CWD, "../") || strings.ContainsAny(c.CWD, "\\:") {
		return false
	}
	if len(c.Args) == 0 || strings.TrimSpace(c.Args[0]) == "" || c.Environment == nil || !uniqueText(c.TestIDs, false) || c.Fixtures == nil || c.Repeats < 1 || c.TimeoutSeconds < 1 {
		return false
	}
	for k := range c.Environment {
		if strings.TrimSpace(k) == "" || strings.ContainsAny(k, "=\x00") {
			return false
		}
	}
	for _, d := range c.Fixtures {
		if !validDigest(d) {
			return false
		}
	}
	return true
}

// Validate performs exact joins against the declared finite scope. It never
// mutates the record or treats claimed passes as verified source/build facts.
func Validate(record Record) Assessment {
	a := Assessment{State: StructurallyComplete}
	add := func(k Key, s State, reason string) {
		a.Findings = append(a.Findings, Finding{k, s, reason})
		if s == Failed || a.State == StructurallyComplete {
			a.State = s
		}
	}
	s := record.Scope
	if !validInputs(s.Inputs) || !validCommand(s.Ledger) || len(s.Targets) == 0 || strings.TrimSpace(s.Producer) == "" || strings.TrimSpace(s.Verifier) == "" || s.Producer == s.Verifier {
		add(Key{}, Failed, "invalid scope or producer/verifier conflict")
	}
	expected := map[Key]Identity{}
	keys := []Key{}
	declare := func(row, target string, id Identity) {
		for _, role := range []Role{Producer, Verifier} {
			k := Key{row, target, role}
			if _, ok := expected[k]; ok {
				add(k, Failed, "duplicate declared key")
				continue
			}
			expected[k] = id
			keys = append(keys, k)
		}
	}
	for n := 1; n <= 8; n++ {
		declare(fmt.Sprintf("B%d", n), "", Identity{Inputs: s.Inputs, Command: s.Ledger})
	}
	targets := []Target{}
	for _, t := range s.Targets {
		if strings.TrimSpace(t.ID) == "" || !validTarget(t.Target) || !validCommand(t.Build) || !validCommand(t.Test) || !validCommand(t.Generate) || len(t.Test.TestIDs) == 0 || t.Generate.Repeats < 2 {
			add(Key{Target: t.ID}, Failed, "invalid target or selection")
		}
		for _, prior := range targets {
			if reflect.DeepEqual(prior, t.Target) {
				add(Key{Target: t.ID}, Failed, "duplicate target tuple")
			}
		}
		targets = append(targets, t.Target)
		declare("B4", t.ID, Identity{s.Inputs, t.Target, t.Build})
		declare("B5", t.ID, Identity{s.Inputs, t.Target, t.Test})
		declare("B6", t.ID, Identity{s.Inputs, t.Target, t.Generate})
	}
	seen := map[Key]bool{}
	generated := map[string]Digest{}
	for _, r := range record.Results {
		k := r.Key
		id, ok := expected[k]
		if !ok {
			add(k, Failed, "undeclared row, target or role")
			continue
		}
		if seen[k] {
			add(k, Failed, "duplicate result")
		}
		seen[k] = true
		actor := s.Producer
		if k.Role == Verifier {
			actor = s.Verifier
		}
		if r.Actor != actor {
			add(k, Failed, "producer/verifier actor mismatch")
		}
		// Each axis is compared independently; no digest of a partial key substitutes
		// for this complete typed identity comparison.
		if !reflect.DeepEqual(r.Identity, id) {
			add(k, Failed, "result identity mismatch")
		}
		switch r.Outcome {
		case Failure:
			add(k, Failed, "observed failure")
		case Missing, Unverified:
			add(k, Unresolved, "missing or unverified evidence")
		case Pass:
		default:
			add(k, Failed, "unknown outcome")
		}
		if len(r.Evidence) == 0 {
			add(k, Unresolved, "missing evidence references")
		}
		for _, d := range r.Evidence {
			if !validDigest(d) {
				add(k, Failed, "malformed evidence reference")
			}
		}
		if k.Target == "" {
			if len(r.Runs) != 0 {
				add(k, Failed, "execution belongs to target children")
			}
			continue
		}
		if len(r.Runs) != id.Command.Repeats {
			add(k, Unresolved, "incomplete repeat runs")
		}
		for _, run := range r.Runs {
			if run.TimedOut || run.Blocked || !run.Completed {
				add(k, Unresolved, "execution incomplete, timed out or blocked")
			}
			if run.Exit != 0 {
				add(k, Failed, "nonzero execution exit")
			}
			if run.Log == "" || run.Output == "" {
				add(k, Unresolved, "lost log or output reference")
			}
			if (run.Log != "" && !validDigest(run.Log)) || (run.Output != "" && !validDigest(run.Output)) {
				add(k, Failed, "malformed run reference")
			}
			if k.Row == "B5" {
				c := run.Counts
				if c.Selected < 0 || c.Executed < 0 || c.Passed < 0 || c.Failed < 0 || c.Skipped < 0 || c.Executed > c.Selected || c.Skipped > c.Selected || c.Executed != c.Selected-c.Skipped || c.Passed > c.Executed || c.Failed != c.Executed-c.Passed {
					add(k, Failed, "contradictory test counts")
				}
				if c.Failed > 0 {
					add(k, Failed, "failed tests")
				}
				if c.Selected != len(id.Command.TestIDs) {
					add(k, Failed, "test selection count mismatch")
				}
				if c.Executed == 0 || c.Skipped != 0 {
					add(k, Unresolved, "zero executed tests or required skip")
				}
			}
			if k.Row == "B6" && validDigest(run.Output) {
				if prior, exists := generated[k.Target]; exists && prior != run.Output {
					add(k, Failed, "generator output disagreement")
				}
				generated[k.Target] = run.Output
			}
		}
	}
	for _, k := range keys {
		if !seen[k] {
			add(k, Unresolved, "missing required result")
		}
	}
	return a
}
