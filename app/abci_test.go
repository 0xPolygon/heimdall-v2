package app

import (
	"encoding/json"
	"testing"

	"github.com/0xPolygon/heimdall-v2/engine"
	mocks "github.com/0xPolygon/heimdall-v2/engine/mock"
	"github.com/0xPolygon/heimdall-v2/helper"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type ABCIAppTestSuite struct {
	suite.Suite
	engineClient *mocks.MockExecutionEngineClient
}

func TestABCIAppTestSuite(t *testing.T) {
	suite.Run(t, new(ABCIAppTestSuite))
}

func (suite *ABCIAppTestSuite) SetupTest() {
	suite.engineClient = mocks.NewMockExecutionEngineClient(gomock.NewController(suite.T()))
}

// TestELRewind tests the scenario where the EL is rewound
// to an earlier block and we are next proposer
func (s *ABCIAppTestSuite) TestELRewind() {
	setupAppResult := SetupApp(s.T(), 1)
	testApp := setupAppResult.App

	testApp.ExecutionEngineClient = s.engineClient
	ctx := testApp.BaseApp.NewContext(true)
	vals := testApp.StakeKeeper.GetAllValidators(ctx)
	valAddr := common.FromHex(vals[0].Signer)
	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}

	validatorPrivKeys := setupAppResult.ValidatorKeys
	helper.SetTestPrivPubKey(setupAppResult.ValidatorKeys[0])

	initExecStateMetadata := checkpointTypes.ExecutionStateMetadata{
		FinalBlockHash:    common.Hex2Bytes("1e90773e7f3fa2fd2003476edfd263418ee5fa357c3df0035cf646aecedcca2e"),
		LatestBlockNumber: 1,
	}

	state := &engine.ForkChoiceState{
		HeadHash:           common.Hash(initExecStateMetadata.FinalBlockHash),
		SafeBlockHash:      common.Hash(initExecStateMetadata.FinalBlockHash),
		FinalizedBlockHash: common.Hash{},
	}

	err := testApp.CheckpointKeeper.SetExecutionStateMetadata(ctx, initExecStateMetadata)
	s.Require().NoError(err)
	extVoteInfo := setupExtendedVoteInfo(s.T(), cmtTypes.BlockIDFlagCommit, common.Hex2Bytes(TxHash1), common.Hex2Bytes(TxHash2), cometVal, validatorPrivKeys[0])

	choice := engine.ForkchoiceUpdatedResponse{
		PayloadId:     hexutil.EncodeUint64(1),
		PayloadStatus: engine.PayloadStatus{Status: "VALID"},
	}

	payload := engine.Payload{
		ExecutionPayload: engine.ExecutionPayload{
			BlockNumber: hexutil.EncodeUint64(2),
		},
	}

	s.engineClient.EXPECT().ForkchoiceUpdatedV2(gomock.Any(), state, gomock.Any()).Return(&choice, nil).Times(1)
	s.engineClient.EXPECT().GetPayloadV2(gomock.Any(), choice.PayloadId).Return(&payload, nil).Times(1)

	dummyReqProposal := abci.RequestPrepareProposal{
		Txs: [][]byte{},
		LocalLastCommit: abci.ExtendedCommitInfo{
			Votes: []abci.ExtendedVoteInfo{extVoteInfo},
		},
		Height: 3,
	}

	resp, err := testApp.PrepareProposal(&dummyReqProposal)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().Len(resp.Txs, 1)

	var metadata HeimdallMetadata
	err = json.Unmarshal(resp.Txs[0], &metadata)
	s.Require().NoError(err)

}
