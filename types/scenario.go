package types

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// LoadTx is a wrapper that has pre-encoded json rpc payload and eth transaction.
//
// Lifecycle field concurrency contract: a *LoadTx is passed by pointer through
// the buffered txChan. Each lifecycle field (the timestamps and
// SequenceIndex) is written at most once, by whichever goroutine owns the tx at
// that stage, and is immutable thereafter; ownership transfers with the pointer
// across the channels, so the writes need no locking. The open-loop scheduler
// writes IntendedSendTime and SequenceIndex while it solely owns the tx (before
// the sender hand-off); the sender writes AttemptedSendTime; the inclusion
// tracker writes InclusionTime. A zero timestamp means "not recorded" (e.g.
// prewarm txs, or a stage not yet reached) — consumers must treat it as
// untracked, never as the zero epoch.
type LoadTx struct {
	EthTx    *ethtypes.Transaction
	Scenario *TxScenario

	// IntendedSendTime is when the tx was scheduled to be sent. In the open-loop
	// arrival model the scheduler writes the true scheduled instant t₀ + i/λ
	// here (independent of when a sender is free), which is the basis for
	// coordinated-omission-free latency. In the legacy closed-loop model it
	// instead holds the back-pressured enqueue time and must not be used to
	// derive latency. A LoadTx cannot self-describe which model wrote it — an
	// open-loop tx and a closed-loop tx are byte-identical (both can have
	// SequenceIndex 0). Latency and schedule-lag consumers must gate on the
	// run-level arrival model (RunSummary.ArrivalModel), not on any field here.
	IntendedSendTime time.Time
	// SequenceIndex is the monotonic per-campaign arrival index i assigned by
	// the open-loop scheduler, which schedules tx i at t₀ + i/λ. It attributes
	// per-tx schedule lag (IntendedSendTime vs AttemptedSendTime) back to a
	// position in the arrival sequence. Zero in the legacy closed-loop model,
	// where no scheduler assigns it — so the value alone does not identify the
	// model (see IntendedSendTime); the run's arrival model is authoritative.
	SequenceIndex uint64
	// AttemptedSendTime is when the send was actually attempted, written by the
	// sender goroutine that owns the tx between dequeue and send completion.
	AttemptedSendTime time.Time
	// InclusionTime is when the tx was observed included on-chain, written only
	// by the inclusion tracker (single writer, under its registry lock). The
	// clock is the wall-clock instant the including block's newHead header
	// ARRIVES at the tracker (time.Now() at header receipt), cached per block
	// number and applied to every tx matched in that block — NOT the body-fetch
	// completion time and NOT header.Time.
	InclusionTime time.Time
}

// JSONRPCRequest represents json rpc request.
type JSONRPCRequest struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// TxScenario captures the scenario of this test transaction.
type TxScenario struct {
	Name  string
	Nonce uint64
	// Operation is the call shape this transaction issues, from the frozen
	// vocabulary in config. The generator records the scenario's own operation
	// here; a scenario that draws from a weighted basket overwrites it with the
	// draw. It is never empty, so a consumer records the dimension on every
	// sample and needs no placeholder for a scenario that draws nothing.
	Operation string
	Sender    Account
	Receiver  common.Address
}

// NewTxScenario describes one transaction a scenario is about to build. Every
// field is a parameter so a new call site cannot leave one at its zero value:
// an empty Operation reaches the metrics as an empty label, which Prometheus
// treats as absent, collides with the streams that carry a real name, and
// discards a sample with no scrape error to show it.
func NewTxScenario(name, operation string, nonce uint64, sender Account, receiver common.Address) *TxScenario {
	return &TxScenario{
		Name:      name,
		Operation: operation,
		Nonce:     nonce,
		Sender:    sender,
		Receiver:  receiver,
	}
}

// NewSetupTx builds a transaction the run sends to prepare itself, not to offer
// load. It has no timing basis, so IntendedSendTime stays zero and the inclusion
// tracker leaves it out of the latency histogram.
func NewSetupTx(tx *ethtypes.Transaction, scenario *TxScenario) *LoadTx {
	return &LoadTx{EthTx: tx, Scenario: scenario}
}

// NewEnqueuedTx builds a transaction the run offers as load, stamped with the
// instant it was handed to the sender.
//
// That instant is an enqueue time, not an arrival schedule: the sender applies
// back-pressure, so it already carries the delay a schedule would have measured.
// A latency taken from it understates by whatever the pipeline was holding, which
// is the coordinated-omission error. Only a scheduler that fixes an instant
// independent of sender readiness can supply a schedule, and this repo has none;
// see the arrival-model note in sender/doc.go.
func NewEnqueuedTx(tx *ethtypes.Transaction, scenario *TxScenario, enqueuedAt time.Time) *LoadTx {
	return &LoadTx{EthTx: tx, Scenario: scenario, IntendedSendTime: enqueuedAt}
}

// CreateTxFromEthTx creates a LoadTx from an EthTx (pre-marshaled).
func CreateTxFromEthTx(tx *ethtypes.Transaction, scenario *TxScenario) *LoadTx {
	// Return the complete LoadTx object
	return &LoadTx{
		EthTx:    tx,
		Scenario: scenario,
	}
}
