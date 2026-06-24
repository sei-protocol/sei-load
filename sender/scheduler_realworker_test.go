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
// itself, so the suite would stay green even if the real ethClient forgot the
// `if tx.OnComplete != nil { tx.OnComplete(err) }` line in runSender — the one
// load-bearing line that makes the maxInFlight semaphore bound true unacked
// sends rather than nothing (permits would never be released → leak/meaningless
// bound). The tests here wire the REAL ethClient (runSender → sendTx →
// the real ethclient → OnComplete) behind the scheduler and assert the permit
// is genuinely released by the sender on send completion.
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
	acct := types.NewAccount()
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

func (g *signedTxGenerator) GetAccountPools() []*types.AccountPool { return nil }

func (g *signedTxGenerator) issuedCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.issued
}

// newRealSender builds the production ethClient against the given endpoint. It
// is the real
// TxSender the scheduler drives.
func newRealSender(endpoint string, tasks int) *ethClient {
	return newEthClient(&ethClientConfig{
		ChainID:   "test",
		ID:        0,
		Endpoint:  endpoint,
		Tasks:     tasks,
		DryRun:    false,
		Debug:     false,
		Collector: stats.NewCollector(),
	})
}

// TestRealSender_Conservation_OnRealSendPath asserts conservation
// (issued == completed + dropped) where `completed` is driven exclusively by the
// REAL sender invoking tx.OnComplete after sendTx returns — not by a fake. If
// runSender stopped calling OnComplete, completed would stall and
// this would fail.
func TestRealSender_Conservation_OnRealSendPath(t *testing.T) {
	const txCount = 200
	srv := newRPCServer(t)
	gen := newSignedTxGenerator(t, txCount)
	client := newRealSender(srv.url(), 8)

	var completed, succeeded atomic.Uint64
	onSent := func(_ *types.LoadTx, err error) {
		completed.Add(1)
		if err == nil {
			succeeded.Add(1)
		}
	}

	limiter := rate.NewLimiter(rate.Limit(2000), 1)
	sched := newOpenLoopScheduler(gen, client, limiter, 256, onSent)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Run the sender and scheduler in a scope whose teardown WE control via
	// runCancel — not the scheduler's return.
	//
	// service.Run cancels the scope's context as soon as every MAIN task returns.
	// If the scheduler were a main task, the instant it exhausts the generator and
	// returns, service.Run would cancel the sender's context — aborting any send
	// still in flight. A send whose 200 OK the server already counted (handled++)
	// but whose client.SendTransaction had not yet returned would then fail with
	// context-canceled: OnComplete fires with err != nil, so completed++ but NOT
	// succeeded++. That is exactly the observed flake (handled=200, succeeded=199):
	// not a sampling artifact but a teardown that races the last in-flight send.
	//
	// So the scheduler and sender are BACKGROUND tasks, and the lone MAIN task is a
	// gate that blocks until the test calls runCancel(). The scope therefore stays
	// alive — the sender keeps draining reqs and firing OnComplete — until the
	// test has observed quiescence and torn down deliberately.
	runCtx, runCancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.Run(runCtx, func(ctx context.Context, scope service.Scope) error {
			scope.SpawnBg(func() error { return client.Run(ctx) })
			scope.SpawnBg(func() error { return sched.Run(ctx, scope) })
			// Main task: hold the scope open until the test signals teardown.
			<-ctx.Done()
			return nil
		})
	}()

	// Assert ONLY at quiescence. All invariants are sampled together in one
	// predicate so we never read them mid-flight. Post-reorder, the generator is
	// drawn only on admitted ticks, so the conservation anchor is the scheduler's
	// OWN counters, not the generator draw count:
	//
	//   admitted:     Admitted() == txCount        (every draw was admitted — the
	//                 generator is drained and dropped ticks consumed no draw;
	//                 the precondition that makes the fixpoint below stable)
	//   conservation: completed == Admitted()       (every admitted tx reached a
	//                 terminal state via the real sender's OnComplete)
	//   equality:     succeeded == handled          (every server-handled send
	//                 produced exactly one successful worker-driven completion)
	//
	// conservation and equality are transiently off WHILE a send is in flight: the
	// server bumps `handled` when it RECEIVES eth_sendRawTransaction, but the sender
	// bumps `succeeded` only AFTER SendTransaction returns and OnComplete fires — the
	// instants differ by the server→worker-return window. Sampling any of them alone,
	// or at different instants, can catch that window. Requiring all three together,
	// only once they hold, observes the system after that window has drained. The
	// counters are monotonic; once the generator is exhausted no new work is admitted,
	// so once ALL THREE hold they stay held — that stable fixpoint is the quiescent
	// point. (The deeper hazard the gate above fixes is teardown racing that same
	// window; here we additionally refuse to read until the window is empty.)
	//
	// Driven by the real sender's OnComplete — a missing invoke leaves completed
	// (and succeeded) short forever, so convergence never happens and the test
	// fails on the Eventually deadline. CI is slow, so the window is generous;
	// correctness depends on convergence, not on the deadline firing.
	const total = uint64(txCount)
	require.Eventually(t, func() bool {
		admittedAll := sched.Admitted() == total
		conserved := completed.Load() == sched.Admitted()
		balanced := succeeded.Load() == srv.handled.Load()
		return admittedAll && conserved && balanced
	}, 10*time.Second, 2*time.Millisecond,
		"never reached quiescence (want admitted=completed=%d, succeeded=handled)", total)

	// System is quiescent: the generator is drained, no send is in flight, every
	// admitted tx is terminal, and every handled send has its OnComplete recorded.
	// Only now tear down — the counters cannot move under us, so the assertions
	// below re-read a frozen state, not a sampled one.
	runCancel()
	wg.Wait()

	require.Equal(t, total, sched.Admitted(), "every generator draw must be an admitted tx")
	require.Equal(t, total, uint64(gen.issuedCount()), "the generator must be fully drained")
	require.Positive(t, srv.handled.Load(), "the real RPC server must have handled sends")
	require.Equal(t, sched.Admitted(), completed.Load(),
		"every admitted tx must reach a terminal state via the sender's OnComplete")
	require.Equal(t, succeeded.Load(), srv.handled.Load(),
		"each successful completion must correspond to one eth_sendRawTransaction")
}

