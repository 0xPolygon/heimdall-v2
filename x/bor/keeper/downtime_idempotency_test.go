package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// A producer can get the same MsgSetProducerDowntime approved more than once.
// Before the gate, each approval appended another veBlop span at the same start
// block; at/after Kyoto the duplicate is a no-op.
func (s *KeeperTestSuite) TestPostHandleSetProducerDowntimeIdempotency() {
	require := s.Require()

	origKyoto := helper.GetKyotoHeight()
	defer helper.SetKyotoHeight(origKyoto)

	id1, id2, id3 := uint64(1), uint64(2), uint64(3)
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
	addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()

	primeStakeMocks := func() {
		s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).Return(stakeTypes.ValidatorSet{
			Validators: []*stakeTypes.Validator{
				{ValId: id1, Signer: addr1, VotingPower: 100},
				{ValId: id2, Signer: addr2, VotingPower: 100},
				{ValId: id3, Signer: addr3, VotingPower: 100},
			},
		}, nil).AnyTimes()
		s.stakeKeeper.EXPECT().GetSpanEligibleValidators(gomock.Any()).Return([]stakeTypes.Validator{
			{ValId: id1, Signer: addr1, VotingPower: 100},
			{ValId: id2, Signer: addr2, VotingPower: 100},
			{ValId: id3, Signer: addr3, VotingPower: 100},
		}).AnyTimes()
		s.stakeKeeper.EXPECT().GetValidatorFromValID(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
				switch vid {
				case id1:
					return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
				case id2:
					return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
				case id3:
					return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
				default:
					return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
				}
			}).AnyTimes()
		s.stakeKeeper.EXPECT().GetValIdFromAddress(gomock.Any(), addr1).Return(id1, nil).AnyTimes()
	}

	setVotesForAll := func(voteList []uint64) {
		require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
		for _, voter := range []uint64{id1, id2, id3} {
			require.NoError(s.borKeeper.SetProducerVotes(s.ctx, voter, types.ProducerVotes{Votes: voteList}))
		}
	}

	seedSpans := func() {
		valSet, vals := s.genTestValidators()
		require.NotEmpty(vals)
		spans := []types.Span{
			{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
			{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
			{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
		}
		for i := range spans {
			require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
		}
	}

	msg := &types.MsgSetProducerDowntime{
		Producer:      addr1,
		DowntimeRange: types.BlockRange{StartBlock: 150, EndBlock: 350},
	}
	post := func() error {
		return s.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))(s.ctx, msg, sidetxs.Vote_VOTE_YES)
	}
	lastSpanID := func() uint64 {
		last, err := s.borKeeper.GetLastSpan(s.ctx)
		require.NoError(err)
		return last.Id
	}

	prepare := func(kyotoHeight int64) {
		s.SetupTest()
		require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
		s.ctx = s.ctx.WithBlockHeight(500)
		helper.SetKyotoHeight(kyotoHeight)
		primeStakeMocks()
		setVotesForAll([]uint64{id1, id2, id3})
		require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))
		seedSpans()
	}

	s.Run("at/after kyoto, duplicate downtime creates no extra span", func() {
		prepare(1) // active at height 500

		require.NoError(post())
		require.Equal(uint64(3), lastSpanID())

		require.NoError(post())
		require.Equal(uint64(3), lastSpanID())
	})

	s.Run("below kyoto, legacy path appends a span each time", func() {
		prepare(1000) // inactive at height 500

		require.NoError(post())
		require.Equal(uint64(3), lastSpanID())

		require.NoError(post())
		require.Equal(uint64(4), lastSpanID())
	})
}
