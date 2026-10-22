package helper

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/slashmanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statereceiver"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/contracts/validatorset"
	"github.com/0xPolygon/heimdall-v2/metrics"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	// smart contracts' events names
	NewHeaderBlockEvent = "NewHeaderBlock"
	TopUpFeeEvent       = "TopUpFee"
	StakedEvent         = "Staked"
	StakeUpdateEvent    = "StakeUpdate"
	UnstakeInitEvent    = "UnstakeInit"
	SignerChangeEvent   = "SignerChange"
	StateSyncedEvent    = "StateSynced"
	SlashedEvent        = "Slashed"
	UnJailedEvent       = "UnJailed"

	// error messages
	errUnableToConnect = "unable to connect to bor chain"
	errEventNotFound   = "event not found"
)

// IContractCaller represents contract caller
type IContractCaller interface {
	GetHeaderInfo(ctx context.Context, headerID uint64, rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (root common.Hash, start, end, createdAt uint64, proposer string, err error)
	GetRootHash(ctx context.Context, start, end, checkpointLength uint64) ([]byte, error)
	GetVoteOnHash(start, end uint64, hash, milestoneID string) (bool, error)
	GetValidatorInfo(valID uint64, stakingInfoInstance *stakinginfo.Stakinginfo) (validator types.Validator, err error)
	GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error)
	CurrentHeaderBlock(rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (uint64, error)
	GetBalance(address common.Address) (*big.Int, error)
	SendCheckpoint(signedData []byte, sigs [][3]*big.Int, rootChainAddress common.Address, rootChainInstance *rootchain.Rootchain) (err error)
	GetCheckpointSign(txHash common.Hash) ([]byte, []byte, []byte, error)
	GetMainChainBlock(ctx context.Context, blockNum *big.Int) (*ethTypes.Header, error)
	GetMainChainFinalizedBlock(ctx context.Context) (*ethTypes.Header, error)
	GetBorChainBlock(context.Context, *big.Int) (*ethTypes.Header, error)
	GetBorChainBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error)
	GetBorChainBlockTd(ctx context.Context, blockHash common.Hash) (uint64, error)
	GetBorChainBlockAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error)
	IsTxConfirmed(ctx context.Context, txHash common.Hash, requiredConfirmations uint64) bool
	GetConfirmedTxReceipt(ctx context.Context, txHash common.Hash, requiredConfirmations uint64) (*ethTypes.Receipt, error)
	GetBlockNumberFromTxHash(common.Hash) (*big.Int, error)

	DecodeNewHeaderBlockEvent(string, *ethTypes.Receipt, uint64) (*rootchain.RootchainNewHeaderBlock, error)

	DecodeValidatorTopupFeesEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoTopUpFee, error)
	DecodeValidatorJoinEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoStaked, error)
	DecodeValidatorStakeUpdateEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoStakeUpdate, error)
	DecodeValidatorExitEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoUnstakeInit, error)
	DecodeSignerUpdateEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoSignerChange, error)

	DecodeStateSyncedEvent(string, *ethTypes.Receipt, uint64) (*statesender.StatesenderStateSynced, error)

	DecodeSlashedEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoSlashed, error)
	DecodeUnJailedEvent(string, *ethTypes.Receipt, uint64) (*stakinginfo.StakinginfoUnJailed, error)

	GetMainTxReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
	GetBorTxReceipt(common.Hash) (*ethTypes.Receipt, error)
	ApproveTokens(*big.Int, common.Address, common.Address, *erc20.Erc20) error
	StakeFor(common.Address, *big.Int, *big.Int, bool, common.Address, *stakemanager.Stakemanager) error
	CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error)
	CurrentSpanNumber(validatorSet *validatorset.Validatorset) (Number *big.Int)
	GetSpanDetails(id *big.Int, validatorSet *validatorset.Validatorset) (*big.Int, *big.Int, *big.Int, error)
	CurrentStateCounter(stateSenderInstance *statesender.Statesender) (Number *big.Int)
	CheckIfBlocksExist(ctx context.Context, end uint64) (bool, error)
	GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error)
	GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error)
	GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error)
	GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error)
	GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error)
	GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error)
	GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error)
	GetTokenInstance(tokenAddress string) (*erc20.Erc20, error)
}

// BorGRPCClienter is the subset of *grpc.BorGRPCClient used by helper code.
// Declared as an interface so tests can inject fakes without dialing a real
// bor gRPC server. The concrete *grpc.BorGRPCClient satisfies this interface
// automatically.
type BorGRPCClienter interface {
	borReader
	GetTdByNumber(ctx context.Context, blockNum *big.Int) (uint64, error)
	BorBlockReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
}

// ContractCaller contract caller
type ContractCaller struct {
	MainChainClient    *ethclient.Client
	MainChainRPCClient *rpc.Client
	MainChainTimeout   time.Duration

	BorChainClient    *ethclient.Client
	BorChainRPCClient *rpc.Client
	BorChainTimeout   time.Duration

	BorChainGrpcFlag   bool
	BorChainGrpcClient BorGRPCClienter

	RootChainABI     abi.ABI
	StakingInfoABI   abi.ABI
	ValidatorSetABI  abi.ABI
	StateReceiverABI abi.ABI
	StateSenderABI   abi.ABI
	StakeManagerABI  abi.ABI
	SlashManagerABI  abi.ABI
	PolTokenABI      abi.ABI

	contractInstanceCache map[contractInstanceKey]interface{}
	contractInstanceMu    *sync.RWMutex

	// prefetchMu protects the round-scoped prefetch state used by ExtendVote.
	prefetchMu *sync.RWMutex
	// prefetchedReceipts stores the prefetched L1 tx receipts from ExtendVoteHandler.
	// Reset after each round of ExtendVoteHandler.
	prefetchedReceipts map[common.Hash]*ethTypes.Receipt
	// finalizedHeaderCache stores the last fetched finalized main chain block header.
	// Reset after each round of ExtendVoteHandler.
	finalizedHeaderCache *ethTypes.Header
}

