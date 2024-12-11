package types

import (
	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/helper"
	hexCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewMsgEventRecord - construct state msg
func NewMsgEventRecord(
	from string,
	txHash string,
	logIndex uint64,
	blockNumber uint64,
	id uint64,
	contractAddress sdk.AccAddress,
	data []byte,
	chainID string,
) MsgEventRecord {
	contractAddressBytes, err := hexCodec.NewHexCodec().BytesToString(contractAddress)
	if err != nil {
		contractAddressBytes = ""
	}

	return MsgEventRecord{
		From:            util.FormatAddress(from),
		TxHash:          txHash,
		LogIndex:        logIndex,
		BlockNumber:     blockNumber,
		Id:              id,
		ContractAddress: contractAddressBytes,
		Data:            data,
		ChainId:         chainID,
	}
}

// Route Implements Msg
func (msg MsgEventRecord) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgEventRecord) Type() string { return "event-record" }

// ValidateBasic Implements Msg
func (msg MsgEventRecord) ValidateBasic() error {
	bytes, err := hexCodec.NewHexCodec().StringToBytes(msg.From)
	if err != nil {
		return sdkerrors.ErrInvalidAddress
	}

	tempFrom := sdk.AccAddress(bytes)
	if tempFrom.Empty() {
		return sdkerrors.ErrInvalidAddress
	}

	if msg.TxHash == "" {
		return ErrEmptyTxHash
	}

	// TODO HV2: Double check this
	// DO NOT REMOVE THIS CHANGE
	if len(msg.Data) > helper.MaxStateSyncSize {
		return ErrSizeExceed
	}

	return nil
}

// GetTxHash Returns tx hash
func (msg MsgEventRecord) GetTxHash() string {
	return msg.TxHash
}

// GetLogIndex Returns log index
func (msg MsgEventRecord) GetLogIndex() uint64 {
	return msg.LogIndex
}
