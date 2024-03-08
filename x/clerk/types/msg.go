package types

import (
	// TODO HV2 - this is implemented in auth PR
	// hexCodec "github.com/0xPolygon/cosmos-sdk/codec/address/"
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
	return MsgEventRecord{
		// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
		// From:            hexCodec.BytesToString(from),
		TxHash:      txHash,
		LogIndex:    logIndex,
		BlockNumber: blockNumber,
		ID:          id,
		// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
		// ContractAddress: hexCodec.BytesToString(contractAddress),
		Data:    data,
		ChainID: chainID,
	}
}

// Route Implements Msg.
func (msg MsgEventRecord) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgEventRecord) Type() string { return "event-record" }

// TODO HV2 - fix errors
// ValidateBasic Implements Msg.
func (msg MsgEventRecord) ValidateBasic() error {
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// tempFrom := hexCodec.StringToBytes(msg.From)
	// if tempFrom.Empty() {
	// 	return sdkerrors.ErrInvalidAddress
	// }

	// TODO HV2 -
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

// TODO HV2 - fix errors
// GetSigners Implements Msg.
func (msg MsgEventRecord) GetSigners() []sdk.AccAddress {
	// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
	// return []sdk.AccAddress{hexCodec.StringToBytes(msg.From)}
	return []sdk.AccAddress{}
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
