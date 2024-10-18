package helper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/bits"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/input"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:generate mockgen -destination=./mocks/http_client_mock.go -package=mocks . HTTPClient
type HTTPClient interface {
	Get(string) (resp *http.Response, err error)
}

var (
	Client HTTPClient
)

// GetFromAddress get returns the from address from the context's name
func GetFromAddress(cliCtx client.Context) string {
	ac := address.NewHexCodec()
	fromAddress := cliCtx.GetFromAddress()
	addressString, err := ac.BytesToString(fromAddress.Bytes())
	if err != nil {
		panic(err)
	}
	return addressString
}

func init() {
	Client = &http.Client{}
}

// GetPubObjects returns PubKeySecp256k1 public key
func GetPubObjects(pubkey crypto.PubKey) secp256k1.PubKey {
	var pubObject secp256k1.PubKey

	cdc.MustUnmarshalBinaryBare(pubkey.Bytes(), &pubObject)

	return pubObject
}

// GetMerkleProofList return proof array
// each proof has one byte for direction: 0x0 for left and 0x1 for right
func GetMerkleProofList(proof *merkle.Proof) [][]byte {
	var result [][]byte
	computeHashFromAunts(proof.Index, proof.Total, proof.LeafHash, proof.Aunts, &result)

	return result
}

// AppendBytes appends bytes
func AppendBytes(data ...[]byte) []byte {
	var result []byte
	for _, v := range data {
		result = append(result, v[:]...)
	}

	return result
}

// Use the leafHash and innerHashes to get the root merkle hash.
// If the length of the innerHashes slice isn't exactly correct, the result is nil.
// Recursive impl.
func computeHashFromAunts(index int64, total int64, leafHash []byte, innerHashes [][]byte, newInnerHashes *[][]byte) []byte {
	if index >= total || index < 0 || total <= 0 {
		return nil
	}

	switch total {
	case 1:
		if len(innerHashes) != 0 {
			return nil
		}

		return leafHash
	default:
		if len(innerHashes) == 0 {
			return nil
		}

		numLeft := getSplitPoint(total)
		if index < numLeft {
			leftHash := computeHashFromAunts(index, numLeft, leafHash, innerHashes[:len(innerHashes)-1], newInnerHashes)
			if leftHash == nil {
				return nil
			}

			*newInnerHashes = append(*newInnerHashes, append(rightPrefix, innerHashes[len(innerHashes)-1]...))

			return innerHash(leftHash, innerHashes[len(innerHashes)-1])
		}

		rightHash := computeHashFromAunts(index-numLeft, total-numLeft, leafHash, innerHashes[:len(innerHashes)-1], newInnerHashes)
		if rightHash == nil {
			return nil
		}

		*newInnerHashes = append(*newInnerHashes, append(leftPrefix, innerHashes[len(innerHashes)-1]...))

		return innerHash(innerHashes[len(innerHashes)-1], rightHash)
	}
}

//
// Inner functions
//

// getSplitPoint returns the largest power of 2 less than length
func getSplitPoint(length int64) int64 {
	if length < 1 {
		panic("Trying to split a tree with size < 1")
	}

	uLength := uint(length)
	bitlen := bits.Len(uLength)

	k := 1 << uint(bitlen-1)
	if k == int(length) {
		k >>= 1
	}

	return int64(k)
}

// TODO: make these have a large predefined capacity
var (
	innerPrefix = []byte{1}

	leftPrefix  = []byte{0}
	rightPrefix = []byte{1}
)

// returns tmhash(0x01 || left || right)
func innerHash(left []byte, right []byte) []byte {
	return tmhash.Sum(append(innerPrefix, append(left, right...)...))
}

// ToBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func ToBytes32(x []byte) [32]byte {
	var y [32]byte

	copy(y[:], x)

	return y
}

// GetPowerFromAmount returns power from amount -- note that this will populate amount object
func GetPowerFromAmount(amount *big.Int) (*big.Int, error) {
	decimals18 := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)
	if amount.Cmp(decimals18) == -1 {
		return nil, errors.New("amount must be more than 1 token")
	}

	return amount.Div(amount, decimals18), nil
}

