package helper

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/slashmanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statereceiver"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/contracts/validatorset"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// smart contracts' events names
const (
	NewHeaderBlockEvent = "NewHeaderBlock"
	TopUpFeeEvent       = "TopUpFee"
	StakedEvent         = "Staked"
	StakeUpdateEvent    = "StakeUpdate"
	UnstakeInitEvent    = "UnstakeInit"
	SignerChangeEvent   = "SignerChange"
	StateSyncedEvent    = "StateSynced"
	SlashedEvent        = "Slashed"
	UnJailedEvent       = "UnJailed"
)

// ContractsABIsMap is a cached map holding the ABIs of the contracts
var ContractsABIsMap = make(map[string]*abi.ABI)

// IContractCaller represents contract caller
type IContractCaller interface {
	GetHeaderInfo(headerID uint64, rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (root common.Hash, start, end, createdAt uint64, proposer string, err error)
	GetRootHash(start, end, checkpointLength uint64) ([]byte, error)
	GetVoteOnHash(start, end uint64, hash, milestoneID string) (bool, error)
	GetValidatorInfo(valID uint64, stakingInfoInstance *stakinginfo.Stakinginfo) (validator types.Validator, err error)
	GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error)
	CurrentHeaderBlock(rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (uint64, error)
	GetBalance(address common.Address) (*big.Int, error)
	SendCheckpoint(signedData []byte, sigs [][3]*big.Int, rootChainAddress common.Address, rootChainInstance *rootchain.Rootchain) (err error)
	GetCheckpointSign(txHash common.Hash) ([]byte, []byte, []byte, error)
	GetMainChainBlock(*big.Int) (*ethTypes.Header, error)
	GetPolygonPosChainBlock(*big.Int) (*ethTypes.Header, error)
	IsTxConfirmed(common.Hash, uint64) bool
	GetConfirmedTxReceipt(common.Hash, uint64) (*ethTypes.Receipt, error)
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

	GetMainTxReceipt(common.Hash) (*ethTypes.Receipt, error)
	GetPolygonPosTxReceipt(common.Hash) (*ethTypes.Receipt, error)
	ApproveTokens(*big.Int, common.Address, common.Address, *erc20.Erc20) error
	StakeFor(common.Address, *big.Int, *big.Int, bool, common.Address, *stakemanager.Stakemanager) error
	CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error)
	CurrentSpanNumber(validatorSet *validatorset.Validatorset) (Number *big.Int)
	GetSpanDetails(id *big.Int, validatorSet *validatorset.Validatorset) (*big.Int, *big.Int, *big.Int, error)
	CurrentStateCounter(stateSenderInstance *statesender.Statesender) (Number *big.Int)
	CheckIfBlocksExist(end uint64) bool
	GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error)
	GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error)
	GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error)
	GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error)
	GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error)
	GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error)
	GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error)
	GetPolygonPosTokenInstance(tokenAddress string) (*erc20.Erc20, error)
}

// ContractCaller contract caller
type ContractCaller struct {
	MainChainClient        *ethclient.Client
	MainChainRPC           *rpc.Client
	MainChainTimeout       time.Duration
	PolygonPosChainClient  *ethclient.Client
	PolygonPosChainRPC     *rpc.Client
	PolygonPosChainTimeout time.Duration

	RootChainABI       abi.ABI
	StakingInfoABI     abi.ABI
	ValidatorSetABI    abi.ABI
	StateReceiverABI   abi.ABI
	StateSenderABI     abi.ABI
	StakeManagerABI    abi.ABI
	SlashManagerABI    abi.ABI
	PolygonPosTokenABI abi.ABI

	ReceiptCache *lru.Cache

	ContractInstanceCache map[common.Address]interface{}
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
	contractCallerObj.PolygonPosChainClient = GetPolygonPosClient()
	contractCallerObj.PolygonPosChainTimeout = config.BorRPCTimeout
	contractCallerObj.MainChainRPC = GetMainChainRPCClient()
	contractCallerObj.PolygonPosChainRPC = GetPolygonPosRPCClient()
	contractCallerObj.ReceiptCache, err = lru.New(1000)

	if err != nil {
		return contractCallerObj, err
	}

	// listeners and processors instance cache (address->ABI)
	contractCallerObj.ContractInstanceCache = make(map[common.Address]interface{})
	// package global cache (string->ABI)
	if err = populateABIs(&contractCallerObj); err != nil {
		return contractCallerObj, err
	}

	return contractCallerObj, nil
}

