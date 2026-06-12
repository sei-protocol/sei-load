package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// LoadTx is a wrapper that has pre-encoded json rpc payload and eth transaction.
//
// Lifecycle field concurrency contract: a *LoadTx is passed by pointer through
// buffered channels (txChan, sentTxs). Each lifecycle field (the timestamps and
// SequenceIndex) is written at most once, by whichever goroutine owns the tx at
// that stage, and is immutable thereafter; ownership transfers with the pointer
// across the channels, so the writes need no locking. The open-loop scheduler
// writes IntendedSendTime and SequenceIndex while it solely owns the tx (before
// the worker hand-off); the worker writes AttemptedSendTime; the inclusion
// tracker writes InclusionTime. A zero timestamp means "not recorded" (e.g.
// prewarm txs, or a stage not yet reached) — consumers must treat it as
// untracked, never as the zero epoch.
type LoadTx struct {
	EthTx          *ethtypes.Transaction
	JSONRPCPayload []byte
	Payload        []byte
	Scenario       *TxScenario

	// IntendedSendTime is when the tx was scheduled to be sent. In the open-loop
	// arrival model the scheduler writes the true scheduled instant t₀ + i/λ
	// here (independent of when a sender is free), which is the basis for
	// coordinated-omission-free latency. In the legacy closed-loop model it
	// instead holds the back-pressured enqueue time and must not be used to
	// derive latency. SequenceIndex disambiguates which model produced it: a
	// nonzero SequenceIndex (or index 0 from a campaign start) marks the
	// open-loop scheduler as the writer.
	IntendedSendTime time.Time
	// SequenceIndex is the monotonic per-campaign arrival index i assigned by
	// the open-loop scheduler, which schedules tx i at t₀ + i/λ. It attributes
	// per-tx schedule lag (IntendedSendTime vs AttemptedSendTime) back to a
	// position in the arrival sequence. Zero in the legacy closed-loop model,
	// where no scheduler assigns it.
	SequenceIndex uint64
	// AttemptedSendTime is when the send was actually attempted, written by the
	// worker goroutine that owns the tx between dequeue and the sentTxs hand-off.
	AttemptedSendTime time.Time
	// InclusionTime is when the tx was observed included on-chain, written only
	// by the inclusion tracker.
	InclusionTime time.Time
}

// JSONRPCRequest represents json rpc request.
type JSONRPCRequest struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func toJSONRequestBytes(rawTx []byte) ([]byte, error) {
	req := &JSONRPCRequest{
		Version: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  json.RawMessage(fmt.Sprintf(`["0x%x"]`, rawTx)),
		ID:      json.RawMessage("0"),
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ShardID returns the shard id for the given number of shards.
func (tx *LoadTx) ShardID(n int) int {
	addressBigInt := new(big.Int).SetBytes(tx.Scenario.Sender.Address.Bytes())
	mod := new(big.Int).Mod(addressBigInt, big.NewInt(int64(n)))
	return int(mod.Int64())
}

// TxScenario captures the scenario of this test transaction.
type TxScenario struct {
	Name     string
	Sender   *Account
	Receiver common.Address
}

// CreateTxFromEthTx creates a LoadTx from an EthTx (pre-marshaled).
func CreateTxFromEthTx(tx *ethtypes.Transaction, scenario *TxScenario) *LoadTx {
	// Convert to raw transaction bytes for JSON-RPC payload
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		panic("Failed to marshal transaction: " + err.Error())
	}

	// Create JSON-RPC payload
	jsonRPCPayload, err := toJSONRequestBytes(rawTx)
	if err != nil {
		panic("Failed to create JSON-RPC payload: " + err.Error())
	}

	// Return the complete LoadTx object
	return &LoadTx{
		EthTx:          tx,
		JSONRPCPayload: jsonRPCPayload,
		Payload:        rawTx,
		Scenario:       scenario,
	}
}
