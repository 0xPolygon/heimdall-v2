package app

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xPolygon/heimdall-v2/engine"
	mocks "github.com/0xPolygon/heimdall-v2/engine/mock"
	"github.com/0xPolygon/heimdall-v2/helper"
	contractMock "github.com/0xPolygon/heimdall-v2/helper/mocks"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type ABCIAppTestSuite struct {
	suite.Suite
	engineClient   *mocks.MockExecutionEngineClient
	contractCaller contractMock.IContractCaller
}

func TestABCIAppTestSuite(t *testing.T) {
	suite.Run(t, new(ABCIAppTestSuite))
}

func (suite *ABCIAppTestSuite) SetupTest() {
	suite.engineClient = mocks.NewMockExecutionEngineClient(gomock.NewController(suite.T()))
	suite.contractCaller = contractMock.IContractCaller{}
}

// TestELRewindProposer tests the scenario where the proposer's EL has rewound
// to an earlier block
func (s *ABCIAppTestSuite) TestELRewindProposer() {
	testApp, extVoteInfo, ctx := s.testSetup()
	choice := engine.ForkchoiceUpdatedResponse{
		PayloadId:     hexutil.EncodeUint64(1),
		PayloadStatus: engine.PayloadStatus{Status: "VALID"},
	}

	proposerPayloadRes := engine.Payload{
		ExecutionPayload: engine.ExecutionPayload{
			BlockNumber: hexutil.EncodeUint64(1),
		},
	}

	s.engineClient.EXPECT().ForkchoiceUpdatedV2(gomock.Any(), gomock.Any(), gomock.Any()).Return(&choice, nil).Times(1)
	s.engineClient.EXPECT().GetPayloadV2(gomock.Any(), choice.PayloadId).Return(&proposerPayloadRes, nil).Times(1)

	dummyReqPrepareProposal := abci.RequestPrepareProposal{
		Txs: [][]byte{},
		LocalLastCommit: abci.ExtendedCommitInfo{
			Votes: []abci.ExtendedVoteInfo{extVoteInfo},
		},
		Height: 3,
	}

	resp1, err := testApp.PrepareProposal(&dummyReqPrepareProposal)
	s.Require().NoError(err)
	s.Require().NotNil(resp1)

	s.Require().Len(resp1.Txs, 1)

	var metadata HeimdallMetadata
	err = json.Unmarshal(resp1.Txs[0], &metadata)
	s.Require().NoError(err)

	var executionPayload engine.ExecutionPayload
	err = json.Unmarshal(metadata.MarshaledExecutionPayload, &executionPayload)
	s.Require().NoError(err)

	processorPayloadRes := engine.NewPayloadResponse{
		Status: "INVALID",
	}

	header := types.Header{
		Number: big.NewInt(2),
	}

	s.contractCaller.On("GetBorChainBlock", ctx, gomock.Nil()).Return(header, nil).Once()
	s.engineClient.EXPECT().NewPayloadV2(gomock.Any(), executionPayload).Return(&processorPayloadRes, nil).Times(1)

	dummyReqProcessProposal := abci.RequestProcessProposal{
		Txs:    resp1.Txs,
		Height: 3,
	}

	resp2, err := testApp.ProcessProposal(&dummyReqProcessProposal)
	s.Require().NoError(err)
	s.Require().NotNil(resp2)
	s.Require().Equal(resp2.Status, abci.ResponseProcessProposal_REJECT)
}

// TestELRewindProposer is the same as TestELRewindProposer with the difference
// that the verifier's EL has rewound to an earlier block
func (s *ABCIAppTestSuite) TestELRewindProcessor() {
	testApp, extVoteInfo, ctx := s.testSetup()
	choice := engine.ForkchoiceUpdatedResponse{
		PayloadId:     hexutil.EncodeUint64(1),
		PayloadStatus: engine.PayloadStatus{Status: "VALID"},
	}

	proposerPayloadRes := engine.Payload{
		ExecutionPayload: engine.ExecutionPayload{
			BlockNumber: hexutil.EncodeUint64(5),
		},
	}

	s.engineClient.EXPECT().ForkchoiceUpdatedV2(gomock.Any(), gomock.Any(), gomock.Any()).Return(&choice, nil).Times(1)
	s.engineClient.EXPECT().GetPayloadV2(gomock.Any(), choice.PayloadId).Return(&proposerPayloadRes, nil).Times(1)

	dummyReqPrepareProposal := abci.RequestPrepareProposal{
		Txs: [][]byte{},
		LocalLastCommit: abci.ExtendedCommitInfo{
			Votes: []abci.ExtendedVoteInfo{extVoteInfo},
		},
		Height: 3,
	}

	resp1, err := testApp.PrepareProposal(&dummyReqPrepareProposal)
	s.Require().NoError(err)
	s.Require().NotNil(resp1)

	s.Require().Len(resp1.Txs, 1)

	var metadata HeimdallMetadata
	err = json.Unmarshal(resp1.Txs[0], &metadata)
	s.Require().NoError(err)

	var executionPayload engine.ExecutionPayload
	err = json.Unmarshal(metadata.MarshaledExecutionPayload, &executionPayload)
	s.Require().NoError(err)

	// Execution client might trigger a sync knowing it's behind
	processorPayloadRes := engine.NewPayloadResponse{
		Status: "SYNCING",
	}

	header := types.Header{
		Number: big.NewInt(4),
	}

	s.contractCaller.On("GetBorChainBlock", ctx, gomock.Nil()).Return(header, nil).Once()
	s.engineClient.EXPECT().NewPayloadV2(gomock.Any(), executionPayload).Return(&processorPayloadRes, nil).Times(1)

	dummyReqProcessProposal := abci.RequestProcessProposal{
		Txs:    resp1.Txs,
		Height: 3,
	}

	resp2, err := testApp.ProcessProposal(&dummyReqProcessProposal)
	s.Require().NoError(err)
	s.Require().NotNil(resp2)
	s.Require().Equal(resp2.Status, abci.ResponseProcessProposal_REJECT)
}

func (s *ABCIAppTestSuite) testSetup() (*HeimdallApp, abci.ExtendedVoteInfo, sdk.Context) {
	setupAppResult := SetupApp(s.T(), 1)
	testApp := setupAppResult.App

	testApp.ExecutionEngineClient = s.engineClient
	testApp.caller = &s.contractCaller
	ctx := testApp.BaseApp.NewContext(true)
	vals := testApp.StakeKeeper.GetAllValidators(ctx)
	valAddr := common.FromHex(vals[0].Signer)
	cometVal := abci.Validator{
		Address: valAddr,
		Power:   vals[0].VotingPower,
	}

	validatorPrivKeys := setupAppResult.ValidatorKeys
	helper.SetTestPrivPubKey(setupAppResult.ValidatorKeys[0])
	extVoteInfo := setupExtendedVoteInfo(s.T(), cmtTypes.BlockIDFlagCommit, common.Hex2Bytes(TxHash1), common.Hex2Bytes(TxHash2), cometVal, validatorPrivKeys[0], 0)
	return testApp, extVoteInfo, ctx
}
