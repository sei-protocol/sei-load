package config

import (
	"encoding/json"
	"fmt"
	"maps"
	mrand "math/rand/v2"
	"slices"
	"strings"
)

// The operations the storagerw scenario supports. They are FROZEN wire values;
// see package doc.
const (
	OpRmw   = "rmw"
	OpRead  = "read"
	OpWrite = "write"
)

// The operations a scenario that draws from no basket records. A profile cannot
// weight these, so they are not wire values, but they share the operation
// vocabulary with the names above and reach the same metric dimension. They are
// FROZEN for that reason: a query matches an operation by value.
//
// Each names a call shape, not a scenario. Scenarios that issue the same call
// share a name, and the scenario dimension separates them — so a query can total
// one call shape across scenarios, or split it by scenario, without either
// dimension standing in for the other.
const (
	// OpTransfer is a native value transfer to another account.
	OpTransfer = "transfer"
	// OpSelfTransfer is a zero-value transfer to the sender, which touches one
	// account rather than two.
	OpSelfTransfer = "self_transfer"
	// OpERC20Transfer is ERC20 transfer(address,uint256).
	OpERC20Transfer = "erc20_transfer"
	// OpERC721Mint is ERC721 mint(address,uint256).
	OpERC721Mint = "erc721_mint"
	// OpDisperseEther is Disperse disperseEtherFixed(address[]).
	OpDisperseEther = "disperse_ether"
)

// StorageRWOperations is the operation set the storagerw scenario draws from.
var StorageRWOperations = NewOperationSet(OpRmw, OpRead, OpWrite)

// scenarioOperations maps a scenario's wire name, lowercased, to the operations
// it supports. A scenario absent from the table supports none.
var scenarioOperations = map[string]*OperationSet{
	"storagerw": StorageRWOperations,
}

// operationsFor returns the operations a scenario supports, or nil if it
// supports none.
func operationsFor(scenario string) *OperationSet {
	return scenarioOperations[strings.ToLower(scenario)]
}

// OperationSet is the operations one scenario supports, in the order a weighted
// draw walks them. The first is the scenario's default. Both the names and the
// order are part of the saved-workload contract; see package doc.
type OperationSet struct {
	names []string
	known map[string]struct{}
}

// NewOperationSet declares a scenario's operations in draw order. It panics on
// an empty declaration, which leaves no default, or on a repeated name, which
// would claim two sub-ranges of one draw.
func NewOperationSet(names ...string) *OperationSet {
	if len(names) == 0 {
		panic("operation set: at least one operation is required")
	}
	known := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, repeated := known[name]; repeated {
			panic(fmt.Sprintf("operation set: %q is declared twice", name))
		}
		known[name] = struct{}{}
	}
	return &OperationSet{names: slices.Clone(names), known: known}
}

// Names returns the operations in draw order.
func (s *OperationSet) Names() []string { return slices.Clone(s.names) }

// OperationMix is the relative weighting of a scenario's operations, keyed by
// operation name. The weights need not sum to anything in particular: a per-tx
// draw selects an operation in proportion to its weight over the total.
type OperationMix map[string]uint64

// MarshalJSON omits zero weights, which claim no share of a draw.
func (m OperationMix) MarshalJSON() ([]byte, error) {
	weighted := make(map[string]uint64, len(m))
	for name, weight := range m {
		if weight != 0 {
			weighted[name] = weight
		}
	}
	return json.Marshal(weighted)
}

// validate rejects a mix that is present but cannot select anything, so an
// operator who writes "operations": {} — or misspells every name — gets an error
// instead of a silent default run. It also rejects a name the scenario does not
// support, and weights that sum past uint64. An absent mix is the documented
// default and passes.
func (m OperationMix) validate(scenario string, set *OperationSet) error {
	if m == nil {
		return nil
	}
	if set == nil {
		return fmt.Errorf("scenario %q: operations is set but this scenario has no operations", scenario)
	}

	var total uint64
	for _, name := range slices.Sorted(maps.Keys(m)) {
		if _, ok := set.known[name]; !ok {
			return fmt.Errorf("scenario %q: unknown operation %q (supported: %s)",
				scenario, name, strings.Join(set.names, ", "))
		}
		if total+m[name] < total {
			return fmt.Errorf("scenario %q: operations weights sum past uint64", scenario)
		}
		total += m[name]
	}
	if total == 0 {
		return fmt.Errorf("scenario %q: operations is set but every weight is 0; omit it for the %q default",
			scenario, set.names[0])
	}
	return nil
}

// OperationPicker is one mix resolved against one set: the weighted operations
// in draw order, with the running totals a draw compares against.
type OperationPicker struct {
	names    []string
	cum      []uint64
	total    uint64
	fallback string
}

// Picker resolves mix against the set's draw order once, so a per-transaction
// Select walks a fixed slice and allocates nothing. An absent or all-zero mix
// yields a picker that returns the set's first operation and draws no
// randomness.
//
// Picker panics on a name the set does not declare. A profile reaches it only
// through Scenario.Validate, which rejects that name at load.
func (s *OperationSet) Picker(mix OperationMix) *OperationPicker {
	picker := &OperationPicker{fallback: s.names[0]}
	for _, name := range slices.Sorted(maps.Keys(mix)) {
		if _, ok := s.known[name]; !ok {
			panic(fmt.Sprintf("operation %q is not one of %s", name, strings.Join(s.names, ", ")))
		}
	}
	for _, name := range s.names {
		weight := mix[name]
		if weight == 0 {
			continue
		}
		// Validate rejects a mix whose weights overflow, but a picker built from
		// an unvalidated mix would wrap its cumulative total and draw a workload
		// nobody configured — silently, since every weight still looks sane.
		// Assert it here too rather than trust the caller, the same way an
		// undeclared name is asserted below.
		if picker.total+weight < picker.total {
			panic(fmt.Sprintf("operation weights for %s sum past uint64", strings.Join(s.names, ", ")))
		}
		picker.total += weight
		picker.names = append(picker.names, name)
		picker.cum = append(picker.cum, picker.total)
	}
	return picker
}

// Select draws one operation in proportion to its weight.
func (p *OperationPicker) Select(rng *mrand.Rand) string {
	if p.total == 0 {
		return p.fallback
	}
	u := rng.Uint64N(p.total)
	for i, cum := range p.cum {
		if u < cum {
			return p.names[i]
		}
	}
	return p.names[len(p.names)-1]
}
