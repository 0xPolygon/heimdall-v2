package types

import (
	"bytes"
	"errors"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"math/big"
	"strconv"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/types"
)

var (
	_ sdk.Msg = &MsgCheckpoint{}
	_ sdk.Msg = &MsgCheckpointAck{}
	_ sdk.Msg = &MsgCheckpointNoAck{}
)

// NewMsgCheckpointBlock creates new checkpoint message using mentioned arguments
func NewMsgCheckpointBlock(
	proposer string,
	startBlock uint64,
	endBlock uint64,
	rootHash []byte,
	accountRootHash []byte,
	borChainID string,
) *MsgCheckpoint {
	return &MsgCheckpoint{
		Proposer:        util.FormatAddress(proposer),
		StartBlock:      startBlock,
		EndBlock:        endBlock,
		RootHash:        rootHash,
		AccountRootHash: accountRootHash,
		BorChainId:      borChainID,
	}
}

func (msg MsgCheckpoint) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.RootHash, common.Hash{}.Bytes()) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", string(msg.RootHash))
	}

	if len(msg.RootHash) != common.HashLength {
		return ErrInvalidMsg.Wrapf("Invalid roothash length %v", len(msg.RootHash))
	}

	addrBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if msg.StartBlock >= msg.EndBlock || msg.EndBlock == 0 {
		return ErrInvalidMsg.Wrapf("End should be greater than to start block start block=%d,end block=%d", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpoint) GetSideSignBytes() []byte {
	// keccak256(abi.encoded(proposer, startBlock, endBlock, rootHash, accountRootHash, bor chain id))
	borChainID, _ := strconv.ParseUint(msg.BorChainId, 10, 64)

	ac := addressCodec.NewHexCodec()
	proposerBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		panic(errors.New("invalid proposer while getting side sign bytes for checkpoint msg"))
	}

	return types.AppendBytes32(
		proposerBytes,
		new(big.Int).SetUint64(msg.StartBlock).Bytes(),
		new(big.Int).SetUint64(msg.EndBlock).Bytes(),
		msg.RootHash,
		msg.AccountRootHash,
		new(big.Int).SetUint64(borChainID).Bytes(),
	)
}

var _ sdk.Msg = &MsgCheckpointAck{}

func NewMsgCheckpointAck(
	from string,
	number uint64,
	proposer string,
	startBlock uint64,
	endBlock uint64,
	rootHash []byte,
	txHash []byte,
	logIndex uint64,
) MsgCheckpointAck {
	return MsgCheckpointAck{
		From:       util.FormatAddress(from),
		Number:     number,
		Proposer:   proposer,
		StartBlock: startBlock,
		EndBlock:   endBlock,
		RootHash:   rootHash,
		TxHash:     txHash,
		LogIndex:   logIndex,
	}
}

// ValidateBasic validate basic
func (msg MsgCheckpointAck) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	addrBytes, err = ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if bytes.Equal(msg.RootHash, common.Hash{}.Bytes()) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", string(msg.RootHash))
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpointAck) GetSideSignBytes() []byte {
	return nil
}

var _ sdk.Msg = &MsgCheckpointNoAck{}

func NewMsgCheckpointNoAck(from string) MsgCheckpointNoAck {
	return MsgCheckpointNoAck{
		From: util.FormatAddress(from),
	}
}

func (msg MsgCheckpointNoAck) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	return nil
}