// GetRootChainInstance returns RootChain contract instance for selected base chain
func (c *ContractCaller) GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error) {
	address := common.HexToAddress(rootChainAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := rootchain.NewRootchain(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the root chain instance from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*rootchain.Rootchain), nil
}

// GetStakingInfoInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error) {
	address := common.HexToAddress(stakingInfoAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := stakinginfo.NewStakinginfo(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the stakinginfo instance from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*stakinginfo.Stakinginfo), nil
}

// GetValidatorSetInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error) {
	address := common.HexToAddress(validatorSetAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := validatorset.NewValidatorset(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the validator set from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*validatorset.Validatorset), nil
}

// GetStakeManagerInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error) {
	address := common.HexToAddress(stakingManagerAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := stakemanager.NewStakemanager(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the stake manager from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*stakemanager.Stakemanager), nil
}

// GetSlashManagerInstance returns slashManager contract instance for selected base chain
func (c *ContractCaller) GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error) {
	address := common.HexToAddress(slashManagerAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := slashmanager.NewSlashmanager(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the slash manager from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*slashmanager.Slashmanager), nil
}

// GetStateSenderInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error) {
	address := common.HexToAddress(stateSenderAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := statesender.NewStatesender(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the statesender from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*statesender.Statesender), nil
}

// GetStateReceiverInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error) {
	address := common.HexToAddress(stateReceiverAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := statereceiver.NewStatereceiver(address, polygonPosClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the statereceiver from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*statereceiver.Statereceiver), nil
}

// GetPolygonPosTokenInstance returns stakingInfo contract instance for selected base chain
func (c *ContractCaller) GetPolygonPosTokenInstance(tokenAddress string) (*erc20.Erc20, error) {
	address := common.HexToAddress(tokenAddress)

	contractInstance, ok := c.ContractInstanceCache[address]
	if !ok {
		ci, err := erc20.NewErc20(address, mainChainClient)
		c.ContractInstanceCache[address] = ci

		if err != nil {
			Logger.Error("error in fetching the polygon pos token address from mainchain client", "error", err)
			return nil, err
		}

		return ci, err
	}

	return contractInstance.(*erc20.Erc20), nil
}

// GetHeaderInfo get header info from checkpoint number
func (c *ContractCaller) GetHeaderInfo(headerID uint64, rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (
	root common.Hash,
	start,
	end,
	createdAt uint64,
	proposer string,
	err error,
) {
	// get header from rootChain
	checkpointBigInt := big.NewInt(0).Mul(big.NewInt(0).SetUint64(headerID), big.NewInt(0).SetUint64(childBlockInterval))

	headerBlock, err := rootChainInstance.HeaderBlocks(nil, checkpointBigInt)
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

// GetRootHash get root hash from bor chain for the corresponding start and end block
func (c *ContractCaller) GetRootHash(start, end, checkpointLength uint64) ([]byte, error) {
	noOfBlock := end - start + 1

	if start > end {
		return nil, errors.New("start is greater than end")
	}

	if noOfBlock > checkpointLength {
		return nil, errors.New("number of headers requested exceeds checkpoint length")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.PolygonPosChainTimeout)
	defer cancel()

	rootHash, err := c.PolygonPosChainClient.GetRootHash(ctx, start, end)

	if err != nil {
		Logger.Error("could not fetch rootHash from polygon pos chain", "error", err)
		return nil, err
	}

	return common.FromHex(rootHash), nil
}

// GetVoteOnHash get vote on hash from bor chain for the corresponding milestone
func (c *ContractCaller) GetVoteOnHash(start, end uint64, hash, milestoneID string) (bool, error) {
	if start > end {
		return false, errors.New("Start block number is greater than the end block number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.PolygonPosChainTimeout)
	defer cancel()

	vote, err := c.PolygonPosChainClient.GetVoteOnHash(ctx, start, end, hash, milestoneID)
	if err != nil {
		return false, errors.New(fmt.Sprint("Error in fetching vote from polygon pos chain", "err", err))
	}

	return vote, nil
}

// GetLastChildBlock fetch current child block
func (c *ContractCaller) GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error) {
	GetLastChildBlock, err := rootChainInstance.GetLastChildBlock(nil)
	if err != nil {
		Logger.Error("Could not fetch current child block from rootChain contract", "error", err)
		return 0, err
	}

	return GetLastChildBlock.Uint64(), nil
}

// CurrentHeaderBlock fetches current header block
func (c *ContractCaller) CurrentHeaderBlock(rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (uint64, error) {
	currentHeaderBlock, err := rootChainInstance.CurrentHeaderBlock(nil)
	if err != nil {
		Logger.Error("Could not fetch current header block from rootChain contract", "error", err)
		return 0, err
	}

	return currentHeaderBlock.Uint64() / childBlockInterval, nil
}

// GetBalance get balance of account (returns big.Int balance won't fit in uint64)
func (c *ContractCaller) GetBalance(address common.Address) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	balance, err := c.MainChainClient.BalanceAt(ctx, address, nil)
	if err != nil {
		Logger.Error("unable to fetch balance of account from root chain", "Address", address.String(), "error", err)
		return big.NewInt(0), err
	}

	return balance, nil
}

// GetValidatorInfo get validator info
func (c *ContractCaller) GetValidatorInfo(valID uint64, stakingInfoInstance *stakinginfo.Stakinginfo) (validator types.Validator, err error) {

	stakerDetails, err := stakingInfoInstance.GetStakerDetails(nil, big.NewInt(int64(valID)))
	if err != nil && &stakerDetails != nil {
		Logger.Error("error fetching validator information from stake manager", "validatorId", valID, "status", stakerDetails.Status, "error", err)
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
func (c *ContractCaller) GetMainChainBlock(blockNum *big.Int) (header *ethTypes.Header, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	latestBlock, err := c.MainChainClient.HeaderByNumber(ctx, blockNum)
	if err != nil {
		Logger.Error("unable to connect to main chain", "error", err)
		return
	}

	return latestBlock, nil
}

// GetMainChainFinalizedBlock returns finalized main chain block header (post-merge)
func (c *ContractCaller) GetMainChainFinalizedBlock() (header *ethTypes.Header, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	latestFinalizedBlock, err := c.MainChainClient.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		Logger.Error("unable to connect to polygon pos chain", "error", err)
		return
	}

	return latestFinalizedBlock, nil
}

// GetMainChainBlockTime returns main chain block time
func (c *ContractCaller) GetMainChainBlockTime(ctx context.Context, blockNum uint64) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, c.MainChainTimeout)
	defer cancel()

	latestBlock, err := c.MainChainClient.BlockByNumber(ctx, big.NewInt(0).SetUint64(blockNum))
	if err != nil {
		Logger.Error("unable to connect to polygon pos chain", "error", err)
		return time.Time{}, err
	}

	return time.Unix(int64(latestBlock.Time()), 0), nil
}

// GetPolygonPosChainBlock returns child chain block header
func (c *ContractCaller) GetPolygonPosChainBlock(blockNum *big.Int) (header *ethTypes.Header, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.PolygonPosChainTimeout)
	defer cancel()

	latestBlock, err := c.PolygonPosChainClient.HeaderByNumber(ctx, blockNum)
	if err != nil {
		Logger.Error("unable to connect to polygon pos chain", "error", err)
		return
	}

	return latestBlock, nil
}

// GetBlockNumberFromTxHash gets block number of transaction
func (c *ContractCaller) GetBlockNumberFromTxHash(tx common.Hash) (*big.Int, error) {
	var rpcTx rpcTransaction
	if err := c.MainChainRPC.CallContext(context.Background(), &rpcTx, "eth_getTransactionByHash", tx); err != nil {
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
func (c *ContractCaller) IsTxConfirmed(txHash common.Hash, requiredConfirmations uint64) bool {
	// get main tx receipt
	receipt, err := c.GetConfirmedTxReceipt(txHash, requiredConfirmations)
	if err != nil {
		Logger.Error("error while fetching the tx receipt", "error", err)
		return false
	}

	if receipt == nil {
		return false
	}

	return true
}

// GetConfirmedTxReceipt returns confirmed tx receipt
func (c *ContractCaller) GetConfirmedTxReceipt(tx common.Hash, requiredConfirmations uint64) (*ethTypes.Receipt, error) {
	var receipt *ethTypes.Receipt

	receiptCache, ok := c.ReceiptCache.Get(tx.String())
	if !ok {
		var err error

		// get main tx receipt
		receipt, err = c.GetMainTxReceipt(tx)
		if err != nil {
			Logger.Error("error while fetching mainChain receipt", "txHash", tx.Hex(), "error", err)
			return nil, err
		}

		c.ReceiptCache.Add(tx.String(), receipt)
	} else {
		receipt, ok = receiptCache.(*ethTypes.Receipt)
		if !ok {
			return nil, errors.New("error in casting the fetched receipt into eth receipt")
		}
	}

	receiptBlockNumber := receipt.BlockNumber.Uint64()

	Logger.Debug("tx included in block", "block", receiptBlockNumber, "tx", tx)

	// fetch the last finalized main chain block (available post-merge)
	latestFinalizedBlock, err := c.GetMainChainFinalizedBlock()
	if err != nil {
		Logger.Error("error getting latest finalized block from main chain", "error", err)
	}

	// If latest finalized block is available, use it to check if receipt is finalized or not.
	// Else, fallback to the `requiredConfirmations` value
	if latestFinalizedBlock != nil {
		Logger.Debug("latest finalized block on main chain obtained", "Block", latestFinalizedBlock.Number.Uint64(), "receipt block", receiptBlockNumber)

		if receiptBlockNumber > latestFinalizedBlock.Number.Uint64() {
			return nil, errors.New("not enough confirmations")
		}
	} else {
		// get current/latest main chain block
		latestBlk, err := c.GetMainChainBlock(nil)
		if err != nil {
			Logger.Error("error getting latest block from main chain", "error", err)
			return nil, err
		}

		Logger.Debug("latest block on main chain obtained", "Block", latestBlk.Number.Uint64(), "receipt block", receiptBlockNumber)

		diff := latestBlk.Number.Uint64() - receiptBlockNumber
		if diff < requiredConfirmations {
			return nil, errors.New("not enough confirmations")
		}
	}

	return receipt, nil
}

//
// Validator decode events
//

// DecodeNewHeaderBlockEvent represents new header block event
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

	return nil, errors.New("event not found")
}

// DecodeValidatorTopupFeesEvent represents topUp for fees tokens
func (c *ContractCaller) DecodeValidatorTopupFeesEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoTopUpFee, error) {
	var (
		event = new(stakinginfo.StakinginfoTopUpFee)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, TopUpFeeEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")
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

	return nil, errors.New("event not found")
}

// DecodeValidatorStakeUpdateEvent represents validator stake update event
func (c *ContractCaller) DecodeValidatorStakeUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoStakeUpdate, error) {
	var (
		event = new(stakinginfo.StakinginfoStakeUpdate)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, StakeUpdateEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")

}

// DecodeValidatorExitEvent represents validator stake unStake event
func (c *ContractCaller) DecodeValidatorExitEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnstakeInit, error) {
	var (
		event = new(stakinginfo.StakinginfoUnstakeInit)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, UnstakeInitEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")

}

// DecodeSignerUpdateEvent represents sig update event
func (c *ContractCaller) DecodeSignerUpdateEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSignerChange, error) {
	var (
		event = new(stakinginfo.StakinginfoSignerChange)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, SignerChangeEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")
}

// DecodeStateSyncedEvent decode state sync data
func (c *ContractCaller) DecodeStateSyncedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*statesender.StatesenderStateSynced, error) {
	var (
		event = new(statesender.StatesenderStateSynced)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StateSenderABI, event, StateSyncedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")
}

// decode slashing events

// DecodeSlashedEvent represents tick ack on contract
func (c *ContractCaller) DecodeSlashedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoSlashed, error) {
	var (
		event = new(stakinginfo.StakinginfoSlashed)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, SlashedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")
}

// DecodeUnJailedEvent represents unJail on contract
func (c *ContractCaller) DecodeUnJailedEvent(contractAddressString string, receipt *ethTypes.Receipt, logIndex uint64) (*stakinginfo.StakinginfoUnJailed, error) {
	var (
		event = new(stakinginfo.StakinginfoUnJailed)
	)

	contractAddress := common.HexToAddress(contractAddressString)

	for _, vLog := range receipt.Logs {
		if uint64(vLog.Index) == logIndex && bytes.Equal(vLog.Address.Bytes(), contractAddress.Bytes()) {
			if err := UnpackLog(&c.StakingInfoABI, event, UnJailedEvent, vLog); err != nil {
				return nil, err
			}

			return event, nil
		}
	}

	return nil, errors.New("event not found")
}

//
// Account root related functions
//

// CurrentAccountStateRoot get current account root from on chain
func (c *ContractCaller) CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error) {
	accountStateRoot, err := stakingInfoInstance.GetAccountStateRoot(nil)

	if err != nil {
		Logger.Error("unable to get current account state root", "error", err)

		var emptyArr [32]byte

		return emptyArr, err
	}

	return accountStateRoot, nil
}

//
// Span related functions
//

// CurrentSpanNumber get current span
func (c *ContractCaller) CurrentSpanNumber(validatorSetInstance *validatorset.Validatorset) (Number *big.Int) {
	result, err := validatorSetInstance.CurrentSpanNumber(nil)
	if err != nil {
		Logger.Error("unable to get current span number", "error", err)
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
	if &d != nil {
		return d.Number, d.StartBlock, d.EndBlock, err
	}
	return nil, nil, nil, errors.New("unable to get span details")
}

// CurrentStateCounter get state counter
func (c *ContractCaller) CurrentStateCounter(stateSenderInstance *statesender.Statesender) (Number *big.Int) {
	result, err := stateSenderInstance.Counter(nil)
	if err != nil {
		Logger.Error("unable to get current counter number", "error", err)
		return nil
	}

	return result
}

// CheckIfBlocksExist - check if the given block exists on local chain
func (c *ContractCaller) CheckIfBlocksExist(end uint64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), c.PolygonPosChainTimeout)
	defer cancel()

	block := c.GetBlockByNumber(ctx, end)
	if block == nil {
		return false
	}

	return end == block.NumberU64()
}

// GetBlockByNumber returns blocks by number from child chain (bor)
func (c *ContractCaller) GetBlockByNumber(ctx context.Context, blockNumber uint64) *ethTypes.Block {
	block, err := c.PolygonPosChainClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		Logger.Error("unable to fetch block by number from child chain", "block", block, "err", err)
		return nil
	}

	return block
}