// TestRealSender_PermitReleasedBySender is the teeth: with maxInFlight=1 and a
// single sender task, the RPC server blocks the first send. The real sender is
// parked inside sendTx, so it has NOT yet called tx.OnComplete and the
// single permit stays held — every subsequent arrival must drop. Releasing the
// blocked send lets the sender return from sendTx, fire OnComplete, and
// free the permit, so flow resumes.
//
// If someone deletes the `if tx.OnComplete != nil { tx.OnComplete(err) }` invoke
// in runSender, the permit is never released even after the send completes: the
// sender would never accept a second tx, so handled stays at 1 and the
// resume assertion fails. That is the falsification this test exists for.
func TestRealSender_PermitReleasedBySender(t *testing.T) {
	srv := newRPCServer(t)
	srv.setBlocking(true)

	// Plenty of arrivals so the scheduler keeps offering txs while the first is
	// parked; the surplus must drop because the lone permit is held.
	gen := newSignedTxGenerator(t, 1000)
	// One task: a single runSender owns the only permit's lifecycle, so the
	// permit can only be freed by that sender calling OnComplete.
	client := newRealSender(srv.url(), 1)

	var completed atomic.Uint64
	onSent := func(_ *types.LoadTx, _ error) { completed.Add(1) }

	// Fast arrival clock so many txs are offered during the blocked window.
	limiter := rate.NewLimiter(rate.Limit(5000), 1) // 0.2ms gap
	sched := newOpenLoopScheduler(gen, client, limiter, 1, onSent)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.Run(ctx, func(ctx context.Context, scope service.Scope) error {
			scope.SpawnBg(func() error { return client.Run(ctx) })
			return sched.Run(ctx, scope)
		})
	}()

	// Wait until exactly one send is genuinely in flight (parked in the handler).
	<-srv.arrived

	// While that send is parked, the sender has not fired OnComplete, so the lone
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

	// Release the blocked send. The real sender now returns from sendTx
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
		"flow must resume after the sender releases the permit via OnComplete "+
			"(handled=%d completed=%d)", srv.handled.Load(), completed.Load())

	// Flow resumed: at least one further send completed after the release, so the
	// worker really did free the permit. Record the post-resume state before tear
	// down (strict end-to-end conservation is covered by the conservation test;
	// here cancel may leave a single tx mid-flight, so we only bound leaks).
	require.Greater(t, completed.Load(), uint64(1), "resumed sends must complete")

	cancel()
	wg.Wait()

	// No leak past the one tx that may be mid-flight at cancel. Post-reorder, the
	// generator is drawn only on admitted ticks, so every draw is an admitted tx:
	// generator-draw count must equal Admitted() exactly (dropped ticks consumed
	// no draw — the determinism property, on the real worker path).
	admitted := sched.Admitted()
	require.Equal(t, admitted, uint64(gen.issuedCount()),
		"every generator draw must be an admitted tx; dropped ticks consume no draw")

	// Each admitted tx completes exactly once via the sender's OnComplete, so
	// completed must equal Admitted() — minus at most the single tx left mid-flight
	// when cancel raced its in-flight send.
	require.LessOrEqual(t, completed.Load(), admitted, "no admitted tx may complete more than once")
	require.GreaterOrEqual(t, completed.Load()+1, admitted,
		"at most one admitted tx may be unaccounted at cancel (admitted=%d completed=%d dropped=%d)",
		admitted, completed.Load(), sched.Dropped())
}