// contractInstanceKey keys cached bindings by type as well as address, so two
// binding types configured at the same address can't collide.
type contractInstanceKey struct {
	address common.Address
	binding reflect.Type
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

type rpcTransaction struct {
	txExtraInfo
}

// NewContractCaller contract caller
func NewContractCaller() (*ContractCaller, error) {
	contractCallerObj := &ContractCaller{}
	config := GetConfig()
	contractCallerObj.MainChainClient = GetMainClient()
	contractCallerObj.MainChainTimeout = config.EthRPCTimeout
	contractCallerObj.BorChainClient = GetBorClient()
	contractCallerObj.BorChainTimeout = GetBorChainCallTimeout()
	contractCallerObj.MainChainRPCClient = GetMainChainRPCClient()
	contractCallerObj.BorChainRPCClient = GetBorRPCClient()
	contractCallerObj.BorChainGrpcFlag = config.BorGRPCFlag
	if client := GetBorGRPCClient(); client != nil {
		contractCallerObj.BorChainGrpcClient = client
	}

	contractCallerObj.contractInstanceCache = make(map[contractInstanceKey]interface{})
	contractCallerObj.contractInstanceMu = &sync.RWMutex{}

	contractCallerObj.prefetchMu = &sync.RWMutex{}
	contractCallerObj.prefetchedReceipts = make(map[common.Hash]*ethTypes.Receipt)
	contractCallerObj.finalizedHeaderCache = nil

	if err := populateABIs(contractCallerObj); err != nil {
		return nil, err
	}

	return contractCallerObj, nil
}

// cachedBinding returns the binding of type T for address, constructing it
// against backend and caching it on first use.
func cachedBinding[T any](c *ContractCaller, address string, backend bind.ContractBackend, newBinding func(common.Address, bind.ContractBackend) (*T, error)) (*T, error) {
	key := contractInstanceKey{address: common.HexToAddress(address), binding: reflect.TypeFor[T]()}

	c.contractInstanceMu.RLock()
	cached, ok := c.contractInstanceCache[key]
	c.contractInstanceMu.RUnlock()
	if ok {
		return cached.(*T), nil
	}

	instance, err := newBinding(key.address, backend)
	if err != nil {
		Logger.Error("Error creating contract binding", "binding", key.binding.String(), "address", key.address, "error", err)
		return nil, err
	}

	c.contractInstanceMu.Lock()
	c.contractInstanceCache[key] = instance
	c.contractInstanceMu.Unlock()

	return instance, nil
}

// GetRootChainInstance returns the RootChain contract instance for a selected chain
func (c *ContractCaller) GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error) {
	return cachedBinding(c, rootChainAddress, mainChainClient, rootchain.NewRootchain)
}

// GetStakingInfoInstance returns the stakingInfo contract instance for a selected chain
func (c *ContractCaller) GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error) {
	return cachedBinding(c, stakingInfoAddress, mainChainClient, stakinginfo.NewStakinginfo)
}

// GetValidatorSetInstance returns the validatorSet contract instance for a selected chain
func (c *ContractCaller) GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error) {
	return cachedBinding(c, validatorSetAddress, mainChainClient, validatorset.NewValidatorset)
}

// GetStakeManagerInstance returns the stakeManager contract instance for a selected chain
func (c *ContractCaller) GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error) {
	return cachedBinding(c, stakingManagerAddress, mainChainClient, stakemanager.NewStakemanager)
}

// GetSlashManagerInstance returns the slashManager contract instance for a selected chain
func (c *ContractCaller) GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error) {
	return cachedBinding(c, slashManagerAddress, mainChainClient, slashmanager.NewSlashmanager)
}

// GetStateSenderInstance returns the stateSender contract instance for a selected chain
func (c *ContractCaller) GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error) {
	return cachedBinding(c, stateSenderAddress, mainChainClient, statesender.NewStatesender)
}

// GetStateReceiverInstance returns the stateReceiver contract instance for a selected chain
func (c *ContractCaller) GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error) {
	return cachedBinding(c, stateReceiverAddress, borClient, statereceiver.NewStatereceiver)
}

// GetTokenInstance returns the token contract instance for a selected chain
func (c *ContractCaller) GetTokenInstance(tokenAddress string) (*erc20.Erc20, error) {
	return cachedBinding(c, tokenAddress, mainChainClient, erc20.NewErc20)
}

