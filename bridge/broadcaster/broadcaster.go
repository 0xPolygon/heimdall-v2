package broadcaster

import (
	"context"
	"fmt"
	"sync"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	cometTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

// TxBroadcaster is used to broadcast transaction to each chain
type TxBroadcaster struct {
	logger log.Logger

	CliCtx client.Context

	heimdallMutex sync.Mutex
	borMutex      sync.Mutex

	lastSeqNo uint64
	accNum    uint64
}

// NewTxBroadcaster creates a new instance of TxBroadcaster
func NewTxBroadcaster(cdc codec.Codec) *TxBroadcaster {
	cliCtx := client.Context{}.WithCodec(cdc)
	cliCtx.BroadcastMode = flags.BroadcastSync

	// current address
	address := helper.GetAddress()
	ac := addressCodec.NewHexCodec()
	addressString, err := ac.BytesToString(address)
	if err != nil {
		panic("Error converting address to string")
	}

	account, err := util.GetAccount(cliCtx, addressString)
	if err != nil {
		panic(fmt.Sprintf("Error connecting to rest-server, please start server before bridge. Error: %v", err))
	}

	return &TxBroadcaster{
		logger:    log.NewNopLogger().With("module", "txBroadcaster"),
		CliCtx:    cliCtx,
		lastSeqNo: account.GetSequence(),
		accNum:    account.GetAccountNumber(),
	}
}

// BroadcastToHeimdall broadcast to heimdall
func (tb *TxBroadcaster) BroadcastToHeimdall(msg sdk.Msg, event interface{}) (*sdk.TxResponse, error) {
	tb.heimdallMutex.Lock()
	defer tb.heimdallMutex.Unlock()
	defer util.LogElapsedTimeForStateSyncedEvent(event, "BroadcastToHeimdall", time.Now())

	txCfg := tb.CliCtx.TxConfig

	txBldr := txCfg.NewTxBuilder()
	err := txBldr.SetMsgs(msg)
	if err != nil {
		return &sdk.TxResponse{}, err
	}

	signMode, err := authsign.APISignModeToInternal(txCfg.SignModeHandler().DefaultMode())
	if err != nil {
		return &sdk.TxResponse{}, err
	}

	err = txBldr.SetSignatures(helper.GetSignature(signMode, tb.lastSeqNo))
	if err != nil {
		return &sdk.TxResponse{}, err
	}
	txBldr.SetMemo(viper.GetString("memo"))

	txBldr.SetGasLimit(uint64(cometTypes.DefaultBlockParams().MaxGas))
	txBldr.SetFeeAmount(ante.DefaultFeeWantedPerTx)

	// create a factory
	txf := clienttx.Factory{}.
		WithTxConfig(txCfg).
		WithAccountRetriever(tb.CliCtx.AccountRetriever).
		WithChainID(tb.CliCtx.ChainID).
		WithSignMode(signMode).
		WithAccountNumber(tb.accNum).
		WithSequence(tb.lastSeqNo).
		WithKeybase(tb.CliCtx.Keyring)

	// setting this to true to as the if block in BroadcastTx
	// might cause a cancelled transaction.
	tb.CliCtx.SkipConfirm = true

	txResponse, err := helper.BroadcastTx(tb.CliCtx, txf, msg)

	// Check for an error from broadcasting the transaction
	if err != nil {
		tb.logger.Error("Error while broadcasting the heimdall transaction", "error", err)

		// Handle fetching account and updating seqNo
		if handleAccountUpdateErr := updateAccountSequence(tb); handleAccountUpdateErr != nil {
			return txResponse, handleAccountUpdateErr
		}

		return txResponse, err
	}

	// Now check if the transaction response is not okay
	if txResponse.Code != abci.CodeTypeOK {
		tb.logger.Error("Transaction response returned a non-ok code", "txResponseCode", txResponse.Code)

		// Handle fetching account and updating seqNo
		if handleAccountUpdateErr := updateAccountSequence(tb); handleAccountUpdateErr != nil {
			return txResponse, handleAccountUpdateErr
		}

		return txResponse, fmt.Errorf("broadcast succeeded but received non-ok response code: %d", txResponse.Code)
	}

	txHash := txResponse.TxHash

	tb.logger.Info("Tx sent on heimdall", "txHash", txHash, "accSeq", tb.lastSeqNo, "accNum", tb.accNum, "txResponse", txResponse)
	// increment account sequence
	tb.lastSeqNo += 1

	return txResponse, nil
}

// Helper function to update account sequence
func updateAccountSequence(tb *TxBroadcaster) error {
	// current address
	address := helper.GetAddress()
	ac := addressCodec.NewHexCodec()
	addressString, err := ac.BytesToString(address)
	if err != nil {
		return fmt.Errorf("error converting address to string: %v", err)
	}

	// fetch from APIs
	account, errAcc := util.GetAccount(tb.CliCtx, addressString)
	if errAcc != nil {
		tb.logger.Error("Error fetching account from rest-api", "url", helper.GetHeimdallServerEndpoint(fmt.Sprintf(util.AccountDetailsURL, helper.GetAddress())))
		return errAcc
	}

	// update seqNo for safety
	tb.lastSeqNo = account.GetSequence()

	return nil
}

// BroadcastToBorChain broadcasts a msg to bor chain
func (tb *TxBroadcaster) BroadcastToBorChain(msg ethereum.CallMsg) error {
	tb.borMutex.Lock()
	defer tb.borMutex.Unlock()

	// get bor client
	borClient := helper.GetBorClient()

	// get auth
	auth, err := helper.GenerateAuthObj(borClient, *msg.To, msg.Data)

	if err != nil {
		tb.logger.Error("Error generating auth object", "error", err)
		return err
	}

	// Create the transaction, sign it and schedule it for execution
	rawTx := types.NewTx(&types.LegacyTx{
		Nonce:    auth.Nonce.Uint64(),
		To:       msg.To,
		Value:    msg.Value,
		Gas:      auth.GasLimit,
		GasPrice: auth.GasPrice,
		Data:     msg.Data,
	})

	// signer
	signedTx, err := auth.Signer(auth.From, rawTx)
	if err != nil {
		tb.logger.Error("Error signing the transaction", "error", err)
		return err
	}

	tb.logger.Info("Sending transaction to bor", "txHash", signedTx.Hash())

	// create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), helper.GetConfig().BorRPCTimeout)
	defer cancel()

	// broadcast transaction
	if err = borClient.SendTransaction(ctx, signedTx); err != nil {
		tb.logger.Error("Error while broadcasting the transaction to bor chain", "error", err)
		return err
	}

	return nil
}

// BroadcastToRootchain broadcast to rootchain
func (tb *TxBroadcaster) BroadcastToRootchain() {}