// TestDispatcher_PrewarmRateLimitedInOpenLoop guards the prewarm-flood
// regression: in open-loop the sender loop is ungated, but
// the scheduler paces only the MAIN load. Prewarm runs first over those same
// ungated senders, so it must pace itself off the shared limiter or it floods
// the SUT. With workers wired exactly as in open-loop, a low limit, and many
// more prewarm txs than the worker pool could absorb instantly, an unpaced
// prewarm would drain in well under the limiter's minimum span. We assert the
// run took at least the paced floor — i.e. the limiter actually gated prewarm —
// and that every prewarm tx still reached the RPC server (no drops on prewarm).
func TestDispatcher_PrewarmRateLimitedInOpenLoop(t *testing.T) {
	srv := newRPCServer(t)
	const prewarmTxs = 40
	const rps = 200.0 // limiter: 200 tx/s → unpaced 40 txs is near-instant

	client := newRealSender(srv.url(), 8)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.Run(ctx, func(ctx context.Context, scope service.Scope) error {
			scope.SpawnBg(func() error { return client.Run(ctx) })
			<-ctx.Done()
			return nil
		})
	}()

	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	d := NewDispatcher(newSignedTxGenerator(t, 0), client)
	d.SetOpenLoop(limiter, 256) // sets d.limiter so Prewarm self-paces
	d.SetPrewarmGenerator(newSignedTxGenerator(t, prewarmTxs))

	start := time.Now()
	require.NoError(t, d.Prewarm(ctx))
	elapsed := time.Since(start)

	// Paced floor: (N-1) gaps at the limiter rate (burst=1 lets the first through
	// immediately). Use half as a generous lower bound to absorb scheduling slop
	// while still excluding the unpaced (near-zero) case decisively.
	pacedFloor := time.Duration(float64(prewarmTxs-1) / rps * float64(time.Second))
	require.Greater(t, elapsed, pacedFloor/2,
		"prewarm must be limiter-paced in open-loop, not flooded (elapsed=%s floor=%s)",
		elapsed, pacedFloor)

	require.Eventually(t, func() bool {
		return srv.handled.Load() == uint64(prewarmTxs)
	}, 2*time.Second, 2*time.Millisecond,
		"every prewarm tx must reach the SUT (handled=%d want=%d)",
		srv.handled.Load(), prewarmTxs)

	cancel()
	wg.Wait()
}