// GetHeaderInfo get header info from the checkpoint number
func (c *ContractCaller) GetHeaderInfo(ctx context.Context, headerID uint64, rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (
	root common.Hash,
	start,
	end,
	createdAt uint64,
	proposer string,
	err error,
) {
	checkpointBigInt := big.NewInt(0).Mul(big.NewInt(0).SetUint64(headerID), big.NewInt(0).SetUint64(childBlockInterval))

	callCtx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()
	headerBlock, err := rootChainInstance.HeaderBlocks(&bind.CallOpts{Context: callCtx}, checkpointBigInt)
	if err != nil {
		return root, start, end, createdAt, proposer, errors.New("unable to fetch checkpoint block")
	}

	return headerBlock.Root,
		headerBlock.Start.Uint64(),
		headerBlock.End.Uint64(),
		headerBlock.CreatedAt.Uint64(),
		headerBlock.Proposer.String(),
		nil
}

// GetRootHash get root hash from the bor chain for the corresponding start and end block
func (c *ContractCaller) GetRootHash(ctx context.Context, start, end, checkpointLength uint64) ([]byte, error) {
	noOfBlock := end - start + 1

	if start > end {
		return nil, errors.New("start is greater than end")
	}

	if noOfBlock > checkpointLength {
		return nil, errors.New("number of headers requested exceeds checkpoint length")
	}

	callCtx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return nil, err
	}

	rootHash, err := bor.GetRootHash(callCtx, start, end)
	if err != nil {
		Logger.Error("Could not fetch rootHash from bor chain", "error", err)
		return nil, err
	}

	decoded := common.FromHex(rootHash)
	if len(decoded) != common.HashLength {
		return nil, fmt.Errorf("bor rootHash: expected %d bytes, got %d", common.HashLength, len(decoded))
	}
	if (common.BytesToHash(decoded) == common.Hash{}) {
		return nil, errors.New("bor rootHash: zero value")
	}

	return decoded, nil
}

// GetVoteOnHash get vote on hash from the bor chain for the corresponding milestone
func (c *ContractCaller) GetVoteOnHash(start, end uint64, hash, milestoneID string) (bool, error) {
	if start > end {
		return false, errors.New("Start block number is greater than the end block number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return false, err
	}

	vote, err := bor.GetVoteOnHash(ctx, start, end, hash, milestoneID)
	if err != nil {
		return false, errors.New(fmt.Sprint("Error in fetching vote from bor chain", "err", err))
	}

	return vote, nil
}

// GetLastChildBlock fetch current child block
func (c *ContractCaller) GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	lastChildBlock, err := rootChainInstance.GetLastChildBlock(opts)
	if err != nil {
		Logger.Error("Could not fetch current child block from rootChain contract", "error", err)
		return 0, err
	}

	if lastChildBlock == nil {
		Logger.Error("Contract returned nil value for lastChildBlock")
		return 0, fmt.Errorf("contract returned nil value")
	}

	return lastChildBlock.Uint64(), nil
}

// CurrentHeaderBlock fetches the current header block
func (c *ContractCaller) CurrentHeaderBlock(rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (uint64, error) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	currentHeaderBlock, err := rootChainInstance.CurrentHeaderBlock(opts)
	if err != nil {
		Logger.Error("Could not fetch current header block from rootChain contract", "error", err)
		return 0, err
	}

	if currentHeaderBlock == nil {
		Logger.Error("Contract returned nil value for currentHeaderBlock")
		return 0, fmt.Errorf("contract returned nil value")
	}

	return currentHeaderBlock.Uint64() / childBlockInterval, nil
}

// GetBalance get balance of an account (returns big.Int balance won't fit in uint64)
func (c *ContractCaller) GetBalance(address common.Address) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	balance, err := c.MainChainClient.BalanceAt(ctx, address, nil)
	if err != nil {
		Logger.Error("Unable to fetch balance of account from root chain", "Address", address.String(), "error", err)
		return big.NewInt(0), err
	}

	return balance, nil
}

// GetValidatorInfo get validator info
func (c *ContractCaller) GetValidatorInfo(valID uint64, stakingInfoInstance *stakinginfo.Stakinginfo) (validator types.Validator, err error) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	stakerDetails, err := stakingInfoInstance.GetStakerDetails(opts, big.NewInt(int64(valID)))
	if err != nil {
		Logger.Error("Error fetching validator information from stake manager", "validatorId", valID, "error", err)
		return
	}

	newAmount, err := GetPowerFromAmount(stakerDetails.Amount)
	if err != nil {
		return
	}

	// newAmount
	validator = types.Validator{
		ValId:       valID,
		VotingPower: newAmount.Int64(),
		StartEpoch:  stakerDetails.ActivationEpoch.Uint64(),
		EndEpoch:    stakerDetails.DeactivationEpoch.Uint64(),
		Signer:      stakerDetails.Signer.String(),
	}

	return validator, nil
}

// GetMainChainBlock returns main chain block header
func (c *ContractCaller) GetMainChainBlock(ctx context.Context, blockNum *big.Int) (header *ethTypes.Header, err error) {
	callCtx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()

	latestBlock, err := c.MainChainClient.HeaderByNumber(callCtx, blockNum)
	if err != nil {
		Logger.Error("Unable to connect to main chain", "error", err)
		return
	}

	return latestBlock, nil
}

// GetMainChainFinalizedBlock returns the finalized main chain block header (post-merge)
func (c *ContractCaller) GetMainChainFinalizedBlock(ctx context.Context) (header *ethTypes.Header, err error) {
	callCtx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()

	latestFinalizedBlock, err := c.MainChainClient.HeaderByNumber(callCtx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		Logger.Error(errUnableToConnect, "error", err)
		return
	}

	return latestFinalizedBlock, nil
}

