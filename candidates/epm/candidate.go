// Package epm validates internal contribution-candidate vectors, not Matter
// attributes. It has no SDK, provider, publication, or admission integration.
package epm

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// Key retains independent source and measurement axes. Phase must be explicit
// (use a supplied token such as "none" for a non-phase fact).
type Key struct {
	Owner, Profile, Source, Scope, Phase, Fact string
}

// Input is a caller assertion, not qualified native or accepted target evidence.
// Decimal * 10^Scale is in Unit; OutputScale selects an internal integer grid
// in that SAME unit. Neither scale is a Matter encoding declaration.
type Input struct {
	Key          Key
	Availability string // "available" or "unavailable"; empty/unknown is withheld
	Decimal      string // optional leading minus, digits, optional fractional digits
	Unit         string
	Scale        int
	OutputUnit   string
	OutputScale  int
	Evidence     []byte // opaque native evidence, including on rejection
	Accuracy     []byte // opaque caller metadata; never interpreted or synthesized
}

type Result struct {
	Native Input
	Status string // "exact", "unavailable", or "withheld"
	Reason string
	Value  *int64 // nil unless exact; an exact zero has a non-nil pointer
}

// Validate returns detached, deterministically ordered results for every input.
// Duplicate keys withhold all conflicting rows, never select a convenient value.
// Rejection of one row does not erase another row. No last-known-good store or
// freshness transition is implemented here; those remain with the source owner.
func Validate(inputs []Input) []Result {
	counts := make(map[Key]int, len(inputs))
	for _, in := range inputs {
		counts[in.Key]++
	}
	results := make([]Result, 0, len(inputs))
	for _, in := range inputs {
		in.Evidence = clone(in.Evidence)
		in.Accuracy = clone(in.Accuracy)
		r := Result{Native: in, Status: "withheld"}
		switch {
		case !validKey(in.Key):
			r.Reason = "invalid-identity"
		case counts[in.Key] != 1:
			r.Reason = "duplicate-identity"
		case !knownUnit(in.Unit) || in.OutputUnit != in.Unit:
			r.Reason = "unit-unknown-or-mismatch"
		case in.Scale < -18 || in.Scale > 18 || in.OutputScale < -18 || in.OutputScale > 18:
			r.Reason = "scale-out-of-bounds"
		case in.Availability == "unavailable" && in.Decimal == "":
			r.Status = "unavailable"
		case in.Availability != "available":
			r.Reason = "availability-unknown-or-conflicting"
		default:
			value, reason := convert(in.Decimal, in.Scale, in.OutputScale)
			r.Reason = reason
			if reason == "" {
				r.Status, r.Value = "exact", &value
			}
		}
		results = append(results, r)
	}
	// Quote preserves even invalid UTF-8 in rejected input; no normalization can
	// hide a conflict. This complete order is not a persisted identity format.
	sort.SliceStable(results, func(i, j int) bool {
		return order(results[i].Native) < order(results[j].Native)
	})
	return results
}

func order(in Input) string {
	fields := []string{in.Key.Owner, in.Key.Profile, in.Key.Source, in.Key.Scope, in.Key.Phase, in.Key.Fact,
		in.Availability, in.Decimal, in.Unit, strconv.Itoa(in.Scale), in.OutputUnit, strconv.Itoa(in.OutputScale),
		strconv.FormatBool(in.Evidence == nil), string(in.Evidence), strconv.FormatBool(in.Accuracy == nil), string(in.Accuracy)}
	for i := range fields {
		fields[i] = strconv.Quote(fields[i])
	}
	return strings.Join(fields, "|")
}

func clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	return append([]byte{}, b...)
}

func validKey(k Key) bool {
	for _, s := range []string{k.Owner, k.Profile, k.Source, k.Scope, k.Phase, k.Fact} {
		if len(s) == 0 || len(s) > 128 {
			return false
		}
		for _, c := range s {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-/:", c)) {
				return false
			}
		}
	}
	return true
}

func knownUnit(s string) bool {
	// Local arithmetic labels only; no fact-to-unit or target mapping is implied.
	switch s {
	case "W", "var", "VA", "A", "V", "Hz", "ratio":
		return true
	}
	return false
}

func convert(decimal string, scale, outputScale int) (int64, string) {
	if len(decimal) == 0 || len(decimal) > 128 {
		return 0, "invalid-decimal"
	}
	digits := decimal
	negative := strings.HasPrefix(digits, "-")
	if negative {
		digits = digits[1:]
	}
	parts := strings.Split(digits, ".")
	if len(parts) > 2 {
		return 0, "invalid-decimal"
	}
	fraction := 0
	for i, p := range parts {
		if p == "" {
			return 0, "invalid-decimal"
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return 0, "invalid-decimal"
			}
		}
		if i == 1 {
			fraction = len(p)
		}
	}
	n, _ := new(big.Int).SetString(strings.Join(parts, ""), 10)
	if negative {
		n.Neg(n)
	}
	exponent := scale - outputScale - fraction
	abs := exponent
	if abs < 0 {
		abs = -abs
	}
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs)), nil)
	if exponent >= 0 {
		n.Mul(n, power)
	} else {
		remainder := new(big.Int)
		n.QuoRem(n, power, remainder)
		if remainder.Sign() != 0 {
			return 0, "precision-loss"
		}
	}
	if !n.IsInt64() {
		return 0, "integer-overflow"
	}
	return n.Int64(), ""
}
