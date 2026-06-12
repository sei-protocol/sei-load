package sender

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/service"
)

// This file is the production-path safety net for the open-loop in-flight bound.
//
// Every other scheduler test drives a FAKE TxSender that invokes tx.OnComplete
// itself, so the suite would stay green even if the real Worker forgot the
// `if tx.OnComplete != nil { tx.OnComplete(err) }` line in runTxSender — the one
// load-bearing line that makes the maxInFlight semaphore bound true unacked
// sends rather than nothing (permits would never be released → leak/meaningless
// bound). The tests here wire the REAL Worker (runTxSender → sendTransaction →
// the real ethclient → OnComplete) behind the scheduler and assert the permit
// is genuinely released by the worker on send completion.
//
// Harness: an httptest.Server speaking the minimal JSON-RPC the ethclient send
// path touches. SendTransaction issues exactly one eth_sendRawTransaction call
// per tx (verified against go-ethereum v1.16.1: HTTP dial makes no RPC call, and
// SendTransaction marshals the tx and calls eth_sendRawTransaction; no
// eth_chainId round-trip). This keeps the harness loopback-only and lets us
// exercise the real worker send path end to end with a controllable response —
// including a "block until released" mode for the maxInFlight=1 assertion.

// jsonRPCReq is the subset of a JSON-RPC request we parse from the ethclient.
type jsonRPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
}

// rpcServer is an httptest-backed JSON-RPC endpoint that answers
// eth_sendRawTransaction. It counts handled sends and can be put into a
// "block" mode where each send parks until explicitly released, so a test can
// hold a send in flight and observe the in-flight bound.
type rpcServer struct {
	srv *httptest.Server

	entered atomic.Uint64 // eth_sendRawTransaction calls that entered the handler
	handled atomic.Uint64 // eth_sendRawTransaction calls that returned a result

	// When blocking, every send waits on a fresh gate handed out via started so
	// the test can release them one at a time. arrived is signaled when a send
	// has entered the handler (so the test knows a send is genuinely in flight).
	mu       sync.Mutex
	blocking bool
	gates    []chan struct{} // one per blocked send, in arrival order
	arrived  chan struct{}   // buffered; one token per send that entered the handler
}

func newRPCServer(t *testing.T) *rpcServer {
	t.Helper()
	s := &rpcServer{arrived: make(chan struct{}, 1024)}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *rpcServer) url() string { return s.srv.URL }

// setBlocking toggles the block-until-released mode.
func (s *rpcServer) setBlocking(b bool) {
	s.mu.Lock()
	s.blocking = b
	s.mu.Unlock()
}

// releaseOne unblocks the oldest parked send. Returns false if none is parked
// yet (caller should retry after observing an arrival).
func (s *rpcServer) releaseOne() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.gates) == 0 {
		return false
	}
	gate := s.gates[0]
	s.gates = s.gates[1:]
	close(gate)
	return true
}

func (s *rpcServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req jsonRPCReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Method == "eth_sendRawTransaction" {
		s.entered.Add(1)
		s.mu.Lock()
		blocking := s.blocking
		var gate chan struct{}
		if blocking {
			gate = make(chan struct{})
			s.gates = append(s.gates, gate)
		}
		s.mu.Unlock()

		if blocking {
			// Announce arrival, then park until released or the request ctx is
			// canceled (campaign teardown). Parking here holds the real worker
			// inside sendTransaction, so the scheduler's permit stays held.
			s.arrived <- struct{}{}
			select {
			case <-gate:
			case <-r.Context().Done():
			}
		}
	}

	id := req.ID
	if len(id) == 0 {
		id = json.RawMessage("0")
	}
	// A non-error result is enough; ethclient discards the value (nil out arg).
	resp := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  string          `json:"result"`
	}{
		JSONRPC: "2.0",
		ID:      id,
		Result:  "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	if req.Method == "eth_sendRawTransaction" {
		s.handled.Add(1)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&resp)
}

// signedTxGenerator yields real, signed DynamicFee transactions (the production
// EVMTransfer shape) so the real ethclient marshals and ships a valid raw tx to
// the JSON-RPC server. It is the generator.Generator the scheduler drives.
type signedTxGenerator struct {
	mu        sync.Mutex
	remaining int
	acct      *types.Account
	signer    ethtypes.Signer
	chainID   *big.Int
	issued    int
}

func newSignedTxGenerator(t *testing.T, n int) *signedTxGenerator {
	t.Helper()
	acct, err := types.NewAccount()
	require.NoError(t, err)
	chainID := big.NewInt(1)
	return &signedTxGenerator{
		remaining: n,
		acct:      acct,
		signer:    ethtypes.NewCancunSigner(chainID),
		chainID:   chainID,
	}
}