// getOrFetchReceipt returns a receipt from prefetched receipts or fetches from L1.
func (c *ContractCaller) getOrFetchReceipt(ctx context.Context, tx common.Hash) (*ethTypes.Receipt, error) {
	c.prefetchMu.RLock()
	var cachedReceipt *ethTypes.Receipt
	if c.prefetchedReceipts != nil {
		cachedReceipt = c.prefetchedReceipts[tx]
	}
	c.prefetchMu.RUnlock()

	if cachedReceipt != nil {
		Logger.Debug("Receipt found in prefetched receipts", "tx", tx.Hex())
		return cachedReceipt, nil
	}

	Logger.Debug("Fetching the receipt from the main chain", "tx", tx.Hex())

	receipt, err := c.GetMainTxReceipt(ctx, tx)
	if err != nil {
		Logger.Error("Error while fetching receipt from ethereum", "txHash", tx.Hex(), "error", err)
		return nil, err
	}

	if receipt == nil {
		Logger.Error("Tx receipt not found on ethereum chain", "txHash", tx.Hex())
		return nil, errors.New("ethereum tx receipt not found")
	}

	return receipt, nil
}

// GetMainChainBlockTime returns main chain block time
func (c *ContractCaller) GetMainChainBlockTime(ctx context.Context, blockNum uint64) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()

	latestBlock, err := c.MainChainClient.BlockByNumber(ctx, big.NewInt(0).SetUint64(blockNum))
	if err != nil {
		Logger.Error(errUnableToConnect, "error", err)
		return time.Time{}, err
	}

	return time.Unix(int64(latestBlock.Time()), 0), nil
}

// GetBorChainBlock returns bor chain block header
func (c *ContractCaller) GetBorChainBlock(ctx context.Context, blockNum *big.Int) (header *ethTypes.Header, err error) {
	defer metrics.RecordBorRPCCallDuration("get_bor_chain_block", time.Now())

	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		Logger.Error(errUnableToConnect, "error", err)
		return nil, err
	}

	// LatestBlockNumber is BlockNumber(-2) in go-ethereum rpc
	blockID := int64(rpc.LatestBlockNumber)
	if blockNum != nil {
		blockID = blockNum.Int64()
	}

	latestBlock, err := bor.HeaderByNumber(ctx, blockID)
	if err != nil {
		// both HTTP and gRPC map a missing block to ethereum.NotFound.
		if !errors.Is(err, ethereum.NotFound) {
			Logger.Error(errUnableToConnect, "error", err)
		}
		return
	}

	return latestBlock, nil
}

// GetBorChainBlockInfoInBatch returns bor chain block headers, total difficulties,
// and authors for the inclusive range [start, end]. It dispatches to gRPC when
// BorChainGrpcFlag is set, otherwise to the HTTP JSON-RPC batch.
// In both paths, it tries to get blocks from the range interval
// but returns only the ones found on the chain.
func (c *ContractCaller) GetBorChainBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	defer metrics.RecordBorRPCCallDuration("get_bor_chain_block_info_in_batch", time.Now())

	if start < 0 || end < 0 || end < start {
		return nil, nil, nil, fmt.Errorf("invalid range [%d,%d]", start, end)
	}
	// Prevents int64 (end-start+1) overflow
	if end-start > borgrpc.MaxBlockInfoBatchSize-1 {
		return nil, nil, nil, fmt.Errorf("range too large: %d blocks exceeds max %d", end-start+1, borgrpc.MaxBlockInfoBatchSize)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return nil, nil, nil, err
	}

	return bor.GetBlockInfoInBatch(timeoutCtx, start, end)
}

// tdResp is the JSON shape returned by eth_getTdByNumber.
type tdResp struct {
	TotalDifficulty hexutil.Uint64 `json:"totalDifficulty"`
}

// buildBorBatchElems constructs the flat BatchElem slice for a single BatchCallContext call.
func buildBorBatchElems(start, end int64, hdrOut []*ethTypes.Header, tdOut []*tdResp, authorOut []*common.Address) []rpc.BatchElem {
	totalBlocks := end - start + 1
	// The 2* here is a capacity hint; mutating it doesn't break correctness because append grows the slice.
	// mutator-disable-next-line slice-capacity hint only
	elems := make([]rpc.BatchElem, 0, 2*totalBlocks)

	for i := start; i <= end; i++ {
		blockNumHex := fmt.Sprintf("0x%x", i)
		elems = append(elems, rpc.BatchElem{Method: "eth_getHeaderByNumber", Args: []interface{}{blockNumHex}, Result: &hdrOut[i-start]})
	}

	for i := start; i <= end; i++ {
		blockNumHex := fmt.Sprintf("0x%x", i)
		elems = append(elems, rpc.BatchElem{Method: "eth_getTdByNumber", Args: []interface{}{blockNumHex}, Result: &tdOut[i-start]})
	}

	for i := start; i <= end; i++ {
		if i > 0 { // skip genesis block
			blockNumHex := fmt.Sprintf("0x%x", i)
			elems = append(elems, rpc.BatchElem{Method: "bor_getAuthor", Args: []interface{}{blockNumHex}, Result: &authorOut[i-start]})
		}
	}

	return elems
}

// borAuthorFromBatch retrieves the author address for a non-genesis block from the flat
// batchElems slice. It returns the address and true on success, or the zero address and
// false when the batch entry indicates an error or a nil result.
func borAuthorFromBatch(i int, start, totalBlocks int64, batchElems []rpc.BatchElem, authors []*common.Address) (common.Address, bool) {
	authorReqIndex := 2*int(totalBlocks) + i
	if start == 0 {
		// genesis block has no author entry in the batch, so all later indices shift left by 1.
		authorReqIndex--
	}
	elem := batchElems[authorReqIndex]
	if elem.Error != nil || authors[i] == nil {
		return common.Address{}, false
	}
	return *authors[i], true
}

