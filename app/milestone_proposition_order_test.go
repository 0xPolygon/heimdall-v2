package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	topUpTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

// borSim is a minimal Bor chain: every block records the address that sealed it, and
// headers are hash-linked so Heimdall's parent-hash checks are exercised for real.
type borSim struct {
	*helpermocks.IContractCaller
	headers []*ethTypes.Header
	authors []common.Address
}

func (b *borSim) head() uint64 { return uint64(len(b.headers) - 1) }

func (b *borSim) produce(author common.Address) {
	n := uint64(len(b.headers))
	h := &ethTypes.Header{Number: new(big.Int).SetUint64(n), Difficulty: big.NewInt(1), Extra: append(author.Bytes(), byte(n), byte(n>>8))}
	if n > 0 {
		h.ParentHash = b.headers[n-1].Hash()
	}
	b.headers, b.authors = append(b.headers, h), append(b.authors, author)
}

func (b *borSim) GetBorChainBlock(_ context.Context, n *big.Int) (*ethTypes.Header, error) {
	if n == nil {
		return b.headers[b.head()], nil
	}
	return b.headers[n.Uint64()], nil
}

func (b *borSim) GetBorChainBlockAuthor(_ context.Context, n *big.Int) (*common.Address, error) {
	a := b.authors[n.Uint64()]
	return &a, nil
}

func (b *borSim) GetBorChainBlockInfoInBatch(_ context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	var (
		hs  []*ethTypes.Header
		tds []uint64
		as  []common.Address
	)
	for n := uint64(start); n <= uint64(end) && n <= b.head(); n++ {
		hs, tds, as = append(hs, b.headers[n]), append(tds, n), append(as, b.authors[n])
	}
	return hs, tds, as, nil
}

// fakeL1 is a local stand-in for one validator's Ethereum RPC endpoint. Every request
// costs `rtt`. Receipts live in L1 block 100; the finalized head is block 90, i.e. the
// side tx references a real but not-yet-finalized L1 transaction.
func fakeL1(t *testing.T, rtt time.Duration) *httptest.Server {
	return fakeL1WithBatchDelay(t, rtt, 0)
}

// fakeL1WithBatchDelay is fakeL1 where a batch request (the receipt prefetch) waits
// batchDelay more than a single request.
func fakeL1WithBatchDelay(t *testing.T, rtt, batchDelay time.Duration) *httptest.Server {
	zero32 := "0x" + strings.Repeat("0", 64)
	bloom := "0x" + strings.Repeat("0", 512)
	header := fmt.Sprintf(`{"parentHash":%q,"sha3Uncles":%q,"miner":"0x0000000000000000000000000000000000000000","stateRoot":%q,"transactionsRoot":%q,"receiptsRoot":%q,"logsBloom":%q,"difficulty":"0x0","number":"0x5a","gasLimit":"0x1c9c380","gasUsed":"0x0","timestamp":"0x1","extraData":"0x","mixHash":%q,"nonce":"0x0000000000000000","baseFeePerGas":"0x1","hash":%q}`,
		zero32, zero32, zero32, zero32, zero32, bloom, zero32, zero32)
	receipt := func(i int) string {
		return fmt.Sprintf(`{"blockNumber":"0x64","status":"0x1","cumulativeGasUsed":"0x5208","gasUsed":"0x5208","logsBloom":%q,"logs":[],"transactionHash":"0x%064x"}`, bloom, i+1)
	}
	type req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	answer := func(q req, i int) string {
		if q.Method == "eth_getBlockByNumber" {
			return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, header)
		}
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, receipt(i))
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(rtt)
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
			time.Sleep(batchDelay)
			var qs []req
			_ = json.Unmarshal(body, &qs)
			parts := make([]string, len(qs))
			for i, q := range qs {
				parts[i] = answer(q, i)
			}
			fmt.Fprint(w, "["+strings.Join(parts, ",")+"]")
			return
		}
		var q req
		_ = json.Unmarshal(body, &q)
		fmt.Fprint(w, answer(q, 0))
	}))
}