func (g *signedTxGenerator) Generate() (*types.LoadTx, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.remaining == 0 {
		return nil, false
	}
	g.remaining--
	g.issued++

	to := g.acct.Address
	inner := &ethtypes.DynamicFeeTx{
		ChainID:   g.chainID,
		Nonce:     g.acct.GetAndIncrementNonce(),
		To:        &to,
		Value:     big.NewInt(1),
		Gas:       21000,
		GasTipCap: big.NewInt(2_000_000_000),
		GasFeeCap: big.NewInt(200_000_000_000),
	}
	signed, err := ethtypes.SignTx(ethtypes.NewTx(inner), g.signer, g.acct.PrivKey)
	if err != nil {
		// Generators have no error channel; a signing failure here is a test bug.
		panic(err)
	}
	scenario := &types.TxScenario{Name: "realworker", Sender: g.acct, Receiver: to}
	return types.CreateTxFromEthTx(signed, scenario), true
}

func (g *signedTxGenerator) GenerateN(int) []*types.LoadTx { panic("unused") }
func (g *signedTxGenerator) GetAccountPools() []types.AccountPool { return nil }

func (g *signedTxGenerator) issuedCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.issued
}

// newRealWorker builds the production Worker against the given endpoint, in the
// open-loop configuration (RateLimited=false so the scheduler owns the clock,
// TrackReceipts=false so watchTransactions returns immediately and we exercise
// only the send path). It is the real TxSender the scheduler drives.
func newRealWorker(endpoint string, tasks, buffer int) *Worker {
	return NewWorker(&WorkerConfig{
		ID:          0,
		SeiChainID:  "test",
		Endpoint:    endpoint,
		BufferSize:  buffer,
		Tasks:       tasks,
		DryRun:      false,
		Debug:       false,
		Collector:   stats.NewCollector(),
		RateLimited: false,
	})
}

// TestRealWorker_Conservation_OnRealSendPath asserts conservation
// (issued == completed + dropped) where `completed` is driven exclusively by the
// REAL worker invoking tx.OnComplete after sendTransaction returns — not by a
// fake. If runTxSender stopped calling OnComplete, completed would stall and
// this would fail.
func TestRealWorker_Conservation_OnRealSendPath(t *testing.T) {
	const txCount = 200
	srv := newRPCServer(t)
	gen := newSignedTxGenerator(t, txCount)
	worker := newRealWorker(srv.url(), 8, 256)

	var completed, succeeded atomic.Uint64
	onSent := func(_ *types.LoadTx, err error) {
		completed.Add(1)
		if err == nil {
			succeeded.Add(1)
		}
	}

	limiter := rate.NewLimiter(rate.Limit(2000), 1)
	sched := newOpenLoopScheduler(gen, worker, limiter, 256, onSent)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// Run the real worker (background) and the scheduler (main) in the same
	// scope. The worker must keep draining txChan after the scheduler exhausts
	// the generator, so we DON'T let the scheduler's return tear the scope down:
	// we keep the main task alive until every admitted tx has completed, then
	// cancel. Otherwise an admitted tx still buffered in txChan at cancel time
	// would never fire OnComplete and conservation would (correctly) fail.
	runCtx, runCancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.Run(runCtx, func(ctx context.Context, scope service.Scope) error {
			scope.SpawnBg(func() error { return worker.Run(ctx) })
			return sched.Run(ctx, scope)
		})
	}()

	issued := func() uint64 { return uint64(gen.issuedCount()) }

	// Every issued tx must be completed (by the real worker's OnComplete) or
	// dropped — exactly once each, no leaks, no double-count. Driven by the real
	// worker's OnComplete, so a missing invoke leaves completions short forever.
	require.Eventually(t, func() bool {
		i := issued()
		return i > 0 && completed.Load()+sched.Dropped() == i
	}, 3*time.Second, 2*time.Millisecond,
		"issued=%d completed=%d dropped=%d", issued(), completed.Load(), sched.Dropped())

	runCancel()
	wg.Wait()

	require.Positive(t, issued())

	// The completions came from the REAL send path, not a phantom OnComplete:
	// every send the server successfully handled produced exactly one successful
	// worker-driven completion. (A send that errored client-side — e.g. ctx
	// canceled at teardown — still completes but is not server-handled, so we
	// match handled against the success count, not the total.)
	require.Positive(t, srv.handled.Load(), "the real RPC server must have handled sends")
	require.Equal(t, succeeded.Load(), srv.handled.Load(),
		"each successful completion must correspond to one eth_sendRawTransaction")
}

