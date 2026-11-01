package hvac

import (
	"math"
	"sort"
)

// Thermostat target identity and encoding for the adopted Matter 1.6.1 model.
// These are review constants for a read-only DTO, not SDK symbols or a claim
// that the adopted SDK serves this composition.
const (
	Matter161DataModel        = "1.6.1"
	ThermostatCluster         = "Thermostat"
	ThermostatClusterID       = uint32(0x0201)
	ThermostatRevision        = uint16(11)
	TemperatureUnitsPerDegree = 100
	TemperatureMinWire        = int16(-27315)
	TemperatureMaxWire        = int16(32767)
	TemperatureNullWire       = int16(math.MinInt16)
	FeatureHeatBit            = uint32(0x01)
	FeatureCoolBit            = uint32(0x02)
	FeatureAutoBit            = uint32(0x20)
)

// The only reason Validate gives for a nil Celsius; it is matched exactly so a
// row with any other defect can never become a null encoding.
const unavailableReason161 = "temperature unavailable or nonfinite"

// Target161 grades its own claims. DefinitionEvidence covers reported source
// presence of the cluster identity and the HEAT, COOL and AUTO features only.
// EncodingEvidence covers scale, range, null, enum and feature-bit values, which
// were not inspected in the adopted SDK definitions.
type Target161 struct {
	DataModel, Cluster                   string
	ClusterID                            uint32
	Revision                             uint16
	UnitsPerDegree                       int
	MinWire, MaxWire, NullWire           int16
	DefinitionEvidence, EncodingEvidence string
}

// Subject161 joins rows of one thermostat. Domain and Direction have one accepted
// value here; Fact and Target stay per-row. It is never an endpoint or fabric key.
type Subject161 struct {
	Owner, Profile, Scope string
}

// ModeBinding161 explicitly asserts one target enum for one opaque candidate
// label. No enum is ever derived from the text of a label.
type ModeBinding161 struct {
	Equivalent, Target, Reference string // Target is OFF, AUTO, COOL or HEAT.
	Loss                          []string
}

// TargetEvidence161 is the supplied target-side evidence for one subject.
type TargetEvidence161 struct {
	Subject Subject161
	Auto    Support
	Modes   []ModeBinding161
}

// Provenance161 retains native evidence and loss beside every encoded value.
type Provenance161 struct {
	Key      Key
	Evidence Evidence
	Raw      []byte
	Loss     []string
}

// Temperature161 is one role's encoding. Null marks an unavailable sample at
// NullWire: it is no replacement sample for retained last-known-good state.
type Temperature161 struct {
	Role   string
	Wire   int16
	Null   bool
	Source Provenance161
}

// Temperatures161 keeps every role in its own slot. Assigning a role to a target
// attribute needs an accepted equivalence that is not supplied here.
type Temperatures161 struct {
	MeasuredRoom      *Temperature161
	HeatingConfigured *Temperature161
	HeatingManual     *Temperature161
	HeatingEffective  *Temperature161
	CoolingConfigured *Temperature161
	CoolingManual     *Temperature161
	CoolingEffective  *Temperature161
}

// SystemMode161 is the selected mode with its codebook and binding evidence.
type SystemMode161 struct {
	Target                                string
	Wire                                  uint8
	NativeCode, NativeMeaning, Equivalent string
	CodebookReference, BindingReference   string
	CodebookLoss, BindingLoss             []string
	Source                                Provenance161
}

// FeatureState161 is Yes, No or Unknown with the references that support it.
type FeatureState161 struct {
	State      string
	References []string
}

// Composition161 carries FeatureMap only once HEAT, COOL and AUTO are all decided
// and at least one of HEAT and COOL is present. Its other bits are not decided.
type Composition161 struct {
	Heat, Cool, Auto FeatureState161
	FeatureMap       *uint32
}

// Surface161 names a target surface this adapter never encodes.
type Surface161 struct {
	Name, Disposition, Evidence string
}

// RowWithheld161 explains why one grouped row has no slot.
type RowWithheld161 struct {
	Key     Key
	Reasons []string
}

type Thermostat161 struct {
	Subject      Subject161
	Composition  Composition161
	Temperatures Temperatures161
	SystemMode   *SystemMode161
	Withheld     []RowWithheld161
	Reasons      []string // Subject-level composition and evidence defects.
	Surfaces     []Surface161
}