// collateBorBatchResults walks the flat BatchElem result slice and collects the contiguous
// prefix of successfully fetched blocks, stopping at the first error or missing result.
func collateBorBatchResults(start, totalBlocks int64, batchElems []rpc.BatchElem, hdrs []*ethTypes.Header, tds []*tdResp, authors []*common.Address) ([]*ethTypes.Header, []uint64, []common.Address) {
	headers := make([]*ethTypes.Header, 0, totalBlocks)
	tdSlice := make([]uint64, 0, totalBlocks)
	authorSlice := make([]common.Address, 0, totalBlocks)

	for i := 0; i < int(totalBlocks); i++ {
		blockNum := start + int64(i)
		elemHeader := batchElems[i]
		elemTd := batchElems[i+int(totalBlocks)]

		if elemHeader.Error != nil || elemTd.Error != nil || hdrs[i] == nil || tds[i] == nil {
			Logger.Debug("Error fetching block info", "headerErr", elemHeader.Error, "tdErr", elemTd.Error, "blockNum", blockNum)
			break
		}
		// Verify the returned header's block number matches the requested slot
		// to avoid potential wrong hashes into downstream milestone propositions.
		if hdrs[i].Number == nil || hdrs[i].Number.Uint64() != uint64(blockNum) {
			Logger.Debug("bor batch returned header with mismatched block number", "want", blockNum, "got", hdrs[i].Number)
			break
		}

		var author common.Address
		if blockNum > 0 {
			var ok bool
			author, ok = borAuthorFromBatch(i, start, totalBlocks, batchElems, authors)
			if !ok {
				// statement_deletion only drops a debug message, the break still stops the loop.
				// mutator-disable-next-line operator-log line
				Logger.Debug("Error fetching block author", "blockNum", blockNum)
				break
			}
		}

		headers = append(headers, hdrs[i])
		tdSlice = append(tdSlice, uint64(tds[i].TotalDifficulty))
		authorSlice = append(authorSlice, author)
	}

	return headers, tdSlice, authorSlice
}

// GetBlockInfoInBatch is the HTTP/JSON-RPC implementation of GetBorChainBlockInfoInBatch.
// It issues a single RPC batch call covering headers, total difficulties, and authors for the
// inclusive range [start, end], and returns only the contiguous prefix of blocks found on the chain.
func (h borHTTPClient) GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	// Range arithmetic compensated by the per-index bounds inside buildBorBatchElems/collateBorBatchResults.
	// mutator-disable-next-line range-arithmetic compensated downstream
	totalBlocks := end - start + 1
	rpcClient := h.client.Client()

	headerResults := make([]*ethTypes.Header, totalBlocks)
	tdResults := make([]*tdResp, totalBlocks)
	authorResults := make([]*common.Address, totalBlocks)

	batchElems := buildBorBatchElems(start, end, headerResults, tdResults, authorResults)

	// negate_conditional/branch_removal require a test that induces a transport-level batch failure mid-call.
	// mutator-disable-next-line defensive BatchCallContext error guard
	if err := rpcClient.BatchCallContext(ctx, batchElems); err != nil {
		// implementation-detail: nil return on batch error; callers check err!=nil
		// mutator-disable-next-line return-value on error propagation
		return nil, nil, nil, err
	}

	headers, tds, authors := collateBorBatchResults(start, totalBlocks, batchElems, headerResults, tdResults, authorResults)
	return headers, tds, authors, nil
}

// GetBorChainBlockTd returns total difficulty of a block
func (c *ContractCaller) GetBorChainBlockTd(ctx context.Context, blockHash common.Hash) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return 0, err
	}

	return bor.GetTdByHash(ctx, blockHash)
}

// GetBorChainBlockAuthor returns the producer of the bor block
func (c *ContractCaller) GetBorChainBlockAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		// the return on the next line is what matters; log deletion is observable only in ops.
		// mutator-disable-next-line operator-log line
		Logger.Error(errUnableToConnect, "error", err)
		return nil, err
	}

	author, err := bor.GetAuthor(ctx, blockNum)
	if err != nil {
		if !errors.Is(err, ethereum.NotFound) {
			Logger.Error(errUnableToConnect, "error", err)
		}
		return nil, err
	}
	if author == nil {
		return nil, ethereum.NotFound
	}
	return author, nil
}

// GetBlockNumberFromTxHash gets the block number of transaction
func (c *ContractCaller) GetBlockNumberFromTxHash(tx common.Hash) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	var rpcTx rpcTransaction
	if err := c.MainChainRPCClient.CallContext(ctx, &rpcTx, "eth_getTransactionByHash", tx); err != nil {
		return nil, err
	}

	if rpcTx.BlockNumber == nil {
		return nil, errors.New("no tx found")
	}

	blkNum := big.NewInt(0)

	blkNum, ok := blkNum.SetString(*rpcTx.BlockNumber, 0)
	if !ok {
		return nil, errors.New("unable to set string")
	}

	return blkNum, nil
}

