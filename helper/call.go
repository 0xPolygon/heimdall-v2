package helper

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
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

// ContractsABIsMap is a cached map holding the ABIs of the contracts
var ContractsABIsMap = make(map[string]*abi.ABI)

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
	HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error)
	BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error)
	GetRootHash(ctx context.Context, startBlock uint64, endBlock uint64) (string, error)
	GetVoteOnHash(ctx context.Context, startBlock uint64, endBlock uint64, rootHash string, milestoneId string) (bool, error)
	GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error)
	GetTdByHash(ctx context.Context, hash common.Hash) (uint64, error)
	GetTdByNumber(ctx context.Context, blockNum *big.Int) (uint64, error)
	GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
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

	ContractInstanceCache map[common.Address]interface{}
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

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

type rpcTransaction struct {
	txExtraInfo
}

// NewContractCaller contract caller
func NewContractCaller() (contractCallerObj ContractCaller, err error) {
	config := GetConfig()
	contractCallerObj.MainChainClient = GetMainClient()
	contractCallerObj.MainChainTimeout = config.EthRPCTimeout
	contractCallerObj.BorChainClient = GetBorClient()
	contractCallerObj.BorChainTimeout = config.BorRPCTimeout
	contractCallerObj.MainChainRPCClient = GetMainChainRPCClient()
	contractCallerObj.BorChainRPCClient = GetBorRPCClient()
	contractCallerObj.BorChainGrpcFlag = config.BorGRPCFlag
	if client := GetBorGRPCClient(); client != nil {
		contractCallerObj.BorChainGrpcClient = client
	}

	// listeners and processors instance cache (address->ABI)
	contractCallerObj.ContractInstanceCache = make(map[common.Address]interface{})
	contractCallerObj.contractInstanceMu = &sync.RWMutex{}

	contractCallerObj.prefetchMu = &sync.RWMutex{}
	contractCallerObj.prefetchedReceipts = make(map[common.Hash]*ethTypes.Receipt)
	contractCallerObj.finalizedHeaderCache = nil

	// package global cache (string->ABI)
	if err = populateABIs(&contractCallerObj); err != nil {
		return contractCallerObj, err
	}

	return contractCallerObj, nil
}

// loadContractInstance returns the cached binding for address if present.
func (c *ContractCaller) loadContractInstance(address common.Address) (interface{}, bool) {
	c.contractInstanceMu.RLock()
	defer c.contractInstanceMu.RUnlock()
	v, ok := c.ContractInstanceCache[address]
	return v, ok
}

// storeContractInstance caches a successfully-constructed binding. Failed
// constructions must not be stored — a poisoned entry would type-assert to a
// typed-nil pointer on the next read.
func (c *ContractCaller) storeContractInstance(address common.Address, instance interface{}) {
	c.contractInstanceMu.Lock()
	defer c.contractInstanceMu.Unlock()
	c.ContractInstanceCache[address] = instance
}