// Result161 keeps the unmodified Validate output, including rows that never join
// a thermostat. Its Unknown outcomes and false admission are not changed by any
// encoding; a populated slot means only that supplied assertions were encodable.
type Result161 struct {
	Target            Target161
	Validation        Result
	Thermostats       []Thermostat161
	UnmatchedEvidence []Subject161
}

// EncodeThermostat161 is pure and detached. It reads candidate output and has no
// command, setter, schedule or SDK path.
func EncodeThermostat161(input []Vector, evidence []TargetEvidence161) Result161 {
	r := Result161{
		Target: Target161{
			DataModel: Matter161DataModel, Cluster: ThermostatCluster, ClusterID: ThermostatClusterID, Revision: ThermostatRevision,
			UnitsPerDegree: TemperatureUnitsPerDegree, MinWire: TemperatureMinWire, MaxWire: TemperatureMaxWire, NullWire: TemperatureNullWire,
			DefinitionEvidence: "Proven", EncodingEvidence: "Hypothesis",
		},
		Validation: Validate(input), Thermostats: []Thermostat161{}, UnmatchedEvidence: []Subject161{},
	}
	groups := make(map[Subject161][]Evaluation)
	for _, e := range r.Validation.Rows {
		k := e.Source.Key
		if k.Domain == "Thermal" && k.Direction == Observation && text(k.Owner) && text(k.Profile) && text(k.Scope) {
			s := Subject161{k.Owner, k.Profile, k.Scope}
			groups[s] = append(groups[s], e)
		}
	}
	supplied := make(map[Subject161][]TargetEvidence161)
	for _, ev := range evidence {
		supplied[ev.Subject] = append(supplied[ev.Subject], ev)
		if _, ok := groups[ev.Subject]; !ok {
			r.UnmatchedEvidence = append(r.UnmatchedEvidence, ev.Subject)
		}
	}
	sort.Slice(r.UnmatchedEvidence, func(i, j int) bool { return subjectLess161(r.UnmatchedEvidence[i], r.UnmatchedEvidence[j]) })
	for s, rows := range groups {
		r.Thermostats = append(r.Thermostats, thermostat161(s, rows, supplied[s]))
	}
	sort.Slice(r.Thermostats, func(i, j int) bool { return subjectLess161(r.Thermostats[i].Subject, r.Thermostats[j].Subject) })
	return r
}

func thermostat161(s Subject161, rows []Evaluation, evidence []TargetEvidence161) Thermostat161 {
	t := Thermostat161{Subject: s, Withheld: []RowWithheld161{}, Reasons: []string{}, Surfaces: surfaces161()}
	var ev *TargetEvidence161
	switch len(evidence) {
	case 0:
	case 1:
		ev = &evidence[0]
	default:
		t.Reasons = append(t.Reasons, "duplicate target evidence")
	}
	t.Composition = compose161(rows, ev, &t.Reasons)
	roles := make(map[string]int)
	for _, e := range rows {
		if e.Candidate != nil || nullable161(e) {
			roles[e.Source.Role]++
		}
	}
	for _, e := range rows {
		reasons := append([]string{}, e.Withheld...)
		if e.Candidate == nil && !nullable161(e) {
			t.Withheld = append(t.Withheld, RowWithheld161{e.Source.Key, reasons})
			continue
		}
		why := ""
		switch {
		case roles[e.Source.Role] > 1:
			why = "ambiguous role within thermostat"
		case e.Source.Kind == "mode":
			var m *SystemMode161
			if m, why = mode161(e, t.Composition, ev); why == "" {
				t.SystemMode = m
			}
		default:
			var v *Temperature161
			if v, why = temperature161(e, t.Composition); why == "" {
				*slot161(&t.Temperatures, e.Source.Role) = v
			}
		}
		if why != "" {
			t.Withheld = append(t.Withheld, RowWithheld161{e.Source.Key, append(reasons, why)})
		}
	}
	return t
}

// Only a measured room sample is nullable, and only when its sole defect is that
// the sample is absent; configured, manual and effective targets are not.
func nullable161(e Evaluation) bool {
	return e.Candidate == nil && e.Source.Kind == "temperature" && e.Source.Role == "measured-room" && e.Source.Celsius == nil &&
		len(e.Withheld) == 1 && e.Withheld[0] == unavailableReason161
}