// UnpackSigAndVotes Unpacks Sig and Votes from Tx Payload
func UnpackSigAndVotes(payload []byte, abi abi.ABI) (votes []byte, sigs []byte, checkpointData []byte, err error) {
	// recover Method from signature and ABI
	method := abi.Methods["submitHeaderBlock"]
	decodedPayload := payload[4:]
	inputDataMap := make(map[string]interface{})
	// unpack method inputs
	err = method.Inputs.UnpackIntoMap(inputDataMap, decodedPayload)
	if err != nil {
		return
	}

	if inputDataMap["sigs"] == nil {
		inputDataMap["sigs"] = []byte{}
	}

	if inputDataMap["txData"] == nil {
		inputDataMap["txData"] = []byte{}
	}

	if inputDataMap["vote"] == nil {
		inputDataMap["vote"] = []byte{}
	}

	sigs = inputDataMap["sigs"].([]byte)
	checkpointData = inputDataMap["txData"].([]byte)
	votes = inputDataMap["vote"].([]byte)

	return
}

// EventByID looks up an event by the topic id
func EventByID(abiObject *abi.ABI, sigdata []byte) *abi.Event {
	for _, event := range abiObject.Events {
		if bytes.Equal(event.ID.Bytes(), sigdata) {
			return &event
		}
	}

	return nil
}

// GetHeimdallServerEndpoint returns heimdall server endpoint
func GetHeimdallServerEndpoint(endpoint string) string {
	u, _ := url.Parse(GetConfig().HeimdallServerURL)
	u.Path = path.Join(u.Path, endpoint)

	return u.String()
}

// TODO HV2 - this needs further testing once we have a devnet running. It might be possibly replaced by using the proto services' query clients.
// FetchFromAPI fetches data from any URL
func FetchFromAPI(URL string) (result []byte, err error) {
	resp, err := Client.Get(URL)
	if err != nil {
		return result, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			Logger.Error("Error closing response body:", err)
		}
	}()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}

		return body, nil
	}

	Logger.Debug("Error while fetching data from URL", "status", resp.StatusCode, "URL", URL)

	return result, fmt.Errorf("error while fetching data from url: %s, status: %d", URL, resp.StatusCode)
}

// TODO HV2 - Older version of FetchFromAPI kept for reference, will remove later
/*
// FetchFromAPI fetches data from any URL
func FetchFromAPI(URL string) (result rest.Response, err error) {
	// create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	resp, err := Client.Get(URL)
	if err != nil {
		return result, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			Logger.Error("Error closing response body:", err)
		}
	}()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}

		// unmarshall data from buffer
		var response rest.Response
		if err = cdc.UnmarshalJSON(body, &response); err != nil {
			return result, err
		}

		return response, nil
	}

	Logger.Debug("Error while fetching data from URL", "status", resp.StatusCode, "URL", URL)

	return result, fmt.Errorf("error while fetching data from url: %s, status: %d", URL, resp.StatusCode)
}
*/

// IsPubKeyFirstByteValid checks the validity of the first byte of the public key.
// It must be 0x04 for uncompressed public keys
func IsPubKeyFirstByteValid(pubKey []byte) bool {
	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)

	return bytes.Equal(prefix, pubKey[0:1])
}

// BroadcastTx attempts to generate, sign and broadcast a transaction with the
// given set of messages. It will also simulate gas requirements if necessary.
// It will return an error upon failure.
// HV2 - This function is taken from cosmos-sdk, and it now returns TxResponse as well
func BroadcastTx(clientCtx client.Context, txf clienttx.Factory, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	txf, err := txf.Prepare(clientCtx)
	if err != nil {
		return nil, err
	}

	if txf.SimulateAndExecute() || clientCtx.Simulate {
		if clientCtx.Offline {
			return nil, errors.New("cannot estimate gas in offline mode")
		}

		_, adjusted, err := clienttx.CalculateGas(clientCtx, txf, msgs...)
		if err != nil {
			return nil, err
		}

		txf = txf.WithGas(adjusted)
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", clienttx.GasEstimateResponse{GasEstimate: txf.Gas()})
	}

	if clientCtx.Simulate {
		Logger.Debug("in simulate mode")
		return nil, nil
	}

	tx, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		Logger.Error("error while building unsigned tx", "error", err)
		return nil, err
	}

	if !clientCtx.SkipConfirm {
		// TODO HV2 - create a function
		// func (f Factory) GetTxConfig() client.TxConfig { return f.txConfig }
		// I guess this is no longer needed as this if block is never used
		/*
			encoder := txf.GetTxConfig().TxJSONEncoder()
			if encoder == nil {
				return errors.New("failed to encode transaction: tx json encoder is nil")
			}
		*/

		// Maybe the above code can be replaced with this
		encoder := clientCtx.TxConfig.TxEncoder()

		txBytes, err := encoder(tx.GetTx())
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction: %w", err)
		}

		if err := clientCtx.PrintRaw(txBytes); err != nil {
			Logger.Error("error while printing raw tx", "error", err, "txBytes", txBytes)
		}

		buf := bufio.NewReader(os.Stdin)
		ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf, os.Stderr)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\ncanceled transaction\n", err)
			return nil, err
		}
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, "canceled transaction")
			return nil, nil
		}
	}

	if err = clienttx.Sign(clientCtx.CmdContext, txf, clientCtx.FromName, tx, true); err != nil {
		return nil, err
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(tx.GetTx())
	if err != nil {
		Logger.Error("error while encoding tx", "error", err)
		return nil, err
	}

	// broadcast to a CometBFT node
	res, err := clientCtx.BroadcastTx(txBytes)
	if err != nil {
		Logger.Error("error while broadcasting tx", "error", err)
		return nil, err
	}

	return res, nil
}

