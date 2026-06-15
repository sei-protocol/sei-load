package config

import (
	"math/rand/v2"

	"github.com/sei-protocol/sei-load/utils/rng"
)

// Operation identifies one StorageRW contract method.
type Operation uint8

const (
	// OpRmw is the read-modify-write operation; it is the zero value so a
	// zero/nil OperationMix selects rmw, matching the default.
	OpRmw Operation = iota
	OpRead
	OpWrite
)

// SetStream binds the selector to a deterministic sub-stream (nil = unseeded
// global RNG), mirroring GasPicker.SetStream and Distribution.SetStream.
func (m *OperationMix) SetStream(s *rng.Stream) { m.stream = s }

// Select draws one operation in proportion to the configured weights. A zero
// total (all weights zero) falls back to OpRmw so an empty mix is the default
// rather than a panic.
func (m *OperationMix) Select() Operation {
	total := m.Read + m.Write + m.Rmw
	if total == 0 {
		return OpRmw
	}
	var u uint64
	if m.stream != nil {
		u = m.stream.Uint64N(total)
	} else {
		u = rand.Uint64N(total)
	}
	if u < m.Rmw {
		return OpRmw
	}
	if u < m.Rmw+m.Read {
		return OpRead
	}
	return OpWrite
}
