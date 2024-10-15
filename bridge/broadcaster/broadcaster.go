package broadcaster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/viper"
)

// TxBroadcaster is used to broadcast transaction to each chain
type TxBroadcaster struct {
	logger log.Logger

	CliCtx client.Context

	heimdallMutex sync.Mutex
	maticMutex    sync.Mutex

	lastSeqNo uint64
	accNum    uint64
}

// NewTxBroadcaster creates a new instance of TxBroadcaster
func NewTxBroadcaster(cdc codec.Codec) *TxBroadcaster {
	cliCtx := client.Context{}.WithCodec(cdc)
	cliCtx.BroadcastMode = flags.BroadcastSync

	// current address
	address := helper.GetAddress()

	var account sdk.AccountI

	account, err := util.GetAccount(cliCtx, common.BytesToAddress(address).String())
	if err != nil {
		panic("Error connecting to rest-server, please start server before bridge.")
	}

	return &TxBroadcaster{
		logger:    log.NewNopLogger().With("module", "txBroadcaster"),
		CliCtx:    cliCtx,
		lastSeqNo: account.GetSequence(),
		accNum:    account.GetAccountNumber(),
	}
}

// BroadcastToHeimdall broadcast to heimdall
func (tb *TxBroadcaster) BroadcastToHeimdall(msg sdk.Msg, event interface{}, testOpts ...*helper.TestOpts) (*sdk.TxResponse, error) {
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
	// TODO HV2 - what should be the gas limit and fee amount? How?
	/*
		txBldr.SetFeeAmount(feeAmt)
		txBldr.SetGasLimit(gas)
	*/

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
	if err != nil || txResponse.Code != uint32(abci.CodeTypeOK) {
		tb.logger.Error("Error while broadcasting the heimdall transaction", "error", err, "txResponse", txResponse.Code)

		// current address
		address := helper.GetAddress()

		// fetch from APIs
		account, errAcc := util.GetAccount(tb.CliCtx, string(address[:]))
		if errAcc != nil {
			tb.logger.Error("Error fetching account from rest-api", "url", helper.GetHeimdallServerEndpoint(fmt.Sprintf(util.AccountDetailsURL, helper.GetAddress())))
			return txResponse, errAcc
		}

		// update seqNo for safety
		tb.lastSeqNo = account.GetSequence()

		return txResponse, err
	}

	txHash := txResponse.TxHash

	tb.logger.Info("Tx sent on heimdall", "txHash", txHash, "accSeq", tb.lastSeqNo, "accNum", tb.accNum)
	tb.logger.Debug("Tx successful on heimdall", "txResponse", txResponse)
	// increment account sequence
	tb.lastSeqNo += 1

	return txResponse, nil
}

// BroadcastToMatic broadcast to matic
func (tb *TxBroadcaster) BroadcastToMatic(msg ethereum.CallMsg) error {
	tb.maticMutex.Lock()
	defer tb.maticMutex.Unlock()

	// get matic client
	maticClient := helper.GetPolygonPosClient()

	// get auth
	auth, err := helper.GenerateAuthObj(maticClient, *msg.To, msg.Data)

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
	if err := maticClient.SendTransaction(ctx, signedTx); err != nil {
		tb.logger.Error("Error while broadcasting the transaction to polygonposchain", "error", err)
		return err
	}

	return nil
}

// BroadcastToRootchain broadcast to rootchain
func (tb *TxBroadcaster) BroadcastToRootchain() {}