// TestRealWorker_PermitReleasedByWorker is the teeth: with maxInFlight=1 and a
// single worker task, the RPC server blocks the first send. The real worker is
// parked inside sendTransaction, so it has NOT yet called tx.OnComplete and the
// single permit stays held — every subsequent arrival must drop. Releasing the
// blocked send lets the worker return from sendTransaction, fire OnComplete, and
// free the permit, so flow resumes.
//
// If someone deletes the `if tx.OnComplete != nil { tx.OnComplete(err) }` invoke
// in runTxSender, the permit is never released even after the send completes:
// the worker would never accept a second tx, so handled stays at 1 and the
// resume assertion fails. That is the falsification this test exists for.
func TestRealWorker_PermitReleasedByWorker(t *testing.T) {
	srv := newRPCServer(t)
	srv.setBlocking(true)

	// Plenty of arrivals so the scheduler keeps offering txs while the first is
	// parked; the surplus must drop because the lone permit is held.
	gen := newSignedTxGenerator(t, 1000)
	// One task: a single runTxSender owns the only permit's lifecycle, so the
	// permit can only be freed by that worker calling OnComplete.
	worker := newRealWorker(srv.url(), 1, 1)

	var completed atomic.Uint64
	onSent := func(_ *types.LoadTx, _ error) { completed.Add(1) }

	// Fast arrival clock so many txs are offered during the blocked window.
	limiter := rate.NewLimiter(rate.Limit(5000), 1) // 0.2ms gap
	sched := newOpenLoopScheduler(gen, worker, limiter, 1, onSent)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.Run(ctx, func(ctx context.Context, scope service.Scope) error {
			scope.SpawnBg(func() error { return worker.Run(ctx) })
			return sched.Run(ctx, scope)
		})
	}()

	// Wait until exactly one send is genuinely in flight (parked in the handler).
	<-srv.arrived

	// While that send is parked, the worker has not fired OnComplete, so the lone
	// permit is held. Give the fast scheduler time to offer (and drop) a slew of
	// arrivals, then assert the bound held: exactly one send in flight, none
	// completed yet, and the rest dropped.
	require.Eventually(t, func() bool {
		return sched.Dropped() > 10
	}, 2*time.Second, 2*time.Millisecond,
		"arrivals past the single held permit must drop while the send is in flight")

	require.Equal(t, uint64(0), completed.Load(),
		"no completion may be reported while the only send is still parked")
	require.Equal(t, uint64(1), srv.entered.Load(),
		"exactly one send may be in flight under maxInFlight=1")
	require.Equal(t, uint64(0), srv.handled.Load(),
		"the parked send has not returned a result yet, so the permit is still held")

	// Release the blocked send. The real worker now returns from sendTransaction
	// and MUST invoke tx.OnComplete to free the permit. If it does not (the bug),
	// no further send is ever admitted and handled stays at 1 forever.
	require.True(t, srv.releaseOne(), "one send must be parked and releasable")

	// Switch off blocking so resumed sends complete immediately, and prove flow
	// resumes: more than one send is now handled, which is only possible if the
	// permit from the first send was released by the worker's OnComplete.
	srv.setBlocking(false)
	// Drain any further parked sends that arrived between release and unblocking.
	go func() {
		for ctx.Err() == nil {
			if !srv.releaseOne() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond):
				}
			}
		}
	}()

	require.Eventually(t, func() bool {
		return srv.handled.Load() > 1 && completed.Load() > 1
	}, 3*time.Second, 2*time.Millisecond,
		"flow must resume after the worker releases the permit via OnComplete "+
			"(handled=%d completed=%d)", srv.handled.Load(), completed.Load())

	// Flow resumed: at least one further send completed after the release, so the
	// worker really did free the permit. Record the post-resume state before tear
	// down (strict end-to-end conservation is covered by the conservation test;
	// here cancel may leave a single tx mid-flight, so we only bound leaks).
	require.Greater(t, completed.Load(), uint64(1), "resumed sends must complete")

	cancel()
	wg.Wait()

	// No leak past the one tx that may be mid-flight at cancel: accounted txs
	// (completed + dropped) must never exceed issued, and must trail it by at
	// most that single in-flight tx.
	accounted := completed.Load() + sched.Dropped()
	issued := uint64(gen.issuedCount())
	require.LessOrEqual(t, accounted, issued, "no tx may be counted more than once")
	require.GreaterOrEqual(t, accounted+1, issued,
		"at most one admitted tx may be unaccounted at cancel (issued=%d completed=%d dropped=%d)",
		issued, completed.Load(), sched.Dropped())
}
