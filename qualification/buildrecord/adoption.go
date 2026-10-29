package buildrecord

import (
	"reflect"
	"strings"
)

// AdoptionProfile is a separately supplied expectation, never inferred from
// results. It pins an exact tag, four input manifests, commands and target tuples.
// Inventory is the finite dependency/generated-input closure for those inputs;
// completeness of that supplied inventory is not a source or license audit.
type AdoptionProfile struct {
	Scope     Scope
	Inventory []NoticeItem
}

// NoticeItem identifies one dependency or generated input and its redistribution
// scope. Required kinds are dependency, matter.js-lineage, generated-source,
// generator-tool, replacement, native-requirement and sqlite-redistribution.
// Included items require supplied license and notice references. Only replacement
// and native-requirement may be absent; SQLite may be not-redistributed. All
// dispositions require evidence, and unknown rights never close the ledger.
type NoticeItem struct {
	ID, Kind, Component, Version, Disposition string
	Source                                    Digest
	Licenses, Notices                         []Digest
	Rights                                    Outcome
	Evidence                                  []Digest
}

type NoticeRecord struct {
	Key      Key // B8 parent, separately supplied for producer and verifier.
	Actor    string
	Identity Identity
	Item     NoticeItem
}

// TargetExecution distinguishes an executed B5 target from a cross-compile.
// Method must be executed-target; its evidence is a supplied claim, not platform
// qualification. The corresponding B5 runs and counts are checked by Validate.
type TargetExecution struct {
	Key      Key
	Actor    string
	Identity Identity
	Method   string
	Evidence []Digest
}

type AdoptionRecord struct {
	Build      Record
	Executions []TargetExecution
	Notices    []NoticeRecord
}

