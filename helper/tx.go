package helper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
)

const errUnableToCreateAuthObj = "unable to create auth object"

// EthClient defines the interface for Ethereum client operations needed for transaction creation.
type EthClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

// GenerateAuthObj creates a transaction auth object with EIP-1559 gas pricing.
func GenerateAuthObj(client *ethclient.Client, address common.Address, data []byte) (auth *bind.TransactOpts, err error) {
	return GenerateAuthObjWithContext(context.Background(), GetConfig().EthRPCTimeout, client, address, data)
}

// GenerateAuthObjWithContext creates a transaction auth object using ctx as the
// parent cancellation signal and timeout as the budget for each RPC call.
func GenerateAuthObjWithContext(ctx context.Context, timeout time.Duration, client *ethclient.Client, address common.Address, data []byte) (auth *bind.TransactOpts, err error) {
	return generateAuthObjWithClient(ctx, timeout, client, address, data)
}

type authRPCData struct {
	latestBlock     *types.Block
	suggestedTipCap *big.Int
	nonce           uint64
	gasLimit        uint64
	chainID         *big.Int
}

// generateAuthObjWithClient creates a transaction auth object using the EthClient interface.
// This function is used internally and allows for easier testing with mock clients.
func generateAuthObjWithClient(ctx context.Context, timeout time.Duration, client EthClient, address common.Address, data []byte) (auth *bind.TransactOpts, err error) {
	callMsg := ethereum.CallMsg{
		To:   &address,
		Data: data,
	}

	pkObject := GetPrivKey()
	ecdsaPrivateKey, err := crypto.ToECDSA(pkObject[:])
	if err != nil {
		return
	}

	// Get the from address.
	fromAddress := common.BytesToAddress(pkObject.PubKey().Address().Bytes())
	callMsg.From = fromAddress

	rpcData, err := fetchAuthRPCData(ctx, timeout, client, fromAddress, callMsg)
	if err != nil {
		return
	}

	baseFee := rpcData.latestBlock.BaseFee()
	gasFeeCap, gasTipCap, err := calculateEIP1559Caps(baseFee, rpcData.suggestedTipCap)
	if err != nil {
		return
	}

	auth, err = bind.NewKeyedTransactorWithChainID(ecdsaPrivateKey, rpcData.chainID)
	if err != nil {
		Logger.Error(errUnableToCreateAuthObj, "error", err)
		return
	}

	auth.Nonce = big.NewInt(int64(rpcData.nonce))
	auth.GasLimit = rpcData.gasLimit
	auth.GasFeeCap = gasFeeCap
	auth.GasTipCap = gasTipCap

	Logger.Debug("created EIP-1559 transaction auth",
		"nonce", rpcData.nonce,
		"gasLimit", rpcData.gasLimit,
		"gasFeeCap", gasFeeCap,
		"gasTipCap", gasTipCap,
		"baseFee", baseFee,
	)

	return
}

func calculateEIP1559Caps(baseFee, suggestedTipCap *big.Int) (*big.Int, *big.Int, error) {
	if baseFee == nil {
		err := errors.New("baseFee is nil, EIP-1559 not supported")
		Logger.Error("EIP-1559 not supported on this chain", "error", err)
		return nil, nil, err
	}
	if suggestedTipCap == nil {
		err := errors.New("suggested gas tip cap is nil")
		Logger.Error("unable to fetch suggested gas tip cap", "error", err)
		return nil, nil, err
	}

	configGasFeeCap, configGasTipCap := configuredGasCaps()

	gasTipCap := suggestedTipCap
	if gasTipCap.Cmp(big.NewInt(configGasTipCap)) > 0 {
		Logger.Warn(
			"suggested tip cap exceeds configured maximum, using configured maximum",
			"suggested", suggestedTipCap.String(),
			"configured", configGasTipCap,
		)
		gasTipCap = big.NewInt(configGasTipCap)
	}

	// Calculate gas fee cap: (baseFee * 2) + tipCap.
	// The 2x multiplier provides buffer for baseFee fluctuations.
	gasFeeCap := new(big.Int).Mul(baseFee, big.NewInt(2))
	gasFeeCap.Add(gasFeeCap, gasTipCap)

	// Cap the gas fee cap to configured maximum.
	maxGasFeeCap := big.NewInt(configGasFeeCap)
	if gasFeeCap.Cmp(maxGasFeeCap) > 0 {
		Logger.Warn("calculated gas fee cap exceeds configured maximum, using configured maximum",
			"calculated", gasFeeCap, "configured", maxGasFeeCap)
		gasFeeCap = maxGasFeeCap
	}

	// Ensure tip cap doesn't exceed fee cap.
	if gasTipCap.Cmp(gasFeeCap) > 0 {
		gasTipCap = gasFeeCap
	}

	return gasFeeCap, gasTipCap, nil
}

func configuredGasCaps() (int64, int64) {
	configGasFeeCap := GetConfig().MainChainGasFeeCap
	if configGasFeeCap <= 0 {
		configGasFeeCap = DefaultMainChainGasFeeCap
	}
	configGasTipCap := GetConfig().MainChainGasTipCap
	if configGasTipCap <= 0 {
		configGasTipCap = DefaultMainChainGasTipCap
	}
	return configGasFeeCap, configGasTipCap
}