//
// Receipt functions
//

// GetMainTxReceipt returns main tx receipt
func (c *ContractCaller) GetMainTxReceipt(txHash common.Hash) (*ethTypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	return c.getTxReceipt(ctx, c.MainChainClient, txHash)
}

// GetPolygonPosTxReceipt returns polygon pos tx receipt
func (c *ContractCaller) GetPolygonPosTxReceipt(txHash common.Hash) (*ethTypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.PolygonPosChainTimeout)
	defer cancel()

	return c.getTxReceipt(ctx, c.PolygonPosChainClient, txHash)
}

func (c *ContractCaller) getTxReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*ethTypes.Receipt, error) {
	return client.TransactionReceipt(ctx, txHash)
}

// GetCheckpointSign returns sigs input of committed checkpoint transaction
func (c *ContractCaller) GetCheckpointSign(txHash common.Hash) ([]byte, []byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.MainChainTimeout)
	defer cancel()

	mainChainClient := GetMainClient()

	transaction, isPending, err := mainChainClient.TransactionByHash(ctx, txHash)
	if err != nil {
		Logger.Error("error while Fetching Transaction By hash from MainChain", "error", err)
		return []byte{}, []byte{}, []byte{}, err
	} else if isPending {
		return []byte{}, []byte{}, []byte{}, errors.New("transaction is still pending")
	}

	payload := transaction.Data()
	chainABI := c.RootChainABI

	return UnpackSigAndVotes(payload, chainABI)
}

