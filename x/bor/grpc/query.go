package grpc

import (
	"context"
	"fmt"
	"math/big"

	proto "github.com/0xPolygon/polyproto/bor"
	commonproto "github.com/0xPolygon/polyproto/common"
	protoutil "github.com/0xPolygon/polyproto/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func (c *BorGRPCClient) GetRootHash(ctx context.Context, startBlock uint64, endBlock uint64) (string, error) {
	req := &proto.GetRootHashRequest{
		StartBlockNumber: startBlock,
		EndBlockNumber:   endBlock,
	}

	res, err := c.client.GetRootHash(ctx, req)
	if err != nil {
		return "", err
	}

	return res.RootHash, nil
}

func (c *BorGRPCClient) GetVoteOnHash(ctx context.Context, startBlock uint64, endBlock uint64, rootHash string, milestoneId string) (bool, error) {
	req := &proto.GetVoteOnHashRequest{
		StartBlockNumber: startBlock,
		EndBlockNumber:   endBlock,
		Hash:             rootHash,
		MilestoneId:      milestoneId,
	}

	res, err := c.client.GetVoteOnHash(ctx, req)
	if err != nil {
		return false, err
	}

	return res.Response, nil
}

func (c *BorGRPCClient) HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error) {
	blockNumberAsString := ToBlockNumArg(big.NewInt(blockID))

	req := &proto.GetHeaderByNumberRequest{
		Number: blockNumberAsString,
	}

	res, err := c.client.HeaderByNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Header == nil {
		return nil, ethereum.NotFound
	}

	return protoHeaderToEthHeader(res.Header), nil
}

func (c *BorGRPCClient) BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error) {
	blockNumberAsString := ToBlockNumArg(big.NewInt(blockID))

	req := &proto.GetBlockByNumberRequest{
		Number: blockNumberAsString,
	}

	res, err := c.client.BlockByNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Block == nil || res.Block.Header == nil {
		return nil, ethereum.NotFound
	}

	header := protoHeaderToEthHeader(res.Block.Header)
	return ethTypes.NewBlock(header, nil, nil, nil), nil
}

func (c *BorGRPCClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	req := &proto.ReceiptRequest{
		Hash: protoutil.ConvertHashToH256(txHash),
	}

	res, err := c.client.TransactionReceipt(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Receipt == nil {
		return nil, ethereum.NotFound
	}

	return receiptResponseToTypesReceipt(res.Receipt), nil
}

func (c *BorGRPCClient) BorBlockReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	req := &proto.ReceiptRequest{
		Hash: protoutil.ConvertHashToH256(txHash),
	}

	res, err := c.client.BorBlockReceipt(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Receipt == nil {
		return nil, ethereum.NotFound
	}

	return receiptResponseToTypesReceipt(res.Receipt), nil
}

// GetAuthor returns the author of the block at blockNum. Nil blockNum resolves
// to the latest block (via ToBlockNumArg, which maps nil to "latest").
func (c *BorGRPCClient) GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	req := &proto.GetAuthorRequest{Number: ToBlockNumArg(blockNum)}

	res, err := c.client.GetAuthor(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.Author == nil {
		return nil, fmt.Errorf("bor grpc GetAuthor: nil author")
	}

	arr := protoutil.ConvertH160toAddress(res.Author)
	addr := common.BytesToAddress(arr[:])
	return &addr, nil
}

// GetTdByHash returns the total difficulty of the block identified by hash.
func (c *BorGRPCClient) GetTdByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	req := &proto.GetTdByHashRequest{Hash: protoutil.ConvertHashToH256(hash)}
	res, err := c.client.GetTdByHash(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.TotalDifficulty, nil
}

// GetTdByNumber returns the total difficulty of the block at blockNum.
// Nil blockNum resolves to the latest block.
func (c *BorGRPCClient) GetTdByNumber(ctx context.Context, blockNum *big.Int) (uint64, error) {
	req := &proto.GetTdByNumberRequest{Number: ToBlockNumArg(blockNum)}
	res, err := c.client.GetTdByNumber(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.TotalDifficulty, nil
}

// GetBlockInfoInBatch returns headers, total difficulties, and authors for the
// inclusive block range [start, end]. Returns up to (end-start+1) entries; a
// shorter slice means the server encountered a missing block or other error
// mid-range, matching the HTTP eth_getHeaderByNumber batch semantics.
// Returns an error for invalid input ranges.
func (c *BorGRPCClient) GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	if start < 0 || end < 0 || end < start {
		return nil, nil, nil, fmt.Errorf("invalid range [%d,%d]", start, end)
	}

	req := &proto.GetBlockInfoInBatchRequest{
		StartBlockNumber: uint64(start),
		EndBlockNumber:   uint64(end),
	}
	res, err := c.client.GetBlockInfoInBatch(ctx, req)
	if err != nil {
		return nil, nil, nil, err
	}

	n := len(res.Blocks)
	headers := make([]*ethTypes.Header, 0, n)
	tds := make([]uint64, 0, n)
	authors := make([]common.Address, 0, n)

	for _, b := range res.Blocks {
		if b == nil || b.Header == nil {
			break
		}
		headers = append(headers, protoHeaderToEthHeader(b.Header))
		tds = append(tds, b.TotalDifficulty)

		var addr common.Address
		if b.Author != nil {
			arr := protoutil.ConvertH160toAddress(b.Author)
			addr = common.BytesToAddress(arr[:])
		}
		authors = append(authors, addr)
	}

	return headers, tds, authors, nil
}

func receiptResponseToTypesReceipt(receipt *proto.Receipt) *ethTypes.Receipt {
	// Bloom and Logs have been intentionally left out as they are not used in the current implementation
	return &ethTypes.Receipt{
		Type:              uint8(receipt.Type),
		PostState:         receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		TxHash:            protoutil.ConvertH256ToHash(receipt.TxHash),
		ContractAddress:   protoutil.ConvertH160toAddress(receipt.ContractAddress),
		GasUsed:           receipt.GasUsed,
		EffectiveGasPrice: big.NewInt(receipt.EffectiveGasPrice),
		BlobGasUsed:       receipt.BlobGasUsed,
		BlobGasPrice:      big.NewInt(receipt.BlobGasPrice),
		BlockHash:         protoutil.ConvertH256ToHash(receipt.BlockHash),
		BlockNumber:       big.NewInt(receipt.BlockNumber),
		TransactionIndex:  uint(receipt.TransactionIndex),
	}
}

func ToBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	if number.IsInt64() {
		return rpc.BlockNumber(number.Int64()).String()
	}
	// It's negative and large, which is invalid.
	return fmt.Sprintf("<invalid %d>", number)
}

