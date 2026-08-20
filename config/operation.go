package config

import (
	mrand "math/rand/v2"
)

// Operation identifies one StorageRW contract method.
type Operation uint8

const (
	// OpRmw is the read-modify-write operation. It is the zero value so a
	// zero-weight or absent OperationMix selects rmw, matching the default.
	OpRmw Operation = iota
	OpRead
	OpWrite
)

// OperationMix is the relative weighting of the StorageRW read/write/rmw
// operations. The weights need not sum to anything in particular: a per-tx draw
// selects an operation in proportion to its weight over the total. An all-zero
// mix falls back to rmw, the default.
type OperationMix struct {
	Read  uint64 `json:"read,omitempty"`
	Write uint64 `json:"write,omitempty"`
	Rmw   uint64 `json:"rmw,omitempty"`
}

// Select draws one operation in proportion to the configured weights. A zero
// total falls back to OpRmw, so an empty mix is the default rather than a
// division by zero.
//
// The comparison order (rmw, then read, then write) fixes which weight owns
// which sub-range of the draw. It is arbitrary but must stay stable, because
// changing it changes which operation a given draw selects.
func (m *OperationMix) Select(rng *mrand.Rand) Operation {
	total := m.Read + m.Write + m.Rmw
	if total == 0 {
		return OpRmw
	}
	switch u := rng.Uint64N(total); {
	case u < m.Rmw:
		return OpRmw
	case u < m.Rmw+m.Read:
		return OpRead
	default:
		return OpWrite
	}
}