// TODO HV2 - I don't think we need this anymore
// Keep it for now, will remove later once everything is working
/*
// BuildAndBroadcastMsgs creates transaction and broadcasts it
func BuildAndBroadcastMsgs(cliCtx client.Context,
	txBldr client.TxBuilder,
	msgs []sdk.Msg,
	testOpts ...*TestOpts,
) (*sdk.TxResponse, error) {
	txBytes, err := GetSignedTxBytes(cliCtx, txBldr, msgs, testOpts...)
	if err != nil {
		return &sdk.TxResponse{}, err
	}
	// just simulate
	if cliCtx.Simulate {
		if len(testOpts) == 0 || testOpts[0].app == nil {
			return &sdk.TxResponse{TxHash: "0x" + hex.EncodeToString(txBytes)}, nil
		}

		// Using cliCtx.GetNode() instead of this
		// m := mock.ABCIApp{
		// 	App: testOpts[0].app,
		// }

		node, err := cliCtx.GetNode()
		if err != nil {
			return &sdk.TxResponse{}, err
		}

		res, err := node.BroadcastTxSync(cliCtx.CmdContext, txBytes)
		return sdk.NewResponseFormatBroadcastTx(res), err
	}
	// broadcast to a CometBFT node
	return BroadcastTxBytes(cliCtx, txBytes, "")
}

// BroadcastTxBytes sends request to cometbft using CLI
func BroadcastTxBytes(cliCtx client.Context, txBytes []byte, mode string) (*sdk.TxResponse, error) {
	Logger.Debug("Broadcasting tx bytes to CometBFT", "txBytes", hex.EncodeToString(txBytes), "txHash", hex.EncodeToString(cmtTypes.Tx(txBytes).Hash()))

	if mode != "" {
		cliCtx.BroadcastMode = mode
	}

	return cliCtx.BroadcastTx(txBytes)
}

// GetSignedTxBytes returns signed tx bytes
func GetSignedTxBytes(cliCtx client.Context,
	txBldr client.TxBuilder,
	msgs []sdk.Msg,
	testOpts ...*TestOpts,
) ([]byte, error) {

	txFactory := tx.Factory{}
	txFactory = txFactory.
		WithChainID(testOpts[0].chainId)

	// just simulate (useful for testing)
	if cliCtx.Simulate {
		if len(testOpts) == 0 || testOpts[0].chainId == "" {
			return nil, nil
		}

		// We are no longer able to set ChainID
		// txBldr = txBldr.WithChainID(testOpts[0].chainId)

		return txBldr.BuildAndSign(GetPrivKey(), msgs)
	}

	// TODO HV2 - I don't think we need this anymore
	// txBldr, err := PrepareTxBuilder(cliCtx, txBldr)
	// if err != nil {
	// 	return nil, err
	// }

	fromName := cliCtx.GetFromName()
	if fromName == "" {
		return txBldr.BuildAndSign(GetPrivKey(), msgs)
	}

	if !cliCtx.SkipConfirm {
		stdSignMsg, err := txBldr.BuildSignMsg(msgs)
		if err != nil {
			return nil, err
		}

		json := cliCtx.Codec.MustMarshalJSON(stdSignMsg)

		_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", json)

		buf := bufio.NewReader(os.Stdin)

		ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf)
		if err != nil || !ok {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", "cancelled transaction")
			return nil, err
		}
	}

	passphrase, err := keys.GetPassphrase(fromName)
	if err != nil {
		return nil, err
	}
	// build and sign the transaction
	return txBldr.BuildAndSignWithPassphrase(fromName, passphrase, msgs)
}
*/

func GetSignature(signMode signing.SignMode, accSeq uint64) signing.SignatureV2 {
	cosmosPrivKey := cosmossecp256k1.PrivKey{Key: GetPrivKey()}

	sig := signing.SignatureV2{
		PubKey: cosmosPrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signMode,
		},
		Sequence: accSeq,
	}

	return sig
}