// protoHeaderToEthHeader rebuilds a full ethTypes.Header from the proto wire
// form. Returns nil on a nil or malformed input (e.g., out-of-bounds bloom) so
// callers can map it to ethereum.NotFound.
func protoHeaderToEthHeader(p *proto.Header) *ethTypes.Header {
	if p == nil {
		return nil
	}

	if len(p.Bloom) > ethTypes.BloomByteLength {
		return nil
	}

	if len(p.Difficulty) > 32 || len(p.BaseFee) > 32 {
		return nil
	}

	h := &ethTypes.Header{
		ParentHash:  protoH256ToHash(p.ParentHash),
		UncleHash:   protoH256ToHash(p.UncleHash),
		Coinbase:    protoH160ToAddress(p.Coinbase),
		Root:        protoH256ToHash(p.StateRoot),
		TxHash:      protoH256ToHash(p.TxRoot),
		ReceiptHash: protoH256ToHash(p.ReceiptRoot),
		Difficulty:  new(big.Int).SetBytes(p.Difficulty),
		Number:      new(big.Int).SetUint64(p.Number),
		GasLimit:    p.GasLimit,
		GasUsed:     p.GasUsed,
		Time:        p.Time,
		Extra:       append([]byte(nil), p.ExtraData...),
		MixDigest:   protoH256ToHash(p.MixDigest),
	}
	h.Bloom.SetBytes(p.Bloom)
	copy(h.Nonce[:], p.Nonce)

	if len(p.BaseFee) > 0 {
		h.BaseFee = new(big.Int).SetBytes(p.BaseFee)
	}
	if p.WithdrawalsHash != nil {
		v := protoH256ToHash(p.WithdrawalsHash)
		h.WithdrawalsHash = &v
	}
	// BlobGasUsed and ExcessBlobGas are proto3 `optional`.
	// We use *uint64 as a direct pointer, so the copy preserves nil vs. zero.
	h.BlobGasUsed = p.BlobGasUsed
	h.ExcessBlobGas = p.ExcessBlobGas
	if p.ParentBeaconBlockRoot != nil {
		v := protoH256ToHash(p.ParentBeaconBlockRoot)
		h.ParentBeaconRoot = &v
	}
	if p.RequestsHash != nil {
		v := protoH256ToHash(p.RequestsHash)
		h.RequestsHash = &v
	}
	return h
}

// protoH256ToHash converts a proto H256 (or nil) to a common.Hash.
func protoH256ToHash(h *commonproto.H256) common.Hash {
	if h == nil || h.Hi == nil || h.Lo == nil {
		return common.Hash{}
	}
	b := protoutil.ConvertH256ToHash(h)
	return common.BytesToHash(b[:])
}

// protoH160ToAddress converts a proto H160 (or nil) to a common.Address.
func protoH160ToAddress(a *commonproto.H160) common.Address {
	if a == nil || a.Hi == nil {
		return common.Address{}
	}
	arr := protoutil.ConvertH160toAddress(a)
	return common.BytesToAddress(arr[:])
}