// IsTxConfirmed checks whether the tx corresponding to the given hash is confirmed with given
// requiredConfirmations numbers
func (c *ContractCaller) IsTxConfirmed(ctx context.Context, txHash common.Hash, requiredConfirmations uint64) bool {
	receipt, err := c.GetConfirmedTxReceipt(ctx, txHash, requiredConfirmations)
	if err != nil {
		Logger.Error("Error while fetching the tx receipt", "error", err)
		return false
	}

	return receipt != nil
}

// GetConfirmedTxReceipt returns a tx receipt only if it is finalized (or has the required confirmations).
func (c *ContractCaller) GetConfirmedTxReceipt(ctx context.Context, tx common.Hash, requiredConfirmations uint64) (*ethTypes.Receipt, error) {
	receipt, err := c.getOrFetchReceipt(ctx, tx)
	if err != nil {
		return nil, err
	}

	if receipt.BlockNumber == nil {
		return nil, errors.New("receipt has nil block number")
	}

	receiptBlockNumber := receipt.BlockNumber.Uint64()

	c.prefetchMu.RLock()
	cachedFinalizedHeader := c.finalizedHeaderCache
	c.prefetchMu.RUnlock()

	if cachedFinalizedHeader != nil && cachedFinalizedHeader.Number != nil {
		if receiptBlockNumber <= cachedFinalizedHeader.Number.Uint64() {
			return receipt, nil
		}
	}

	latestFinalizedBlock, err := c.GetMainChainFinalizedBlock(ctx)
	if err != nil {
		Logger.Error("Error getting latest finalized main chain block", "error", err)
	}

	if latestFinalizedBlock != nil && latestFinalizedBlock.Number != nil {
		Logger.Debug("Fetched latest finalized main chain block",
			"blockNumber", latestFinalizedBlock.Number.Uint64(),
		)
		c.prefetchMu.Lock()
		c.finalizedHeaderCache = latestFinalizedBlock
		c.prefetchMu.Unlock()

		if receiptBlockNumber > latestFinalizedBlock.Number.Uint64() {
			return nil, errors.New("receipt block number is ahead of latest finalized main chain block")
		}

		return receipt, nil
	}

	// No finalized API: fall back to N confirmations.
	latestBlock, err := c.GetMainChainBlock(ctx, nil)
	if err != nil {
		Logger.Error("Error getting latest main chain block", "error", err)
		return nil, err
	}
	if latestBlock == nil || latestBlock.Number == nil {
		return nil, errors.New("latest main chain block header or number is nil")
	}
	Logger.Debug("Fetched latest main chain block",
		"blockNumber", latestBlock.Number.Uint64(),
	)

	latestNum := latestBlock.Number.Uint64()
	if latestNum < receiptBlockNumber {
		return nil, errors.New("receipt block number is ahead of latest main chain block")
	}

	diff := latestNum - receiptBlockNumber
	if diff < requiredConfirmations {
		return nil, errors.New("not enough confirmations")
	}

	return receipt, nil
}

// decodeLog unpacks the log at logIndex in receipt as eventName, accepting it
// only if it was emitted by contractAddressString.
func decodeLog[T any](contractABI *abi.ABI, eventName, contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*T, error) {
	event := new(T)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(contractABI, event, eventName, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeNewHeaderBlockEvent represents the new header block event
func (c *ContractCaller) DecodeNewHeaderBlockEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*rootchain.RootchainNewHeaderBlock, error) {
	return decodeLog[rootchain.RootchainNewHeaderBlock](&c.RootChainABI, NewHeaderBlockEvent, contractAddressString, receipt, logIndex)
}

// DecodeValidatorTopupFeesEvent represents topUp for fees tokens
func (c *ContractCaller) DecodeValidatorTopupFeesEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoTopUpFee, error) {
	return decodeLog[stakinginfo.StakinginfoTopUpFee](&c.StakingInfoABI, TopUpFeeEvent, contractAddressString, receipt, logIndex)
}

// DecodeValidatorJoinEvent represents validator staked event
func (c *ContractCaller) DecodeValidatorJoinEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoStaked, error) {
	return decodeLog[stakinginfo.StakinginfoStaked](&c.StakingInfoABI, StakedEvent, contractAddressString, receipt, logIndex)
}

// DecodeValidatorStakeUpdateEvent represents validator stake update event
func (c *ContractCaller) DecodeValidatorStakeUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoStakeUpdate, error) {
	return decodeLog[stakinginfo.StakinginfoStakeUpdate](&c.StakingInfoABI, StakeUpdateEvent, contractAddressString, receipt, logIndex)
}

// DecodeValidatorExitEvent represents validator stake unStake event
func (c *ContractCaller) DecodeValidatorExitEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnstakeInit, error) {
	return decodeLog[stakinginfo.StakinginfoUnstakeInit](&c.StakingInfoABI, UnstakeInitEvent, contractAddressString, receipt, logIndex)
}

// DecodeSignerUpdateEvent represents sig update event
func (c *ContractCaller) DecodeSignerUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSignerChange, error) {
	return decodeLog[stakinginfo.StakinginfoSignerChange](&c.StakingInfoABI, SignerChangeEvent, contractAddressString, receipt, logIndex)
}

// DecodeStateSyncedEvent decode state sync data
func (c *ContractCaller) DecodeStateSyncedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*statesender.StatesenderStateSynced, error) {
	return decodeLog[statesender.StatesenderStateSynced](&c.StateSenderABI, StateSyncedEvent, contractAddressString, receipt, logIndex)
}

// decode slashing events

