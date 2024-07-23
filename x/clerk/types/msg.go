package types

import (
	hexCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	hm2types "github.com/0xPolygon/heimdall-v2/types"
)

// NewMsgEventRecord - construct state msg
func NewMsgEventRecord(
	from sdk.AccAddress,
	txHash string,
	logIndex uint64,
	blockNumber uint64,
	id uint64,
	contractAddress sdk.AccAddress,
	data hm2types.HexBytes,
	chainID string,

) MsgEventRecordRequest {
	contractAddressBytes, err := hexCodec.NewHexCodec().BytesToString(contractAddress)
	if err != nil {
		contractAddressBytes = ""
	}

	fromBytes, err := hexCodec.NewHexCodec().BytesToString(from)
	if err != nil {
		fromBytes = ""
	}

	return MsgEventRecordRequest{
		From:            fromBytes,
		TxHash:          txHash,
		LogIndex:        logIndex,
		BlockNumber:     blockNumber,
		ID:              id,
		ContractAddress: contractAddressBytes,
		Data:            data,
		ChainID:         chainID,
	}
}

// Route Implements Msg.
func (msg MsgEventRecordRequest) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgEventRecordRequest) Type() string { return "event-record" }

// ValidateBasic Implements Msg.
func (msg MsgEventRecordRequest) ValidateBasic() error {
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

	// DO NOT REMOVE THIS CHANGE
	if msg.Data.Size() > helper.LegacyMaxStateSyncSize {
		return ErrSizeExceed
	}

	return nil
}

// GetTxHash Returns tx hash
func (msg MsgEventRecordRequest) GetTxHash() string {
	return msg.TxHash
}

// GetLogIndex Returns log index
func (msg MsgEventRecordRequest) GetLogIndex() uint64 {
	return msg.LogIndex
}