// AssessAdoption retains all B1-B8 findings and adds adoption-specific closure
// checks. It neither authenticates evidence nor changes runtime admission.
func AssessAdoption(profile AdoptionProfile, record AdoptionRecord) Assessment {
	a := Validate(record.Build)
	add := func(k Key, state State, reason string) {
		a.Findings = append(a.Findings, Finding{k, state, reason})
		if state == Failed || a.State == StructurallyComplete {
			a.State = state
		}
	}
	if !reflect.DeepEqual(profile.Scope, record.Build.Scope) {
		add(Key{}, Failed, "adoption scope mismatch")
	}
	// Validate the declared profile as well as the result scope. An incomplete
	// profile must not become complete by borrowing identities from results.
	if !validInputs(profile.Scope.Inputs) {
		add(Key{}, Failed, "invalid supplied exact-tag inputs")
	}
	architectures := map[string]bool{}
	expected := map[Key]Identity{}
	for _, target := range profile.Scope.Targets {
		if target.Target.OS == "linux" {
			architectures[target.Target.Architecture] = true
		}
		for _, role := range []Role{Producer, Verifier} {
			expected[Key{"B5", target.ID, role}] = Identity{profile.Scope.Inputs, target.Target, target.Test}
		}
	}
	if !architectures["amd64"] || !architectures["arm64"] {
		add(Key{}, Unresolved, "Linux amd64 and arm64 children required")
	}
	seen := map[Key]bool{}
	for _, execution := range record.Executions {
		k := execution.Key
		id, ok := expected[k]
		if !ok || seen[k] || !reflect.DeepEqual(id, execution.Identity) || execution.Actor != adoptionActor(profile.Scope, k.Role) {
			add(k, Failed, "target execution join mismatch")
		}
		seen[k] = true
		if execution.Method != "executed-target" {
			add(k, Unresolved, "B5 requires target execution, not cross-compile")
		}
		checkAdoptionReferences(execution.Evidence, k, add)
	}
	for k := range expected {
		if !seen[k] {
			add(k, Unresolved, "missing target execution")
		}
	}
	// A bounded declaration makes omission/duplication visible without discovering
	// or downloading dependencies at validation time.
	const maxItems = 256
	if len(profile.Inventory) == 0 || len(profile.Inventory) > maxItems {
		add(Key{}, Failed, "inventory must contain 1..256 items")
	}
	kinds := map[string]bool{}
	items := map[string]NoticeItem{}
	for _, item := range profile.Inventory {
		if _, duplicate := items[item.ID]; duplicate || strings.TrimSpace(item.ID) == "" {
			add(Key{}, Failed, "invalid or duplicate inventory ID")
		}
		items[item.ID] = item
		kinds[item.Kind] = true
		switch item.Kind {
		case "dependency", "matter.js-lineage", "generated-source", "generator-tool", "replacement", "native-requirement", "sqlite-redistribution":
		default:
			add(Key{}, Failed, "unknown inventory kind")
		}
		if strings.TrimSpace(item.Component) == "" || strings.TrimSpace(item.Version) == "" || !validDigest(item.Source) {
			add(Key{}, Failed, "invalid inventory source identity")
		}
		if item.Rights != Pass {
			add(Key{}, Unresolved, "missing or unknown input rights")
		}
		switch item.Disposition {
		case "included":
			checkAdoptionReferences(item.Licenses, Key{}, add)
			checkAdoptionReferences(item.Notices, Key{}, add)
		case "absent":
			if item.Kind != "replacement" && item.Kind != "native-requirement" {
				add(Key{}, Failed, "unsupported absence")
			}
			if item.Licenses == nil || item.Notices == nil || len(item.Licenses)+len(item.Notices) != 0 {
				add(Key{}, Failed, "absence requires explicit empty license/notice sets")
			}
		case "not-redistributed":
			if item.Kind != "sqlite-redistribution" {
				add(Key{}, Failed, "unsupported redistribution exclusion")
			}
			if item.Licenses == nil || item.Notices == nil || len(item.Licenses)+len(item.Notices) != 0 {
				add(Key{}, Failed, "exclusion requires explicit empty license/notice sets")
			}
		default:
			add(Key{}, Unresolved, "unknown redistribution scope")
		}
		checkAdoptionReferences(item.Evidence, Key{}, add)
	}
	for _, kind := range []string{"dependency", "matter.js-lineage", "generated-source", "generator-tool", "replacement", "native-requirement", "sqlite-redistribution"} {
		if !kinds[kind] {
			add(Key{}, Unresolved, "missing inventory kind: "+kind)
		}
	}
	type noticeKey struct {
		ID   string
		Role Role
	}
	noticed := map[noticeKey]bool{}
	if len(record.Notices) > 2*maxItems {
		add(Key{}, Failed, "notice ledger exceeds bound")
	}
	for _, notice := range record.Notices {
		k := notice.Key
		nk := noticeKey{notice.Item.ID, k.Role}
		item, ok := items[notice.Item.ID]
		id := Identity{Inputs: profile.Scope.Inputs, Command: profile.Scope.Ledger}
		if !ok || noticed[nk] || k.Row != "B8" || k.Target != "" || (k.Role != Producer && k.Role != Verifier) || notice.Actor != adoptionActor(profile.Scope, k.Role) || !reflect.DeepEqual(notice.Identity, id) || !reflect.DeepEqual(notice.Item, item) {
			add(k, Failed, "notice dependency/license join mismatch")
		}
		noticed[nk] = true
	}
	for id := range items {
		for _, role := range []Role{Producer, Verifier} {
			if !noticed[noticeKey{id, role}] {
				add(Key{"B8", "", role}, Unresolved, "missing notice item: "+id)
			}
		}
	}
	return a
}

func adoptionActor(scope Scope, role Role) string {
	if role == Verifier {
		return scope.Verifier
	}
	return scope.Producer
}

func checkAdoptionReferences(refs []Digest, key Key, add func(Key, State, string)) {
	if len(refs) == 0 {
		add(key, Unresolved, "missing supplied license, notice or evidence references")
	}
	seen := map[Digest]bool{}
	for _, ref := range refs {
		if !validDigest(ref) || seen[ref] {
			add(key, Failed, "invalid or duplicate supplied reference")
		}
		seen[ref] = true
	}
}