func temperature161(e Evaluation, c Composition161) (*Temperature161, string) {
	heat, cool, _ := role(e.Source.Role)
	if (heat && c.Heat.State != "Yes") || (cool && c.Cool.State != "Yes") {
		return nil, "target role lacks evidenced feature"
	}
	if nullable161(e) {
		return &Temperature161{e.Source.Role, TemperatureNullWire, true, provenance161(e.Source)}, ""
	}
	wire, why := wire161(*e.Candidate.Celsius)
	if why != "" {
		return nil, why
	}
	return &Temperature161{e.Source.Role, wire, false, provenance161(e.Source)}, ""
}

// wire161 never rounds: the result must be the exact value the source double
// holds, so a sample with more precision than 0.01 C has no encoding.
func wire161(celsius float64) (int16, string) {
	scaled := math.Round(celsius * TemperatureUnitsPerDegree)
	if scaled < float64(TemperatureMinWire) || scaled > float64(TemperatureMaxWire) {
		return 0, "temperature outside target range"
	}
	wire := int16(scaled)
	if float64(wire)/TemperatureUnitsPerDegree != celsius {
		return 0, "temperature not representable at target scale"
	}
	return wire, ""
}

func slot161(t *Temperatures161, role string) **Temperature161 {
	switch role {
	case "measured-room":
		return &t.MeasuredRoom
	case "heating-configured-target":
		return &t.HeatingConfigured
	case "heating-manual-target":
		return &t.HeatingManual
	case "heating-effective-target":
		return &t.HeatingEffective
	case "cooling-configured-target":
		return &t.CoolingConfigured
	case "cooling-manual-target":
		return &t.CoolingManual
	case "cooling-effective-target":
		return &t.CoolingEffective
	}
	panic("unsupported internal temperature role")
}

func mode161(e Evaluation, c Composition161, ev *TargetEvidence161) (*SystemMode161, string) {
	label := e.Candidate.Equivalent
	var bindings []ModeBinding161
	if ev != nil {
		for _, b := range ev.Modes {
			if b.Equivalent == label {
				bindings = append(bindings, b)
			}
		}
	}
	if len(bindings) == 0 {
		return nil, "no explicit enum evidence"
	}
	if len(bindings) > 1 {
		return nil, "duplicate enum evidence"
	}
	b := bindings[0]
	if !text(b.Reference) || !loss(b.Loss) {
		return nil, "incomplete enum evidence"
	}
	wire, requirement, feature, ok := enum161(b.Target, c)
	if !ok {
		return nil, "unsupported target enum"
	}
	for _, m := range e.Source.Codebook {
		if m.Code != e.Source.NativeCode {
			continue
		}
		if m.Requirement != requirement {
			return nil, "target enum contradicts source capability requirement"
		}
		if feature.State != "Yes" {
			return nil, "target enum lacks evidenced feature"
		}
		return &SystemMode161{
			Target: b.Target, Wire: wire, NativeCode: m.Code, NativeMeaning: m.NativeMeaning, Equivalent: label,
			CodebookReference: m.Reference, BindingReference: b.Reference,
			CodebookLoss: stringsCopy(m.Loss), BindingLoss: stringsCopy(b.Loss), Source: provenance161(e.Source),
		}, ""
	}
	panic("validated mode without selected codebook entry")
}

// enum161 is the closed enum set. Any other target mode, including every other
// member of the target enumeration, stays withheld. OFF needs no feature.
func enum161(target string, c Composition161) (wire uint8, requirement string, feature FeatureState161, ok bool) {
	switch target {
	case "OFF":
		return 0, "none", FeatureState161{State: "Yes"}, true
	case "AUTO":
		return 1, "heating-and-cooling", c.Auto, true
	case "COOL":
		return 3, "cooling", c.Cool, true
	case "HEAT":
		return 4, "heating", c.Heat, true
	}
	return 0, "", FeatureState161{}, false
}