// GetRootChainInstance returns the RootChain contract instance for a selected chain
func (c *ContractCaller) GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error) {
	address := common.HexToAddress(rootChainAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*rootchain.Rootchain), nil
	}

	ci, err := rootchain.NewRootchain(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the root chain instance from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetStakingInfoInstance returns stakingInfo contract instance for a selected chain
func (c *ContractCaller) GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error) {
	address := common.HexToAddress(stakingInfoAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*stakinginfo.Stakinginfo), nil
	}

	ci, err := stakinginfo.NewStakinginfo(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the stakingInfo instance from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetValidatorSetInstance returns stakingInfo contract instance for a selected chain
func (c *ContractCaller) GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error) {
	address := common.HexToAddress(validatorSetAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*validatorset.Validatorset), nil
	}

	ci, err := validatorset.NewValidatorset(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the validator set from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetStakeManagerInstance returns stakingInfo contract instance for a selected base chain
func (c *ContractCaller) GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error) {
	address := common.HexToAddress(stakingManagerAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*stakemanager.Stakemanager), nil
	}

	ci, err := stakemanager.NewStakemanager(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the stake manager from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetSlashManagerInstance returns the slashManager contract instance for a selected base chain
func (c *ContractCaller) GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error) {
	address := common.HexToAddress(slashManagerAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*slashmanager.Slashmanager), nil
	}

	ci, err := slashmanager.NewSlashmanager(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the slash manager from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetStateSenderInstance returns stakingInfo contract instance for a selected base chain
func (c *ContractCaller) GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error) {
	address := common.HexToAddress(stateSenderAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*statesender.Statesender), nil
	}

	ci, err := statesender.NewStatesender(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the stateSender from mainChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetStateReceiverInstance returns stakingInfo contract instance for a selected base chain
func (c *ContractCaller) GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error) {
	address := common.HexToAddress(stateReceiverAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*statereceiver.Statereceiver), nil
	}

	ci, err := statereceiver.NewStatereceiver(address, borClient)
	if err != nil {
		Logger.Error("Error in fetching the stateReceiver from borChain client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
}

// GetTokenInstance returns the contract instance for a selected chain
func (c *ContractCaller) GetTokenInstance(tokenAddress string) (*erc20.Erc20, error) {
	address := common.HexToAddress(tokenAddress)

	if cached, ok := c.loadContractInstance(address); ok {
		return cached.(*erc20.Erc20), nil
	}

	ci, err := erc20.NewErc20(address, mainChainClient)
	if err != nil {
		Logger.Error("Error in fetching the token address from client", "error", err)
		return nil, err
	}
	c.storeContractInstance(address, ci)
	return ci, nil
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

	var rootHash string
	var err error

	if c.BorChainGrpcFlag {
		grpcClient, grpcErr := c.getRequiredBorGRPCClient()
		if grpcErr != nil {
			return nil, grpcErr
		}
		rootHash, err = grpcClient.GetRootHash(callCtx, start, end)
	} else {
		rootHash, err = c.BorChainClient.GetRootHash(callCtx, start, end)
	}

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

	var vote bool
	var err error

	if c.BorChainGrpcFlag {
		grpcClient, grpcErr := c.getRequiredBorGRPCClient()
		if grpcErr != nil {
			return false, grpcErr
		}
		vote, err = grpcClient.GetVoteOnHash(ctx, start, end, hash, milestoneID)
	} else {
		vote, err = c.BorChainClient.GetVoteOnHash(ctx, start, end, hash, milestoneID)
	}

	if err != nil {
		return false, errors.New(fmt.Sprint("Error in fetching vote from bor chain", "err", err))
	}

	return vote, nil
}

// GetLastChildBlock fetch current child block
func (c *ContractCaller) GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error) {
	lastChildBlock, err := rootChainInstance.GetLastChildBlock(nil)
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
	currentHeaderBlock, err := rootChainInstance.CurrentHeaderBlock(nil)
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
	stakerDetails, err := stakingInfoInstance.GetStakerDetails(nil, big.NewInt(int64(valID)))
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
	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	var latestBlock *ethTypes.Header

	if c.BorChainGrpcFlag {
		grpcClient, grpcErr := c.getRequiredBorGRPCClient()
		if grpcErr != nil {
			Logger.Error(errUnableToConnect, "error", grpcErr)
			return nil, grpcErr
		}
		if blockNum == nil {
			// LatestBlockNumber is BlockNumber(-2) in go-ethereum rpc
			latestBlock, err = grpcClient.HeaderByNumber(ctx, -2)
		} else {
			latestBlock, err = grpcClient.HeaderByNumber(ctx, blockNum.Int64())
		}
	} else {
		latestBlock, err = c.BorChainClient.HeaderByNumber(ctx, blockNum)
	}

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
// BorChainGrpcFlag is set, otherwise falls back to the HTTP JSON-RPC batch.
// In both paths, it tries to get blocks from the range interval
// but returns only the ones found on the chain.
func (c *ContractCaller) GetBorChainBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	if start < 0 || end < 0 || end < start {
		return nil, nil, nil, fmt.Errorf("invalid range [%d,%d]", start, end)
	}
	// Prevents int64 (end-start+1) overflow
	if end-start > borgrpc.MaxBlockInfoBatchSize-1 {
		return nil, nil, nil, fmt.Errorf("range too large: %d blocks exceeds max %d", end-start+1, borgrpc.MaxBlockInfoBatchSize)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	if c.BorChainGrpcFlag {
		grpcClient, err := c.getRequiredBorGRPCClient()
		if err != nil {
			return nil, nil, nil, err
		}
		return grpcClient.GetBlockInfoInBatch(timeoutCtx, start, end)
	}

	return c.getBorChainBlockInfoInBatchHTTP(timeoutCtx, start, end)
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

// getBorChainBlockInfoInBatchHTTP is the HTTP/JSON-RPC implementation of GetBorChainBlockInfoInBatch.
// It issues a single RPC batch call covering headers, total difficulties, and authors for the
// inclusive range [start, end], and returns only the contiguous prefix of blocks found on the chain.
func (c *ContractCaller) getBorChainBlockInfoInBatchHTTP(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	// Range arithmetic compensated by the per-index bounds inside buildBorBatchElems/collateBorBatchResults.
	// mutator-disable-next-line range-arithmetic compensated downstream
	totalBlocks := end - start + 1
	rpcClient := c.BorChainClient.Client()

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

	if c.BorChainGrpcFlag {
		grpcClient, err := c.getRequiredBorGRPCClient()
		if err != nil {
			return 0, err
		}
		return grpcClient.GetTdByHash(ctx, blockHash)
	}

	rpcClient := c.BorChainClient.Client()

	var resp map[string]interface{}
	if err := rpcClient.CallContext(ctx, &resp, "eth_getTdByHash", blockHash.Hex()); err != nil {
		return 0, err
	}
	// Same path for gRPC and HTTP: a missing block surfaces as ethereum.NotFound
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

// GetBorChainBlockAuthor returns the producer of the bor block
func (c *ContractCaller) GetBorChainBlockAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	ctx, cancel := context.WithTimeout(ctx, c.BorChainTimeout)
	defer cancel()

	if c.BorChainGrpcFlag {
		grpcClient, err := c.getRequiredBorGRPCClient()
		if err != nil {
			// the return on the next line is what matters; log deletion is observable only in ops.
			// mutator-disable-next-line operator-log line
			Logger.Error(errUnableToConnect, "error", err)
			return nil, err
		}
		author, err := grpcClient.GetAuthor(ctx, blockNum)
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

	var author *common.Address
	err := c.BorChainClient.Client().CallContext(ctx, &author, "bor_getAuthor", toBlockNumArg(blockNum))
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
	var rpcTx rpcTransaction
	if err := c.MainChainRPCClient.CallContext(context.Background(), &rpcTx, "eth_getTransactionByHash", tx); err != nil {
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

// DecodeNewHeaderBlockEvent represents the new header block event
func (c *ContractCaller) DecodeNewHeaderBlockEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*rootchain.RootchainNewHeaderBlock, error) {
	event := new(rootchain.RootchainNewHeaderBlock)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.RootChainABI, event, NewHeaderBlockEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeValidatorTopupFeesEvent represents topUp for fees tokens
func (c *ContractCaller) DecodeValidatorTopupFeesEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoTopUpFee, error) {
	event := new(stakinginfo.StakinginfoTopUpFee)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, TopUpFeeEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeValidatorJoinEvent represents validator staked event
func (c *ContractCaller) DecodeValidatorJoinEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoStaked, error) {
	event := new(stakinginfo.StakinginfoStaked)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, StakedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeValidatorStakeUpdateEvent represents validator stake update event
func (c *ContractCaller) DecodeValidatorStakeUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoStakeUpdate, error) {
	event := new(stakinginfo.StakinginfoStakeUpdate)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, StakeUpdateEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeValidatorExitEvent represents validator stake unStake event
func (c *ContractCaller) DecodeValidatorExitEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnstakeInit, error) {
	event := new(stakinginfo.StakinginfoUnstakeInit)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, UnstakeInitEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeSignerUpdateEvent represents sig update event
func (c *ContractCaller) DecodeSignerUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSignerChange, error) {
	event := new(stakinginfo.StakinginfoSignerChange)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, SignerChangeEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeStateSyncedEvent decode state sync data
func (c *ContractCaller) DecodeStateSyncedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*statesender.StatesenderStateSynced, error) {
	event := new(statesender.StatesenderStateSynced)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StateSenderABI, event, StateSyncedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// decode slashing events

// DecodeSlashedEvent represents tick ack on contract
func (c *ContractCaller) DecodeSlashedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSlashed, error) {
	event := new(stakinginfo.StakinginfoSlashed)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, SlashedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

// DecodeUnJailedEvent represents unJail on contract
func (c *ContractCaller) DecodeUnJailedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnJailed, error) {
	event := new(stakinginfo.StakinginfoUnJailed)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, UnJailedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New(errEventNotFound)
}

//
// Account root functions
//

// CurrentAccountStateRoot get current account root from on the chain
func (c *ContractCaller) CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error) {
	accountStateRoot, err := stakingInfoInstance.GetAccountStateRoot(nil)
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
	result, err := validatorSetInstance.CurrentSpanNumber(nil)
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
	d, err := validatorSetInstance.GetSpan(nil, id)
	if err != nil {
		return nil, nil, nil, errors.New("unable to get span details")
	}
	return d.Number, d.StartBlock, d.EndBlock, nil
}

// CurrentStateCounter get state counter
func (c *ContractCaller) CurrentStateCounter(stateSenderInstance *statesender.Statesender) (Number *big.Int) {
	result, err := stateSenderInstance.Counter(nil)
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

	var (
		header *ethTypes.Header
		err    error
	)

	if c.BorChainGrpcFlag {
		grpcClient, grpcErr := c.getRequiredBorGRPCClient()
		if grpcErr != nil {
			return false, grpcErr
		}
		header, err = grpcClient.HeaderByNumber(callCtx, int64(number))
	} else {
		header, err = c.BorChainClient.HeaderByNumber(callCtx, big.NewInt(int64(number)))
	}
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
	var block *ethTypes.Block
	var err error

	if c.BorChainGrpcFlag {
		grpcClient, grpcErr := c.getRequiredBorGRPCClient()
		if grpcErr != nil {
			return nil, grpcErr
		}
		block, err = grpcClient.BlockByNumber(ctx, int64(blockNumber))
	} else {
		block, err = c.BorChainClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	}

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

	return c.getTxReceipt(callCtx, c.MainChainClient, txHash)
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

	if c.BorChainGrpcFlag {
		grpcClient, err := c.getRequiredBorGRPCClient()
		if err != nil {
			return nil, err
		}
		return grpcClient.TransactionReceipt(ctx, txHash)
	}
	return c.getTxReceipt(ctx, c.BorChainClient, txHash)
}

func (c *ContractCaller) getTxReceipt(
	ctx context.Context,
	client interface {
		TransactionReceipt(context.Context, common.Hash) (*ethTypes.Receipt, error)
	},
	txHash common.Hash,
) (*ethTypes.Receipt, error) {
	return client.TransactionReceipt(ctx, txHash)
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

// getRequiredBorGRPCClient returns the bor grpc client or an error
func (c *ContractCaller) getRequiredBorGRPCClient() (BorGRPCClienter, error) {
	if c.BorChainGrpcClient == nil {
		return nil, errors.New("bor grpc client is nil while bor grpc flag is enabled")
	}
	return c.BorChainGrpcClient, nil
}

// utility and helper methods

// populateABIs fills the package level cache for contracts' ABIs.
// When called the first time, ContractsABIsMap will be filled,
// and the getABI method won't be invoked the next time.
// This reduces the number of calls to JSON decode methods made by the contract caller.
// It uses ABIs' definitions instead of contracts' addresses,
// as the latter might not be available at initialization time
func populateABIs(contractCallerObj *ContractCaller) error {
	var ccAbi *abi.ABI

	var err error

	contractsABIs := [8]string{
		rootchain.RootchainMetaData.ABI, stakinginfo.StakinginfoMetaData.ABI, validatorset.ValidatorsetMetaData.ABI,
		statereceiver.StatereceiverMetaData.ABI, statesender.StatesenderMetaData.ABI, stakemanager.StakemanagerMetaData.ABI,
		slashmanager.SlashmanagerMetaData.ABI, erc20.Erc20MetaData.ABI,
	}

	// iterate over supported ABIs
	for _, contractABI := range contractsABIs {
		ccAbi, err = chooseContractCallerABI(contractCallerObj, contractABI)
		if err != nil {
			Logger.Error("Error while fetching contract caller ABI", "error", err)
			return err
		}

		if ContractsABIsMap[contractABI] == nil {
			// fills cached abi map
			if *ccAbi, err = getABI(contractABI); err != nil {
				Logger.Error("Error while getting ABI for contract caller", "name", contractABI, "error", err)
				return err
			}
			ContractsABIsMap[contractABI] = ccAbi
		} else {
			// use cached abi
			*ccAbi = *ContractsABIsMap[contractABI]
		}
	}

	return nil
}

// chooseContractCallerABI extracts and returns the abo.ABI object from the contractCallerObj based on its abi string
func chooseContractCallerABI(contractCallerObj *ContractCaller, abi string) (*abi.ABI, error) {
	switch abi {
	case rootchain.RootchainMetaData.ABI:
		return &contractCallerObj.RootChainABI, nil
	case stakinginfo.StakinginfoMetaData.ABI:
		return &contractCallerObj.StakingInfoABI, nil
	case validatorset.ValidatorsetMetaData.ABI:
		return &contractCallerObj.ValidatorSetABI, nil
	case statereceiver.StatereceiverMetaData.ABI:
		return &contractCallerObj.StateReceiverABI, nil
	case statesender.StatesenderMetaData.ABI:
		return &contractCallerObj.StateSenderABI, nil
	case stakemanager.StakemanagerMetaData.ABI:
		return &contractCallerObj.StakeManagerABI, nil
	case slashmanager.SlashmanagerMetaData.ABI:
		return &contractCallerObj.SlashManagerABI, nil
	case erc20.Erc20MetaData.ABI:
		return &contractCallerObj.PolTokenABI, nil
	}

	return nil, errors.New("no ABI associated with such data")
}

// getABI returns the contract's ABI struct from on its JSON representation
func getABI(data string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(data))
}

// copied from bor/ethclient package
func toBlockNumArg(number *big.Int) string {
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