// DecodeSlashedEvent represents tick ack on contract
func (c *ContractCaller) DecodeSlashedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSlashed, error) {
	return decodeLog[stakinginfo.StakinginfoSlashed](&c.StakingInfoABI, SlashedEvent, contractAddressString, receipt, logIndex)
}

// DecodeUnJailedEvent represents unJail on contract
func (c *ContractCaller) DecodeUnJailedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnJailed, error) {
	return decodeLog[stakinginfo.StakinginfoUnJailed](&c.StakingInfoABI, UnJailedEvent, contractAddressString, receipt, logIndex)
}

//
// Account root functions
//

// CurrentAccountStateRoot get current account root from on the chain
func (c *ContractCaller) CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	accountStateRoot, err := stakingInfoInstance.GetAccountStateRoot(opts)
	if err != nil {
		Logger.Error("Unable to get current account state root", "error", err)

		var emptyArr [32]byte

		return emptyArr, err
	}

	return accountStateRoot, nil
}

//
// Span-related functions
//

// CurrentSpanNumber get current span
func (c *ContractCaller) CurrentSpanNumber(validatorSetInstance *validatorset.Validatorset) (Number *big.Int) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	result, err := validatorSetInstance.CurrentSpanNumber(opts)
	if err != nil {
		Logger.Error("Unable to get current span number", "error", err)
		return nil
	}

	return result
}

// GetSpanDetails get span details
func (c *ContractCaller) GetSpanDetails(id *big.Int, validatorSetInstance *validatorset.Validatorset) (
	*big.Int,
	*big.Int,
	*big.Int,
	error,
) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	d, err := validatorSetInstance.GetSpan(opts, id)
	if err != nil {
		return nil, nil, nil, errors.New("unable to get span details")
	}
	return d.Number, d.StartBlock, d.EndBlock, nil
}

// CurrentStateCounter get state counter
func (c *ContractCaller) CurrentStateCounter(stateSenderInstance *statesender.Statesender) (Number *big.Int) {
	opts, cancel := c.mainChainCallOpts()
	defer cancel()

	result, err := stateSenderInstance.Counter(opts)
	if err != nil {
		Logger.Error("Unable to get current counter number", "error", err)
		return nil
	}

	return result
}

// CheckIfBlocksExist - check if the given block number exists on the local chain.
// Here we check if the block number exists by fetching the header from the bor chain.
func (c *ContractCaller) CheckIfBlocksExist(ctx context.Context, number uint64) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return false, err
	}

	header, err := bor.HeaderByNumber(callCtx, int64(number))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return false, nil
		}
		return false, err
	}
	if header == nil || header.Number == nil {
		return false, nil
	}

	return number == header.Number.Uint64(), nil
}

// GetBlockByNumber returns blocks by number from the child chain (bor)
func (c *ContractCaller) GetBlockByNumber(ctx context.Context, blockNumber uint64) (*ethTypes.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return nil, err
	}

	block, err := bor.BlockByNumber(ctx, int64(blockNumber))
	if err != nil {
		Logger.Error("Unable to fetch block by number from child chain", "block", block, "err", err)
		return nil, err
	}

	return block, nil
}

//
// Receipt functions
//

// GetMainTxReceipt returns main tx receipt
func (c *ContractCaller) GetMainTxReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()

	return c.MainChainClient.TransactionReceipt(callCtx, txHash)
}

// BatchGetMainChainTxReceipts fetches multiple main chain tx receipts in a single JSON-RPC batch call.
// Returns a map of txHash → receipt. Failed individual requests are skipped.
func (c *ContractCaller) BatchGetMainChainTxReceipts(ctx context.Context, txHashes []common.Hash) map[common.Hash]*ethTypes.Receipt {
	if len(txHashes) == 0 || c.MainChainRPCClient == nil {
		return nil
	}

	batch := make([]rpc.BatchElem, len(txHashes))
	for i, hash := range txHashes {
		batch[i] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{hash},
			Result: new(ethTypes.Receipt),
		}
	}

	if err := c.MainChainRPCClient.BatchCallContext(ctx, batch); err != nil {
		Logger.Error("Batch receipt fetch failed", "error", err)
		return nil
	}

	results := make(map[common.Hash]*ethTypes.Receipt, len(txHashes))
	for i, elem := range batch {
		if elem.Error != nil {
			continue
		}
		if receipt, ok := elem.Result.(*ethTypes.Receipt); ok && receipt != nil && receipt.BlockNumber != nil {
			results[txHashes[i]] = receipt
		}
	}

	return results
}

// GetBorTxReceipt returns bor tx receipt
func (c *ContractCaller) GetBorTxReceipt(txHash common.Hash) (*ethTypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.BorChainTimeout)
	defer cancel()

	bor, err := c.borClient()
	if err != nil {
		return nil, err
	}

	return bor.TransactionReceipt(ctx, txHash)
}

// GetCheckpointSign returns sigs input of committed checkpoint transaction
func (c *ContractCaller) GetCheckpointSign(txHash common.Hash) ([]byte, []byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	mainChainClient := GetMainClient()

	transaction, isPending, err := mainChainClient.TransactionByHash(ctx, txHash)
	if err != nil {
		Logger.Error("Error while fetching transaction by hash from MainChain", "error", err)
		return []byte{}, []byte{}, []byte{}, err
	} else if isPending {
		return []byte{}, []byte{}, []byte{}, errors.New("transaction is still pending")
	}

	payload := transaction.Data()
	chainABI := c.RootChainABI

	return UnpackSigAndVotes(payload, chainABI)
}

