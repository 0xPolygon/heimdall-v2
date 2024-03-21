package types

import (
	hexCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	hm2types "github.com/0xPolygon/heimdall-v2/types"
)

// TODO HV2 - how to handle conversation between sdk.AccAddress and string?
// NewMsgEventRecord - construct state msg
func NewMsgEventRecord(
	from sdk.AccAddress,
	txHash hm2types.HeimdallHash,
	logIndex uint64,
	blockNumber uint64,
	id uint64,
	contractAddress sdk.AccAddress,
	data hm2types.HexBytes,
	chainID string,

) MsgEventRecord {
	contractAddressBytes, err := hexCodec.NewHexCodec().BytesToString(contractAddress)
	if err != nil {
		contractAddressBytes = ""
	}

	fromBytes, err := hexCodec.NewHexCodec().BytesToString(from)
	if err != nil {
		fromBytes = ""
	}

	return MsgEventRecord{
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
func (msg MsgEventRecord) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgEventRecord) Type() string { return "event-record" }

// ValidateBasic Implements Msg.
func (msg MsgEventRecord) ValidateBasic() error {
	bytes, err := hexCodec.NewHexCodec().StringToBytes(msg.From)
	tempFrom := sdk.AccAddress(bytes)
	if tempFrom.Empty() || err != nil {
		return sdkerrors.ErrInvalidAddress
	}

	if msg.TxHash.Empty() {
		return sdkerrors.ErrInvalidAddress
	}

	// DO NOT REMOVE THIS CHANGE
	if msg.Data.Size() > helper.LegacyMaxStateSyncSize {
		return ErrSizeExceed
	}

	return nil
}

// TODO HV2 - I don't think we need this
// // GetSignBytes Implements Msg.
// func (msg MsgEventRecord) GetSignBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
// }

// GetSigners Implements Msg.
func (msg MsgEventRecord) GetSigners() []sdk.AccAddress {
	bytes, err := hexCodec.NewHexCodec().StringToBytes(msg.From)
	if err != nil {
		return nil
	}

	return []sdk.AccAddress{bytes}
}

// GetTxHash Returns tx hash
func (msg MsgEventRecord) GetTxHash() hm2types.HeimdallHash {
	return msg.TxHash
}

// GetLogIndex Returns log index
func (msg MsgEventRecord) GetLogIndex() uint64 {
	return msg.LogIndex
}

// GetSideSignBytes returns side sign bytes
func (msg MsgEventRecord) GetSideSignBytes() []byte {
	return nil
}
