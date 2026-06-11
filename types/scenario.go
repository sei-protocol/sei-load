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
// Lifecycle timestamp concurrency contract: a *LoadTx is passed by pointer
// through buffered channels (txChan, sentTxs). Each lifecycle timestamp is
// written at most once, by whichever goroutine owns the tx at that stage, and
// is immutable thereafter; ownership transfers with the pointer across the
// channels, so the writes need no locking. A zero timestamp means "not
// recorded" (e.g. prewarm txs, or a stage not yet reached) — consumers must
// treat it as untracked, never as the zero epoch.
type LoadTx struct {
	EthTx          *ethtypes.Transaction
	JSONRPCPayload []byte
	Payload        []byte
	Scenario       *TxScenario

	// IntendedSendTime is when the tx was scheduled to be sent, written by the
	// dispatcher before the tx is enqueued. It currently holds the enqueue time,
	// which is back-pressured under load; until an open-loop scheduler sets it to
	// the intended schedule instant, it must not be used to derive latency.
	IntendedSendTime time.Time
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
