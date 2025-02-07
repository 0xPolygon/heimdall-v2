package engine

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type JsonrpcRequest struct {
	ID      uint64        `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JsonrpcResponse struct {
	ID      int             `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *EthError       `json:"error"`
}

type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type EthError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err EthError) Error() string {
	return fmt.Sprintf("Error %d (%s)", err.Code, err.Message)
}

// Ethereum structs
// https://github.com/ethereum/execution-apis/blob/main/src/engine

type ForkChoiceState struct {
	HeadHash           common.Hash
	SafeBlockHash      common.Hash
	FinalizedBlockHash common.Hash
}

func (f ForkChoiceState) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		HeadBlockHash      string `json:"headBlockHash"`
		SafeBlockHash      string `json:"safeBlockHash"`
		FinalizedBlockHash string `json:"finalizedBlockHash"`
	}{
		f.HeadHash.Hex(),
		f.SafeBlockHash.Hex(),
		f.FinalizedBlockHash.Hex(),
	})
}

// PayloadAttributes represent the attributes required to start assembling a testPayload
type PayloadAttributes struct {
	Timestamp             hexutil.Uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address
	Withdrawals           []*Withdrawal
	ParentBeaconBlockRoot common.Hash
}

func (p PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp             string        `json:"timestamp"`
		PrevRandao            string        `json:"prevRandao"`
		SuggestedFeeRecipient string        `json:"suggestedFeeRecipient"`
		Withdrawals           []*Withdrawal `json:"withdrawals"`
		ParentBeaconBlockRoot string        `json:"parentBeaconBlockRoot,omitempty"`
	}{
		Timestamp:             p.Timestamp.String(),
		PrevRandao:            p.PrevRandao.Hex(),
		SuggestedFeeRecipient: p.SuggestedFeeRecipient.Hex(),
		Withdrawals:           p.Withdrawals,
		ParentBeaconBlockRoot: p.ParentBeaconBlockRoot.Hex(),
	})
}

type Withdrawal struct {
	Index     string `json:"index"`          // monotonically increasing identifier issued by consensus layer
	Validator string `json:"validatorIndex"` // index of validator associated with withdrawal
	Address   string `json:"address"`        // target address for withdrawn ether
	Amount    string `json:"amount"`         // value of withdrawal in Gwei
}

type ForkchoiceUpdatedResponse struct {
	PayloadId     string        `json:"payloadId"`
	PayloadStatus PayloadStatus `json:"payloadStatus"`
}

type PayloadStatus struct {
	Status          string `json:"status"`
	ValidationError string `json:"validationError"`
	LatestValidHash string `json:"latestValidHash"`
	CriticalError   string `json:"CriticalError"`
}

type Payload struct {
	ExecutionPayload      ExecutionPayload `json:"executionPayload"`
	BlockValue            string           `json:"blockValue"`
	BlobsBundle           BlobsBundle      `json:"blobsBundle"`
	ShouldOverrideBuilder bool             `json:"shouldOverrideBuilder"`
}

type ExecutionPayload struct {
	ParentHash    string        `json:"parentHash"`
	FeeRecipient  string        `json:"feeRecipient"`
	StateRoot     string        `json:"stateRoot"`
	ReceiptsRoot  string        `json:"receiptsRoot"`
	LogsBloom     string        `json:"logsBloom"`
	PrevRandao    string        `json:"prevRandao"`
	BlockNumber   string        `json:"blockNumber"`
	GasLimit      string        `json:"gasLimit"`
	GasUsed       string        `json:"gasUsed"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extraData"`
	BaseFeePerGas string        `json:"baseFeePerGas"`
	BlockHash     string        `json:"blockHash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []*Withdrawal `json:"withdrawals,omitempty"` // is this right?
	BlobGasUsed   string        `json:"blobGasUsed"`
	ExcessBlobGas string        `json:"excessBlobGas"`
}

type BlobsBundle struct {
	Commitments []string `json:"commitments"`
	Proofs      []string `json:"proofs"`
	Blobs       []string `json:"blobs"`
}

type NewPayloadResponse struct {
	Status          string `json:"status"`
	LatestValidHash string `json:"latestValidHash"`
	ValidationError string `json:"validationError"`
}