// utility and helper methods

// populateABIs fills the package level cache for contracts' ABIs
// When called the first time, ContractsABIsMap will be filled and getABI method won't be invoked the next times
// This reduces the number of calls to json decode methods made by the contract caller
// It uses ABIs' definitions instead of contracts addresses, as the latter might not be available at init time
func populateABIs(contractCallerObj *ContractCaller) error {
	var ccAbi *abi.ABI

	var err error

	contractsABIs := [8]string{rootchain.RootchainMetaData.ABI, stakinginfo.StakinginfoMetaData.ABI, validatorset.ValidatorsetMetaData.ABI,
		statereceiver.StatereceiverMetaData.ABI, statesender.StatesenderMetaData.ABI, stakemanager.StakemanagerMetaData.ABI,
		slashmanager.SlashmanagerMetaData.ABI, erc20.Erc20MetaData.ABI}

	// iterate over supported ABIs
	for _, contractABI := range contractsABIs {
		ccAbi, err = chooseContractCallerABI(contractCallerObj, contractABI)
		if err != nil {
			Logger.Error("error while fetching contract caller ABI", "error", err)
			return err
		}

		if ContractsABIsMap[contractABI] == nil {
			// fills cached abi map
			if *ccAbi, err = getABI(contractABI); err != nil {
				Logger.Error("error while getting ABI for contract caller", "name", contractABI, "error", err)
				return err
			} else {
				ContractsABIsMap[contractABI] = ccAbi
				Logger.Debug("ABI initialized", "name", contractABI)
			}
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
		return &contractCallerObj.PolygonPosTokenABI, nil
	}

	return nil, errors.New("no ABI associated with such data")
}

// getABI returns the contract's ABI struct from on its JSON representation
func getABI(data string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(data))
}