// borReader is the set of Bor queries the contract caller makes. The Bor gRPC
// client implements it directly; borHTTPClient implements it over JSON-RPC.
type borReader interface {
	HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error)
	BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error)
	GetRootHash(ctx context.Context, startBlock uint64, endBlock uint64) (string, error)
	GetVoteOnHash(ctx context.Context, startBlock uint64, endBlock uint64, rootHash string, milestoneId string) (bool, error)
	GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error)
	GetTdByHash(ctx context.Context, hash common.Hash) (uint64, error)
	GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
}

// borClient returns the Bor transport selected by BorChainGrpcFlag.
func (c *ContractCaller) borClient() (borReader, error) {
	if !c.BorChainGrpcFlag {
		return borHTTPClient{client: c.BorChainClient}, nil
	}
	if c.BorChainGrpcClient == nil {
		return nil, errors.New("bor grpc client is nil while bor grpc flag is enabled")
	}
	return c.BorChainGrpcClient, nil
}

// borHTTPClient serves borReader over Bor's JSON-RPC endpoint.
type borHTTPClient struct {
	client *ethclient.Client
}

func (h borHTTPClient) HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error) {
	return h.client.HeaderByNumber(ctx, big.NewInt(blockID))
}

func (h borHTTPClient) BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error) {
	return h.client.BlockByNumber(ctx, big.NewInt(blockID))
}

func (h borHTTPClient) GetRootHash(ctx context.Context, startBlock uint64, endBlock uint64) (string, error) {
	return h.client.GetRootHash(ctx, startBlock, endBlock)
}

func (h borHTTPClient) GetVoteOnHash(ctx context.Context, startBlock uint64, endBlock uint64, rootHash string, milestoneId string) (bool, error) {
	return h.client.GetVoteOnHash(ctx, startBlock, endBlock, rootHash, milestoneId)
}

func (h borHTTPClient) GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	var author *common.Address
	if err := h.client.Client().CallContext(ctx, &author, "bor_getAuthor", borgrpc.ToBlockNumArg(blockNum)); err != nil {
		return nil, err
	}
	return author, nil
}

func (h borHTTPClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	return h.client.TransactionReceipt(ctx, txHash)
}

// GetTdByHash returns the total difficulty of the block with the given hash.
func (h borHTTPClient) GetTdByHash(ctx context.Context, blockHash common.Hash) (uint64, error) {
	rpcClient := h.client.Client()

	var resp map[string]interface{}
	if err := rpcClient.CallContext(ctx, &resp, "eth_getTdByHash", blockHash.Hex()); err != nil {
		return 0, err
	}
	// A missing block surfaces as ethereum.NotFound, matching the gRPC transport
	if resp == nil || resp["totalDifficulty"] == nil {
		return 0, ethereum.NotFound
	}

	raw, ok := resp["totalDifficulty"].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected totalDifficulty type %T", resp["totalDifficulty"])
	}

	td, err := hexutil.DecodeUint64(raw)
	if err != nil {
		return 0, fmt.Errorf("failed to decode totalDifficulty %q: %w", raw, err)
	}

	return td, nil
}

// utility and helper methods

// mainChainCallOpts bounds a main chain contract view call by MainChainTimeout.
func (c *ContractCaller) mainChainCallOpts() (*bind.CallOpts, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	return &bind.CallOpts{Context: ctx}, cancel
}

// populateABIs copies each contract's ABI onto the caller. The generated
// bindings parse their ABI once and cache it, so no JSON is decoded per caller.
func populateABIs(contractCallerObj *ContractCaller) error {
	abis := []struct {
		dst      *abi.ABI
		metaData *bind.MetaData
	}{
		{&contractCallerObj.RootChainABI, rootchain.RootchainMetaData},
		{&contractCallerObj.StakingInfoABI, stakinginfo.StakinginfoMetaData},
		{&contractCallerObj.ValidatorSetABI, validatorset.ValidatorsetMetaData},
		{&contractCallerObj.StateReceiverABI, statereceiver.StatereceiverMetaData},
		{&contractCallerObj.StateSenderABI, statesender.StatesenderMetaData},
		{&contractCallerObj.StakeManagerABI, stakemanager.StakemanagerMetaData},
		{&contractCallerObj.SlashManagerABI, slashmanager.SlashmanagerMetaData},
		{&contractCallerObj.PolTokenABI, erc20.Erc20MetaData},
	}

	for _, a := range abis {
		parsed, err := a.metaData.GetAbi()
		if err != nil {
			Logger.Error("Error while parsing contract ABI", "error", err)
			return err
		}
		*a.dst = *parsed
	}

	return nil
}

// BeginPrefetchRound starts a new round of prefetch lifecycle for ExtendVote.
func (c *ContractCaller) BeginPrefetchRound() {
	c.prefetchMu.Lock()
	defer c.prefetchMu.Unlock()

	c.prefetchedReceipts = make(map[common.Hash]*ethTypes.Receipt)
	c.finalizedHeaderCache = nil
}

// EndPrefetchRound clears the round of prefetch lifecycle for ExtendVote.
func (c *ContractCaller) EndPrefetchRound() {
	c.prefetchMu.Lock()
	defer c.prefetchMu.Unlock()

	c.prefetchedReceipts = make(map[common.Hash]*ethTypes.Receipt)
	c.finalizedHeaderCache = nil
}
