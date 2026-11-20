package app

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// TestExtendVoteHandler_CheckpointBufferExpiryUsesRequestTime fails if the handler stops
// setting the context block time from the request, which the keeper-level expiry tests can't see.
func TestExtendVoteHandler_CheckpointBufferExpiryUsesRequestTime(t *testing.T) {
	const (
		runHeight  = int64(3)
		bufferTime = 100 * time.Second
		tBuffer    = int64(1_000_000)
	)
	expiry := tBuffer + int64(bufferTime.Seconds())

	cases := []struct {
		name     string
		purley   bool
		reqTime  int64
		wantVote sidetxs.Vote
	}{
		{name: "pre-Purley: unexpired buffer is not consulted", purley: false, reqTime: expiry - 1, wantVote: sidetxs.Vote_VOTE_YES},
		{name: "Purley: just before expiry", purley: true, reqTime: expiry - 1, wantVote: sidetxs.Vote_VOTE_NO},
		{name: "Purley: at expiry", purley: true, reqTime: expiry, wantVote: sidetxs.Vote_VOTE_YES},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.purley {
				helper.SetPurleyHeight(runHeight)
				t.Cleanup(func() { helper.SetPurleyHeight(0) })
			}

			priv, app, ctx, validatorPrivKeys := SetupAppWithABCICtx(t)
			ctx = ctx.WithBlockHeight(runHeight)
			validators := app.StakeKeeper.GetAllValidators(ctx)

			caller := new(helpermocks.IContractCaller)
			caller.On("GetBorChainBlock", mock.Anything, mock.Anything).
				Return(&ethTypes.Header{Number: big.NewInt(10)}, nil)
			caller.On("GetBorChainBlockInfoInBatch", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
				Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)
			caller.On("CheckIfBlocksExist", mock.Anything, mock.Anything).Return(true, nil)
			rootHash := common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000b0ff0001")
			caller.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(rootHash, nil)

			app.MilestoneKeeper = milestoneKeeper.NewKeeper(app.AppCodec(), authTypes.NewModuleAddress(govtypes.ModuleName).String(), runtime.NewKVStoreService(app.GetKey(milestoneTypes.StoreKey)), caller)
			app.CheckpointKeeper = checkpointKeeper.NewKeeper(app.AppCodec(), runtime.NewKVStoreService(app.GetKey(checkpointTypes.StoreKey)), authTypes.NewModuleAddress(govtypes.ModuleName).String(), &app.StakeKeeper, app.ChainManagerKeeper, &app.TopupKeeper, caller)
			app.BorKeeper = borKeeper.NewKeeper(app.AppCodec(), runtime.NewKVStoreService(app.GetKey(borTypes.StoreKey)), authTypes.NewModuleAddress(govtypes.ModuleName).String(), app.ChainManagerKeeper, &app.StakeKeeper, nil, nil)
			app.BorKeeper.SetContractCaller(caller)
			app.MilestoneKeeper.IContractCaller = caller
			app.caller = caller

			chainParams, err := app.ChainManagerKeeper.GetParams(ctx)
			require.NoError(t, err)
			borChainID := chainParams.ChainParams.BorChainId

			params, err := app.CheckpointKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.CheckpointBufferTime = bufferTime
			require.NoError(t, app.CheckpointKeeper.SetParams(ctx, params))

			require.NoError(t, app.CheckpointKeeper.SetCheckpointBuffer(ctx, checkpointTypes.Checkpoint{
				Id:         1,
				Proposer:   validators[0].Signer,
				StartBlock: 0,
				EndBlock:   63,
				RootHash:   common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000b0ff0000"),
				BorChainId: borChainID,
				Timestamp:  uint64(tBuffer),
			}))

			valSet, err := app.StakeKeeper.GetValidatorSet(ctx)
			require.NoError(t, err)
			dividends, err := app.TopupKeeper.GetAllDividendAccounts(ctx)
			require.NoError(t, err)
			accountRoot, err := hmTypes.GetAccountRootHash(dividends)
			require.NoError(t, err)

			msg := &checkpointTypes.MsgCheckpoint{
				Proposer:        valSet.Proposer.Signer,
				StartBlock:      0,
				EndBlock:        63,
				RootHash:        rootHash,
				AccountRootHash: accountRoot,
				BorChainId:      borChainID,
			}
			txBytes, err := buildSignedTx(msg, ctx, priv, app)
			require.NoError(t, err)
			extCommitBytes, _, _, err := buildExtensionCommits(t, app, common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000001dead"), validators, validatorPrivKeys, 2, nil)
			require.NoError(t, err)

			resp, err := app.ExtendVoteHandler()(ctx, &abci.RequestExtendVote{
				Txs:    [][]byte{extCommitBytes, txBytes},
				Hash:   []byte("test-hash"),
				Height: runHeight,
				Time:   time.Unix(tc.reqTime, 0),
			})
			require.NoError(t, err)

			var ve sidetxs.VoteExtension
			require.NoError(t, gogoproto.Unmarshal(resp.VoteExtension, &ve))
			txHash := cmtTypes.Tx(txBytes).Hash()
			var got []sidetxs.Vote
			for _, r := range ve.SideTxResponses {
				if bytes.Equal(r.TxHash, txHash) {
					got = append(got, r.Result)
				}
			}
			require.Equal(t, []sidetxs.Vote{tc.wantVote}, got)
		})
	}
}
