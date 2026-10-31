package bridge

import "fmt"

// Reservations is a serial, synthetic node-lifecycle ledger. It allocates
// bookkeeping slots only, never SDK endpoints or production admission.
// All association validation is delegated to Review over complete snapshots.
type Reservations struct {
	node    string
	limit   int64
	history []Association
}

func NewReservations(node string, limit int64, retained []Association) (*Reservations, error) {
	if !token(node) || limit <= 0 {
		return nil, fmt.Errorf("node and positive slot limit required")
	}
	for _, a := range retained {
		if a.Node != node || a.Slot > limit {
			return nil, fmt.Errorf("retained association outside node or slot limit")
		}
	}
	r := Review(nil, retained, nil)
	if len(r.Issues) != 0 {
		return nil, fmt.Errorf("invalid retained associations: %v", r.Issues)
	}
	return &Reservations{node: node, limit: limit, history: append([]Association(nil), retained...)}, nil
}

// Reserve returns an unchanged retained association or reserves the next number
// before a caller can attempt SDK effects. Slot zero asks for a new number.
// High-water allocation includes tombstones and never wraps or fills old gaps.
func (p *Reservations) Reserve(a Association) (Association, error) {
	if a.Node != p.node || a.Retired {
		return Association{}, fmt.Errorf("node mismatch or retired proposal")
	}
	var high int64
	for _, old := range p.history {
		if old.Slot > high {
			high = old.Slot
		}
		if source(old) == source(a) {
			if a.Slot == 0 {
				a.Slot = old.Slot
			}
			if old.Retired || old != a {
				return Association{}, fmt.Errorf("retained identity mismatch")
			}
			return old, nil
		}
	}
	if high >= p.limit {
		return Association{}, fmt.Errorf("slot numbers exhausted")
	}
	next := high + 1
	if a.Slot != 0 && a.Slot != next {
		return Association{}, fmt.Errorf("slot is not next reservation")
	}
	a.Slot = next
	proposed := append(p.Retained(), a)
	r := Review(p.history, proposed, nil)
	if len(r.Issues) != 0 {
		return Association{}, fmt.Errorf("association review failed: %v", r.Issues)
	}
	p.history = proposed
	return a, nil
}

// Deselect retains the entire opaque association. The caller withdraws active
// publication separately; deselection cannot forget identity or free a number.
func (p *Reservations) Deselect(binding string) error {
	for _, a := range p.history {
		if a.Binding == binding {
			return nil
		}
	}
	return fmt.Errorf("unknown binding")
}

// Retire retains a one-way tombstone for the entire node lifecycle.
func (p *Reservations) Retire(binding string) error {
	proposed := p.Retained()
	found := false
	for i := range proposed {
		if proposed[i].Binding == binding {
			proposed[i].Retired = true
			found = true
		}
	}
	if !found {
		return fmt.Errorf("unknown binding")
	}
	r := Review(p.history, proposed, nil)
	if len(r.Issues) != 0 {
		return fmt.Errorf("retirement review failed: %v", r.Issues)
	}
	p.history = proposed
	return nil
}

func (p *Reservations) Retained() []Association { return append([]Association(nil), p.history...) }