// fakeBorRPC serves the Bor JSON-RPC calls Heimdall makes, from a live borSim. Each request
// waits delay first; 0 means a healthy, local Bor node.
func fakeBorRPC(t *testing.T, bor *borSim, mu *sync.Mutex, delay time.Duration) *httptest.Server {
	type req struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	num := func(p []json.RawMessage) uint64 {
		if len(p) == 0 {
			return bor.head()
		}
		var s string
		_ = json.Unmarshal(p[0], &s)
		if s == "latest" || s == "" {
			return bor.head()
		}
		n, _ := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
		return n
	}
	answer := func(q req) string {
		mu.Lock()
		defer mu.Unlock()
		n := num(q.Params)
		var result string
		switch {
		case n > bor.head():
			result = "null"
		case q.Method == "eth_getBlockByNumber" || q.Method == "eth_getHeaderByNumber":
			bz, _ := json.Marshal(bor.headers[n])
			result = string(bz)
		case q.Method == "eth_getTdByNumber":
			result = fmt.Sprintf(`{"totalDifficulty":"0x%x"}`, n)
		case q.Method == "bor_getAuthor":
			result = fmt.Sprintf("%q", bor.authors[n].Hex())
		default:
			return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"method not found"}}`, q.ID)
		}
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, result)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
			var qs []req
			_ = json.Unmarshal(body, &qs)
			parts := make([]string, len(qs))
			for i, q := range qs {
				parts[i] = answer(q)
			}
			fmt.Fprint(w, "["+strings.Join(parts, ",")+"]")
			return
		}
		var q req
		_ = json.Unmarshal(body, &q)
		fmt.Fprint(w, answer(q))
	}))
}

// TestExtendVote_ProposesWithManySideTxs runs the real ExtendVote for every validator at every height.
// 7 of 10 equal-power validators reach Ethereum through a hosted endpoint (20 ms round
// trip), 3 run a co-located node (2 ms). Bor is healthy and local for all of them.
// Every block carries 49 ordinary MsgTopupTx side txs.
func TestExtendVote_ProposesWithManySideTxs(t *testing.T) {
	const M = uint64(100)
	helper.SetPhuketHardforkHeight(1)
	helper.SetZurichHardforkHeight(1)
	origIthaca, origRio := helper.GetIthacaHeight(), helper.GetRioHeight()
	t.Cleanup(func() {
		helper.SetPhuketHardforkHeight(0)
		helper.SetZurichHardforkHeight(0)
		helper.SetIthacaHeight(origIthaca)
		helper.SetRioHeight(origRio)
	})
	helper.SetIthacaHeight(1)
	helper.SetRioHeight(int64(M + 1))

	submitter, app, ctx, valKeys := SetupAppWithABCICtxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1}})

	votes := []uint64{validators[0].ValId, validators[1].ValId, validators[2].ValId, validators[3].ValId}
	for _, v := range validators {
		require.NoError(t, app.BorKeeper.SetProducerVotes(ctx, v.ValId, borTypes.ProducerVotes{Votes: votes}))
	}
	params, err := app.BorKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ProducerCount, params.SpanDuration = 4, 100
	require.NoError(t, app.BorKeeper.SetParams(ctx, params))
	active := map[uint64]struct{}{}
	for _, v := range validators {
		active[v.ValId] = struct{}{}
	}
	require.NoError(t, app.BorKeeper.UpdateLatestActiveProducer(ctx, active))

	p1 := common.HexToAddress(validators[0].Signer)
	var mu sync.Mutex
	bor := &borSim{IContractCaller: new(helpermocks.IContractCaller)}
	for n := uint64(0); n <= M; n++ {
		bor.produce(p1)
	}
	borSrv := fakeBorRPC(t, bor, &mu, 0)
	defer borSrv.Close()

	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{Id: 1, StartBlock: 1, EndBlock: 300, BorChainId: "test",
		ValidatorSet: stakeTypes.ValidatorSet{Validators: validators, Proposer: validators[0]}, SelectedProducers: []stakeTypes.Validator{*validators[0]}}))
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, milestoneTypes.Milestone{MilestoneId: "1", StartBlock: 0, EndBlock: M, Hash: bor.headers[M].Hash().Bytes(), BorChainId: "test"}))
	require.NoError(t, app.MilestoneKeeper.SetLastMilestoneBlock(ctx, M))

	// Each validator's own environment: a real helper.ContractCaller pointed at its L1 and Bor endpoints.
	rtts := []time.Duration{2, 2, 2, 20, 20, 20, 20, 20, 20, 20}
	callers := make([]*helper.ContractCaller, len(validators))
	for i := range validators {
		l1 := fakeL1(t, rtts[i]*time.Millisecond)
		t.Cleanup(l1.Close)
		cc, err := helper.NewContractCaller()
		require.NoError(t, err)
		l1rc, err := rpc.Dial(l1.URL)
		require.NoError(t, err)
		borrc, err := rpc.Dial(borSrv.URL)
		require.NoError(t, err)
		cc.MainChainRPCClient, cc.MainChainClient, cc.MainChainTimeout = l1rc, ethclient.NewClient(l1rc), 5*time.Second
		cc.BorChainRPCClient, cc.BorChainClient, cc.BorChainTimeout = borrc, ethclient.NewClient(borrc), 5*time.Second
		callers[i] = cc
	}
	use := func(i int) {
		app.caller = callers[i]
		app.TopupKeeper.SetContractCaller(callers[i])
		app.BorKeeper.SetContractCaller(callers[i])
		app.MilestoneKeeper.IContractCaller = callers[i]
	}

	// Submitter txs: 49 topups naming a real but not yet finalized L1 transaction.
	me := submitter.PubKey().Address().String()
	seq := app.AccountKeeper.GetAccount(ctx, submitter.PubKey().Address().Bytes()).GetSequence()
	l1Tx := common.HexToHash("0x6a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9")
	var padding [][]byte
	for i := 0; i < 49; i++ {
		bz, err := buildSignedTxWithSequence(&topUpTypes.MsgTopupTx{Proposer: me, User: me, Fee: ante.DefaultFeeWantedPerTx[0].Amount.Add(math.NewInt(1)),
			TxHash: l1Tx.Bytes(), LogIndex: uint64(i), BlockNumber: 100}, ctx, submitter, app, seq+uint64(i))
		require.NoError(t, err)
		padding = append(padding, bz)
	}
	blockHash := common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000008dead")
	prevCommit, _, _, err := buildExtensionCommits(t, app, blockHash, validators, valKeys, 2, nil)
	require.NoError(t, err)

	// step runs one Heimdall height: Bor seals a block, every validator runs the real ExtendVote
	// on the proposed block, and the resulting vote extensions feed the next PreBlocker.
	step := func(e int64, blockTxs [][]byte) (withProp int) {
		mu.Lock()
		if producers, err := app.BorKeeper.GetProducersByBlockNumber(ctx.WithBlockHeight(e), bor.head()+1); err == nil && len(producers) > 0 {
			bor.produce(producers[0])
		}
		mu.Unlock()

		ec := &abci.ExtendedCommitInfo{}
		for i, v := range validators {
			use(i)
			res, err := app.ExtendVoteHandler()(ctx.WithBlockHeight(e), &abci.RequestExtendVote{Txs: append([][]byte{prevCommit}, blockTxs...), Height: e, Hash: blockHash})
			require.NoError(t, err)
			var ve sidetxs.VoteExtension
			require.NoError(t, ve.Unmarshal(res.VoteExtension))
			if ve.MilestoneProposition != nil {
				withProp++
			}
			ec.Votes = append(ec.Votes, abci.ExtendedVoteInfo{
				Validator:     abci.Validator{Address: valKeys[i].PubKey().Address(), Power: v.VotingPower},
				VoteExtension: res.VoteExtension, ExtensionSignature: []byte("sig"), NonRpVoteExtension: res.NonRpExtension, NonRpExtensionSignature: []byte("sig"),
				BlockIdFlag: cmtproto.BlockIDFlagCommit,
			})
		}
		bz, err := ec.Marshal()
		require.NoError(t, err)
		use(0)
		_, err = app.PreBlocker(ctx.WithBlockHeight(e+1), &abci.RequestFinalizeBlock{Height: e + 1, Txs: [][]byte{bz}, ProposerAddress: common.FromHex(validators[0].Signer)})
		require.NoError(t, err)
		return withProp
	}
	state := func() (borTypes.Span, uint64) {
		s, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		lm, err := app.MilestoneKeeper.GetLastMilestone(ctx)
		require.NoError(t, err)
		return s, lm.EndBlock
	}

	// Control: an ordinary block. All 10 validators propose and the milestone advances.
	n := step(int64(M), nil)
	s, lm := state()
	t.Logf("control height: %d/10 propose; last milestone %d -> %d; span id %d", n, M, lm, s.Id)
	require.Equal(t, 10, n)
	require.Greater(t, lm, M)
	spanBefore := s.Id

	// 12 blocks carrying the 49 topups (more than the change-producer threshold of 10).
	// With the fix, every validator still builds its milestone proposition, milestones keep
	// advancing, and the healthy producer is never rotated.
	// (A milestone can still skip a single height for protocol reasons, e.g. right after a span
	// is created, so progress is checked over all 12 blocks, not per height.)
	e := int64(M) + 1
	lmBefore := lm
	for b := 1; b <= 12; b++ {
		n = step(e, padding)
		s, lm = state()
		t.Logf("loaded block %2d: %d/10 propose; last milestone %d; span id %d; bor head %d", b, n, lm, s.Id, bor.head())
		require.Equal(t, 10, n, "every validator must still include a milestone proposition with many side txs in the block")
		require.Equal(t, spanBefore, s.Id, "the healthy producer must not be rotated")
		e++
	}
	require.GreaterOrEqual(t, lm, lmBefore+10, "milestones must keep advancing while blocks carry many side txs")
}

// TestExtendVote_ProposesWithManySideTxs_Purley checks the same load on the Purley vote path (live on
// mainnet when the fix ships): every validator, including the 20 ms-L1 ones, must emit a vote
// extension that carries a milestone proposition and that VerifyVoteExtension accepts.
func TestExtendVote_ProposesWithManySideTxs_Purley(t *testing.T) {
	requireAllPropose(t, runLoadedHeight(t, 49, 0, 20*time.Millisecond, 0, true))
}

// TestMilestoneFirst_SlowSideTx_Purley is the class test: ONE side tx whose L1 call hangs past the
// whole ExtendVote budget (any slow side handler) must not remove the milestone proposition.
//
// This is the test that pins the reorder itself: the 49-topup tests also pass because of the
// per-round L1 head cache in helper, so they do not prove the ExtendVote order on their own.
func TestMilestoneFirst_SlowSideTx_Purley(t *testing.T) {
	requireAllPropose(t, runLoadedHeight(t, 1, 0, 2*time.Second, 0, true))
}

// TestMilestoneFirst_SlowSideTx_Legacy is the same class test on the pre-Purley vote path.
func TestMilestoneFirst_SlowSideTx_Legacy(t *testing.T) {
	requireAllPropose(t, runLoadedHeight(t, 1, 0, 2*time.Second, 0, false))
}

// TestMilestoneFirst_SlowBorKeepsSideTxVotes: the proposition now runs first, so a Bor node that
// hangs must not use the whole budget. Every validator must still vote on the side tx.
func TestMilestoneFirst_SlowBorKeepsSideTxVotes(t *testing.T) {
	for _, purley := range []bool{false, true} {
		r := runLoadedHeight(t, 1, 0, 2*time.Millisecond, 2*time.Second, purley)
		require.Equal(t, 10, r.withSideVote, "purley=%v: every validator must vote on the side tx when Bor is slow", purley)
	}
}

// TestMilestoneFirst_SlowBorNoSideTxsKeepsProposition: the half-budget cap only applies when side
// txs wait for votes. In a block with none, a Bor that needs more than half the budget (but less
// than all of it) must still produce a proposition, as before this change.
func TestMilestoneFirst_SlowBorNoSideTxsKeepsProposition(t *testing.T) {
	for _, purley := range []bool{false, true} {
		r := runLoadedHeight(t, 0, 0, 2*time.Millisecond, slowBorDelay, purley)
		require.Equal(t, 10, r.withProp, "purley=%v: no side txs, so the proposition keeps the whole budget", purley)
		// Txs without a side handler (here a fee withdrawal) get no vote, so they reserve no time.
		r = runLoadedHeight(t, 0, 2, 2*time.Millisecond, slowBorDelay, purley)
		require.Equal(t, 10, r.withProp, "purley=%v: only plain txs, so the proposition keeps the whole budget", purley)
	}
}

// slowBorDelay makes one proposition's Bor round trips take more than half of the 800 ms
// ExtendVote budget but well under all of it.
const slowBorDelay = 250 * time.Millisecond

// TestMilestoneFirst_SlowBorAndSlowPrefetchKeepSideTxVotes: after a slow Bor used its half of the
// budget, a slow batch prefetch must not use the rest. Single receipt lookups are fast, so every
// validator must still vote on the side tx.
func TestMilestoneFirst_SlowBorAndSlowPrefetchKeepSideTxVotes(t *testing.T) {
	for _, purley := range []bool{false, true} {
		r := runLoaded(t, loadOpts{nTx: 1, rtt: 2 * time.Millisecond, borDelay: 2 * time.Second, batch: 2 * time.Second, purley: purley})
		require.Equal(t, 10, r.withSideVote, "purley=%v: prefetch must leave time for side handlers", purley)
	}
}

type loadResult struct{ withProp, withSideVote int }

func requireAllPropose(t *testing.T, r loadResult) {
	t.Helper()
	require.Equal(t, 10, r.withProp, "every validator must still include a milestone proposition with many side txs in the block")
}

// runLoadedHeight puts nTx submitter topups in the block, gives every validator an L1 endpoint with
// round trip rtt, runs the real ExtendVote + VerifyVoteExtension (Purley or legacy path) and
// counts the validators whose vote extension has a milestone proposition and a side-tx vote.
// Every Bor request waits borDelay. plainTxs adds txs with no side handler after the topups.
type loadOpts struct {
	nTx, plainTxs        int
	rtt, borDelay, batch time.Duration
	purley               bool
}

func runLoadedHeight(t *testing.T, nTx, plainTxs int, rtt, borDelay time.Duration, purley bool) loadResult {
	t.Helper()
	return runLoaded(t, loadOpts{nTx: nTx, plainTxs: plainTxs, rtt: rtt, borDelay: borDelay, purley: purley})
}

func runLoaded(t *testing.T, o loadOpts) loadResult {
	t.Helper()
	nTx, plainTxs, rtt, borDelay, purley := o.nTx, o.plainTxs, o.rtt, o.borDelay, o.purley
	const M = uint64(100)
	submitter, app, ctx, valKeys := SetupAppWithABCICtxAndValidators(t, 10)
	// Activate Purley (and the live forks) only for the tested height, after app setup.
	compact := int64(0)
	if purley {
		compact = int64(M + 1)
	}
	forkHeights{compact: compact, phuket: 1, zurich: 1, ithaca: 1, kyoto: 1}.set(t)
	origRio := helper.GetRioHeight()
	t.Cleanup(func() { helper.SetRioHeight(origRio) })
	helper.SetRioHeight(int64(M + 1))
	validators := app.StakeKeeper.GetAllValidators(ctx)
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1}})

	p1 := common.HexToAddress(validators[0].Signer)
	var mu sync.Mutex
	bor := &borSim{IContractCaller: new(helpermocks.IContractCaller)}
	for n := uint64(0); n <= M+3; n++ {
		bor.produce(p1)
	}
	borSrv := fakeBorRPC(t, bor, &mu, borDelay)
	defer borSrv.Close()
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{Id: 1, StartBlock: 1, EndBlock: 300, BorChainId: "test",
		ValidatorSet: stakeTypes.ValidatorSet{Validators: validators, Proposer: validators[0]}, SelectedProducers: []stakeTypes.Validator{*validators[0]}}))
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, milestoneTypes.Milestone{MilestoneId: "1", StartBlock: 0, EndBlock: M, Hash: bor.headers[M].Hash().Bytes(), BorChainId: "test"}))
	require.NoError(t, app.MilestoneKeeper.SetLastMilestoneBlock(ctx, M))

	me := submitter.PubKey().Address().String()
	seq := app.AccountKeeper.GetAccount(ctx, submitter.PubKey().Address().Bytes()).GetSequence()
	l1Tx := common.HexToHash("0x6a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9")
	var padding [][]byte
	for i := 0; i < nTx; i++ {
		bz, err := buildSignedTxWithSequence(&topUpTypes.MsgTopupTx{Proposer: me, User: me, Fee: ante.DefaultFeeWantedPerTx[0].Amount.Add(math.NewInt(1)),
			TxHash: l1Tx.Bytes(), LogIndex: uint64(i), BlockNumber: 100}, ctx, submitter, app, seq+uint64(i))
		require.NoError(t, err)
		padding = append(padding, bz)
	}
	for i := 0; i < plainTxs; i++ {
		bz, err := buildSignedTxWithSequence(&topUpTypes.MsgWithdrawFeeTx{Proposer: me, Amount: math.NewInt(1)},
			ctx, submitter, app, seq+uint64(nTx+i))
		require.NoError(t, err)
		padding = append(padding, bz)
	}
	blockHash := common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000008dead")
	prevCommit, _, _, err := buildExtensionCommits(t, app, blockHash, validators, valKeys, 2, nil)
	require.NoError(t, err)

	e := int64(M) + 1
	var r loadResult
	for i := range validators {
		l1 := fakeL1WithBatchDelay(t, rtt, o.batch)
		t.Cleanup(l1.Close)
		cc, err := helper.NewContractCaller()
		require.NoError(t, err)
		l1rc, err := rpc.Dial(l1.URL)
		require.NoError(t, err)
		borrc, err := rpc.Dial(borSrv.URL)
		require.NoError(t, err)
		cc.MainChainRPCClient, cc.MainChainClient, cc.MainChainTimeout = l1rc, ethclient.NewClient(l1rc), 5*time.Second
		cc.BorChainRPCClient, cc.BorChainClient, cc.BorChainTimeout = borrc, ethclient.NewClient(borrc), 5*time.Second
		app.caller = cc
		app.TopupKeeper.SetContractCaller(cc)
		app.BorKeeper.SetContractCaller(cc)
		app.MilestoneKeeper.IContractCaller = cc

		res, err := app.ExtendVoteHandler()(ctx.WithBlockHeight(e), &abci.RequestExtendVote{Txs: append([][]byte{prevCommit}, padding...), Height: e, Hash: blockHash})
		require.NoError(t, err)
		vr, err := app.VerifyVoteExtensionHandler()(ctx.WithBlockHeight(e), &abci.RequestVerifyVoteExtension{
			Height: e, Hash: blockHash, ValidatorAddress: common.FromHex(validators[i].Signer),
			VoteExtension: res.VoteExtension, NonRpVoteExtension: res.NonRpExtension,
		})
		require.NoError(t, err)
		require.Equal(t, abci.ResponseVerifyVoteExtension_ACCEPT, vr.Status, "validator %d vote extension must be accepted", i)
		var ve sidetxs.VoteExtension
		require.NoError(t, ve.Unmarshal(res.VoteExtension))
		if ve.MilestoneProposition != nil {
			r.withProp++
		}
		if len(ve.SideTxResponses) > 0 {
			r.withSideVote++
		}
	}
	t.Logf("purley=%v, %d side txs, L1 rtt %s, Bor delay %s: %d/10 propose, %d/10 vote on side txs",
		purley, nTx, rtt, borDelay, r.withProp, r.withSideVote)
	return r
}
