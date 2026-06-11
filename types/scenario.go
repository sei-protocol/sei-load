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
// Lifecycle timestamp concurrency contract:
// A *LoadTx is passed by pointer through buffered channels (txChan, sentTxs) and
// read by multiple worker goroutines. The lifecycle timestamps below are written
// at most once, each by a single owner, and are immutable thereafter. Workers must
// not mutate these timestamps after a tx is enqueued. This single-writer-before-
// hand-off discipline is what keeps LoadTx race-free without locks.
//   - IntendedSendTs: written exactly once by the dispatcher before the tx is
//     enqueued into the send pipeline; immutable after enqueue.
//   - AttemptedSendTs: reserved for PLT-458; will be written once by the sender at
//     the actual send attempt.
//   - InclusionTs: reserved for PLT-459; will be written exactly once by the
//     inclusion tracker (its sole owner), never by workers.
type LoadTx struct {
	EthTx          *ethtypes.Transaction
	JSONRPCPayload []byte
	Payload        []byte
	Scenario       *TxScenario

	// IntendedSendTs is when the tx was scheduled to be sent. In today's
	// closed-loop dispatcher it is stamped at enqueue into the send pipeline.
	// PLT-458 will move this to the true scheduled instant t₀ + i/λ.
	IntendedSendTs time.Time
	// AttemptedSendTs is when the send was actually attempted. Reserved for
	// PLT-458 (open-loop scheduler); not populated in this PR.
	AttemptedSendTs time.Time
	// InclusionTs is when the tx was observed included on-chain. Reserved for
	// PLT-459 (inclusion tracker); not populated in this PR.
	InclusionTs time.Time
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