func fetchAuthRPCData(ctx context.Context, timeout time.Duration, client EthClient, fromAddress common.Address, callMsg ethereum.CallMsg) (authRPCData, error) {
	var out authRPCData
	var err error

	out.latestBlock, err = fetchAuthLatestBlock(ctx, timeout, client)
	if err != nil {
		return out, err
	}

	out.suggestedTipCap, err = callWithTimeout(ctx, timeout, client.SuggestGasTipCap)
	if err != nil {
		Logger.Error("unable to fetch suggested gas tip cap", "error", err)
		return out, err
	}

	out.nonce, err = callWithTimeout(ctx, timeout, func(callCtx context.Context) (uint64, error) {
		return client.PendingNonceAt(callCtx, fromAddress)
	})
	if err != nil {
		return out, err
	}

	out.gasLimit, err = callWithTimeout(ctx, timeout, func(callCtx context.Context) (uint64, error) {
		return client.EstimateGas(callCtx, callMsg)
	})
	if err != nil {
		Logger.Error("Unable to estimate gas", "error", err)
		return out, err
	}

	out.chainID, err = callWithTimeout(ctx, timeout, client.ChainID)
	if err != nil {
		Logger.Error("Unable to fetch ChainID", "error", err)
		return out, err
	}
	if out.chainID == nil {
		return out, errors.New("chain ID is nil")
	}

	return out, nil
}

func fetchAuthLatestBlock(ctx context.Context, timeout time.Duration, client EthClient) (*types.Block, error) {
	latestBlock, err := callWithTimeout(ctx, timeout, func(callCtx context.Context) (*types.Block, error) {
		return client.BlockByNumber(callCtx, nil)
	})
	if err != nil {
		Logger.Error("unable to fetch latest block", "error", err)
		return nil, err
	}
	if latestBlock == nil {
		return nil, errors.New("latest block is nil")
	}
	if latestBlock.BaseFee() == nil {
		err := errors.New("baseFee is nil, EIP-1559 not supported")
		Logger.Error("EIP-1559 not supported on this chain", "error", err)
		return nil, err
	}
	return latestBlock, nil
}

func rpcCallContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultEthRPCTimeout
	}
	return context.WithTimeout(parent, timeout)
}

func callWithTimeout[T any](parent context.Context, timeout time.Duration, fn func(context.Context) (T, error)) (T, error) {
	callCtx, cancel := rpcCallContext(parent, timeout)
	defer cancel()
	return fn(callCtx)
}

// SendCheckpoint sends checkpoint to rootChain contract.
func (c *ContractCaller) SendCheckpoint(signedData []byte, sigs [][3]*big.Int, rootChainAddress common.Address, rootChainInstance *rootchain.Rootchain) error {
	data, err := c.RootChainABI.Pack("submitCheckpoint", signedData, sigs)
	if err != nil {
		Logger.Error("Unable to pack tx for submitCheckpoint", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), rootChainAddress, data)
	if err != nil {
		Logger.Error(errUnableToCreateAuthObj, "error", err)
		return err
	}

	s := make([]string, 0)
	for i := 0; i < len(sigs); i++ {
		s = append(s, fmt.Sprintf("[%s,%s,%s]", sigs[i][0].String(), sigs[i][1].String(), sigs[i][2].String()))
	}

	Logger.Debug("Sending new checkpoint",
		"sigs", strings.Join(s, ","),
		"data", hex.EncodeToString(signedData),
	)

	tx, err := rootChainInstance.SubmitCheckpoint(auth, signedData, sigs)
	if err != nil {
		Logger.Error("Error while submitting checkpoint", "error", err)
		return err
	}

	Logger.Info("Submitted new checkpoint to rootChain successfully", "txHash", tx.Hash().String())

	return nil
}

// StakeFor stakes for a validator
func (c *ContractCaller) StakeFor(val common.Address, stakeAmount *big.Int, feeAmount *big.Int, acceptDelegation bool, stakeManagerAddress common.Address, stakeManagerInstance *stakemanager.Stakemanager) error {
	signerPubKey := GetPubKey()

	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)

	if !bytes.Equal(prefix, signerPubKey[0:1]) {
		Logger.Error("Public key first byte mismatch", "expected", "0x04", "received", signerPubKey[0:1])
		return errors.New("public key first byte mismatch")
	}
	// pack data based on method definition
	data, err := c.StakeManagerABI.Pack("stakeFor", val, stakeAmount, feeAmount, acceptDelegation, signerPubKey.Bytes())
	if err != nil {
		Logger.Error("Unable to pack tx for stakeFor", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), stakeManagerAddress, data)
	if err != nil {
		Logger.Error(errUnableToCreateAuthObj, "error", err)
		return err
	}

	// stake for stake manager
	tx, err := stakeManagerInstance.StakeFor(
		auth,
		val,
		stakeAmount,
		feeAmount,
		acceptDelegation,
		signerPubKey,
	)
	if err != nil {
		Logger.Error("Error while submitting stake", "error", err)
		return err
	}

	Logger.Info("Submitted stakeFor tx successfully", "txHash", tx.Hash().String())

	return nil
}

// ApproveTokens approves pol token for stake
func (c *ContractCaller) ApproveTokens(amount *big.Int, stakeManager common.Address, tokenAddress common.Address, tokenInstance *erc20.Erc20) error {
	data, err := c.PolTokenABI.Pack("approve", stakeManager, amount)
	if err != nil {
		Logger.Error("Unable to pack tx for approve", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), tokenAddress, data)
	if err != nil {
		Logger.Error(errUnableToCreateAuthObj, "error", err)
		return err
	}

	tx, err := tokenInstance.Approve(auth, stakeManager, amount)
	if err != nil {
		Logger.Error("Error while approving tokens", "error", err)
		return err
	}

	Logger.Info("Sent tokens approve tx successfully", "txHash", tx.Hash().String())

	return nil
}