func compose161(rows []Evaluation, ev *TargetEvidence161, reasons *[]string) Composition161 {
	c := Composition161{
		Heat: capability161(rows, "heating", func(v Vector) Support { return v.Heating }, reasons),
		Cool: capability161(rows, "cooling", func(v Vector) Support { return v.Cooling }, reasons),
	}
	c.Auto = auto161(c.Heat, c.Cool, ev, reasons)
	if c.Heat.State != "Yes" && c.Cool.State != "Yes" {
		*reasons = append(*reasons, "no evidenced heating or cooling feature")
	} else if c.Heat.State != "Unknown" && c.Cool.State != "Unknown" && c.Auto.State != "Unknown" {
		m := uint32(0)
		if c.Heat.State == "Yes" {
			m |= FeatureHeatBit
		}
		if c.Cool.State == "Yes" {
			m |= FeatureCoolBit
		}
		if c.Auto.State == "Yes" {
			m |= FeatureAutoBit
		}
		c.FeatureMap = &m
	}
	return c
}

// capability161 reads only rows that passed Validate: a withheld row, whose
// evidence failed, can neither supply a capability nor contradict a sibling's.
func capability161(rows []Evaluation, name string, pick func(Vector) Support, reasons *[]string) FeatureState161 {
	yes, no := map[string]bool{}, map[string]bool{}
	for _, e := range rows {
		if e.Candidate == nil {
			continue
		}
		switch s := pick(e.Source); s.State {
		case "Yes":
			yes[s.Reference] = true
		case "No":
			no[s.Reference] = true
		}
	}
	switch {
	case len(yes) > 0 && len(no) > 0:
		*reasons = append(*reasons, "contradictory "+name+" evidence")
		return FeatureState161{"Unknown", references161(yes, no)}
	case len(yes) > 0:
		return FeatureState161{"Yes", references161(yes)}
	case len(no) > 0:
		return FeatureState161{"No", references161(no)}
	}
	return FeatureState161{"Unknown", []string{}}
}

// AUTO needs its own evidence and both capabilities; it is never derived from them.
func auto161(heat, cool FeatureState161, ev *TargetEvidence161, reasons *[]string) FeatureState161 {
	supplied := Support{State: "Unknown"}
	if ev != nil {
		supplied = ev.Auto
	}
	if !support(supplied) {
		*reasons = append(*reasons, "invalid AUTO evidence")
		return FeatureState161{"Unknown", []string{}}
	}
	switch {
	case heat.State == "Yes" && cool.State == "Yes":
		refs := []string{}
		if supplied.Reference != "" {
			refs = append(refs, supplied.Reference)
		}
		return FeatureState161{supplied.State, refs}
	case supplied.State == "Yes":
		*reasons = append(*reasons, "AUTO lacks both heating and cooling")
		return FeatureState161{"Unknown", []string{}}
	case heat.State == "No" && cool.State == "No":
		return FeatureState161{"No", sortedUnion161(heat.References, cool.References)}
	case heat.State == "No":
		return FeatureState161{"No", sortedUnion161(heat.References)}
	case cool.State == "No":
		return FeatureState161{"No", sortedUnion161(cool.References)}
	}
	return FeatureState161{"Unknown", []string{}}
}

func references161(sets ...map[string]bool) []string {
	out := []string{}
	for _, set := range sets {
		for ref := range set {
			out = append(out, ref)
		}
	}
	sort.Strings(out)
	return out
}

func sortedUnion161(lists ...[]string) []string {
	set := map[string]bool{}
	for _, list := range lists {
		for _, ref := range list {
			set[ref] = true
		}
	}
	return references161(set)
}

func provenance161(v Vector) Provenance161 {
	return Provenance161{v.Key, v.Evidence, append([]byte(nil), v.Raw...), stringsCopy(v.Loss)}
}

func subjectLess161(a, b Subject161) bool {
	if a.Owner != b.Owner {
		return a.Owner < b.Owner
	}
	if a.Profile != b.Profile {
		return a.Profile < b.Profile
	}
	return a.Scope < b.Scope
}

// surfaces161 lists what stays out of the DTO. Their support and every mandatory
// closure question stay Unknown until the real SDK definitions are inspected.
func surfaces161() []Surface161 {
	names := []string{
		"ControlSequenceOfOperation", "RunningState", "demand", "events", "hold", "local-temperature-not-exposed",
		"occupancy", "presets", "schedules", "setpoint-limits", "setpoint-writes", "system-mode-writes", "write-commands",
	}
	out := make([]Surface161, 0, len(names))
	for _, name := range names {
		out = append(out, Surface161{name, "withheld", "Unknown"})
	}
	return out
}
