package helper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
)

func GenerateAuthObj(client *ethclient.Client, address common.Address, data []byte) (auth *bind.TransactOpts, err error) {
	ctx := context.Background()

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

	// Get the configured gas fee cap and tip cap.
	configGasFeeCap := GetConfig().MainChainGasFeeCap
	if configGasFeeCap <= 0 {
		configGasFeeCap = DefaultMainChainGasFeeCap
	}

	configGasTipCap := GetConfig().MainChainGasTipCap
	if configGasTipCap <= 0 {
		configGasTipCap = DefaultMainChainGasTipCap
	}

	// Get the latest block to retrieve baseFee.
	latestBlock, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		Logger.Error("unable to fetch latest block", "error", err)
		return
	}

	baseFee := latestBlock.BaseFee()
	if baseFee == nil {
		err = errors.New("baseFee is nil, EIP-1559 not supported")
		Logger.Error("EIP-1559 not supported on this chain", "error", err)
		return
	}

	// Get suggested tip cap from the network.
	suggestedTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		Logger.Error("unable to fetch suggested gas tip cap", "error", err)
		return
	}

	// Use the lower of suggested tip or configured tip cap to prioritize lower fees.
	// The configured tip cap acts as a maximum we're willing to pay.
	gasTipCap := suggestedTipCap
	if gasTipCap.Cmp(big.NewInt(configGasTipCap)) > 0 {
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

	// Use PendingNonceAt to account for pending transactions in the mempool.
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return
	}

	callMsg.From = fromAddress
	gasLimit, err := client.EstimateGas(ctx, callMsg)
	if err != nil {
		Logger.Error("unable to estimate gas", "error", err)
		return
	}

	chainId, err := client.ChainID(ctx)
	if err != nil {
		Logger.Error("unable to fetch ChainID", "error", err)
		return
	}

	auth, err = bind.NewKeyedTransactorWithChainID(ecdsaPrivateKey, chainId)
	if err != nil {
		Logger.Error("unable to create auth object", "error", err)
		return
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasLimit = gasLimit
	auth.GasFeeCap = gasFeeCap
	auth.GasTipCap = gasTipCap

	Logger.Debug("created EIP-1559 transaction auth",
		"nonce", nonce,
		"gasLimit", gasLimit,
		"gasFeeCap", gasFeeCap,
		"gasTipCap", gasTipCap,
		"baseFee", baseFee,
	)

	return
}

// SendCheckpoint sends checkpoint to rootChain contract.
func (c *ContractCaller) SendCheckpoint(signedData []byte, sigs [][3]*big.Int, rootChainAddress common.Address, rootChainInstance *rootchain.Rootchain) error {
	data, err := c.RootChainABI.Pack("submitCheckpoint", signedData, sigs)
	if err != nil {
		Logger.Error("unable to pack tx for submitCheckpoint", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), rootChainAddress, data)
	if err != nil {
		Logger.Error("unable to create auth object", "error", err)
		return err
	}

	s := make([]string, 0)
	for i := 0; i < len(sigs); i++ {
		s = append(s, fmt.Sprintf("[%s,%s,%s]", sigs[i][0].String(), sigs[i][1].String(), sigs[i][2].String()))
	}

	Logger.Debug("sending new checkpoint",
		"sigs", strings.Join(s, ","),
		"data", hex.EncodeToString(signedData),
	)

	tx, err := rootChainInstance.SubmitCheckpoint(auth, signedData, sigs)
	if err != nil {
		Logger.Error("error while submitting checkpoint", "error", err)
		return err
	}

	Logger.Info("submitted new checkpoint to rootChain successfully", "txHash", tx.Hash().String())

	return nil
}

// StakeFor stakes for a validator
func (c *ContractCaller) StakeFor(val common.Address, stakeAmount *big.Int, feeAmount *big.Int, acceptDelegation bool, stakeManagerAddress common.Address, stakeManagerInstance *stakemanager.Stakemanager) error {
	signerPubKey := GetPubKey()

	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)

	if !bytes.Equal(prefix, signerPubKey[0:1]) {
		Logger.Error("public key first byte mismatch", "expected", "0x04", "received", signerPubKey[0:1])
		return errors.New("public key first byte mismatch")
	}
	// pack data based on method definition
	data, err := c.StakeManagerABI.Pack("stakeFor", val, stakeAmount, feeAmount, acceptDelegation, signerPubKey.Bytes())
	if err != nil {
		Logger.Error("unable to pack tx for stakeFor", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), stakeManagerAddress, data)
	if err != nil {
		Logger.Error("unable to create auth object", "error", err)
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
		Logger.Error("error while submitting stake", "error", err)
		return err
	}

	Logger.Info("submitted stake successfully", "txHash", tx.Hash().String())

	return nil
}

// ApproveTokens approves pol token for stake
func (c *ContractCaller) ApproveTokens(amount *big.Int, stakeManager common.Address, tokenAddress common.Address, tokenInstance *erc20.Erc20) error {
	data, err := c.PolTokenABI.Pack("approve", stakeManager, amount)
	if err != nil {
		Logger.Error("unable to pack tx for approve", "error", err)
		return err
	}

	auth, err := GenerateAuthObj(GetMainClient(), tokenAddress, data)
	if err != nil {
		Logger.Error("unable to create auth object", "error", err)
		return err
	}

	tx, err := tokenInstance.Approve(auth, stakeManager, amount)
	if err != nil {
		Logger.Error("error while approving approve", "error", err)
		return err
	}

	Logger.Info("Sent approve tx successfully", "txHash", tx.Hash().String())

	return nil
}
