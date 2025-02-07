package engine

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/rand"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestEnginePayloadProcessing(t *testing.T) {

	t.SkipNow() // FIXME: make this work with the new docker-ized setup

	engConf := getEngineConfigs(0)[0]

	_, latest, err := getLatestBlockInfo(context.TODO(), engConf.ethUrl)
	require.NoError(t, err)

	// The engine in on the secure port
	ec, err := NewEngineClient(engConf.engineUrl, engConf.jwtFile)
	require.NoError(t, err)
	defer ec.Close()

	state := ForkChoiceState{
		HeadHash:           *latest,
		SafeBlockHash:      *latest,
		FinalizedBlockHash: common.Hash{},
	}

	w := Withdrawal{
		Index:     "0x1",
		Validator: "0x1",
		Address:   common.Address{}.Hex(),
		Amount:    "0x1",
	}

	attrs := PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{},
		SuggestedFeeRecipient: common.Address{},
		Withdrawals:           []*Withdrawal{&w},
		//ParentBeaconBlockRoot: common.Hash{},
	}

	fmt.Printf("1-> ForkchoiceUpdatedV2: state=%v, attrs=%v\n", state, attrs)

	fcu1, err := ec.ForkchoiceUpdatedV2(&state, &attrs)
	require.NoError(t, err)
	require.NotNil(t, fcu1)
	require.Equal(t, "VALID", fcu1.PayloadStatus.Status)
	require.NotEmpty(t, fcu1.PayloadId)

	fmt.Printf("return: %v\n\n", fcu1)

	fmt.Printf("2-> GetPayloadV2: %v\n", fcu1.PayloadId)
	payload1, err := ec.GetPayloadV2(fcu1.PayloadId)
	require.NoError(t, err)
	require.NotNil(t, payload1)

	fmt.Printf("return: %v\n\n", payload1)

	fmt.Printf("3-> NewPayloadV2: %v\n", payload1.ExecutionPayload)

	newPayload, err := ec.NewPayloadV2(payload1.ExecutionPayload)
	require.NoError(t, err)
	require.NotNil(t, newPayload)
	require.Equal(t, "VALID", newPayload.Status)
	require.Empty(t, newPayload.ValidationError)
	require.NotEmpty(t, newPayload.LatestValidHash)

	fmt.Printf("return: %v\n\n", newPayload)

	// forkchoice with new hashes
	state.FinalizedBlockHash = state.HeadHash
	state.HeadHash = common.HexToHash(newPayload.LatestValidHash)
	state.SafeBlockHash = common.HexToHash(newPayload.LatestValidHash)

	// what to do with payload attributes? what is parent beacon block root?
	attrs.Timestamp = hexutil.Uint64(time.Now().UnixMilli())

	fmt.Printf("4-> ForkchoiceUpdatedV2: state=%v, attrs=%v\n", state, attrs)

	fcu2, err := ec.ForkchoiceUpdatedV2(&state, &attrs)
	require.NoError(t, err)
	require.NotNil(t, fcu2)
	require.Equal(t, "VALID", fcu2.PayloadStatus.Status)
	require.NotEmpty(t, fcu2.PayloadId)

	fmt.Printf("return: %v\n\n", fcu2)
}

func TestMultiEngineExecution(t *testing.T) {

	t.SkipNow() // FIXME: make this work with the new docker-ized setup

	engines := getEngineConfigs(4)

	// require that they are all at same block
	checkEngines(t, engines)

	// -----
	// send a transaction to engines[0]
	eth, err := ethclient.DialContext(context.TODO(), engines[0].ethUrl)
	require.NoError(t, err)

	privateKey, err := crypto.HexToECDSA("26e86e45f6fc45ec6e2ecd128cec80fa1d1505e5507dcd2ae58c3130a7a97b48")
	require.NoError(t, err)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	require.Equal(t, "0x67b1d87101671b127f5f8714789C7192f7ad340e", fromAddress.Hex())

	nonce, err := eth.PendingNonceAt(context.TODO(), fromAddress)
	require.NoError(t, err)

	value := big.NewInt(1000000000000000000 + rand.Int63n(1000000000000000000))
	gasLimit := uint64(21000)

	gasPrice, err := eth.SuggestGasPrice(context.TODO())
	require.NoError(t, err)

	toAddress := common.HexToAddress("0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B")

	toBalance, err := eth.BalanceAt(context.TODO(), toAddress, nil)
	require.NoError(t, err)

	expectedBalanceAfter := big.NewInt(0).Add(toBalance, value)

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

	chainID, err := eth.NetworkID(context.TODO())
	require.NoError(t, err)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	require.NoError(t, err)

	err = eth.SendTransaction(context.TODO(), signedTx)
	require.NoError(t, err)

	// -----

	_, latest, err := getLatestBlockInfo(context.TODO(), engines[0].ethUrl)
	require.NoError(t, err)

	// The engine in on the secure port
	ec0, err := NewEngineClient(engines[0].engineUrl, engines[0].jwtFile)
	require.NoError(t, err)
	defer ec0.Close()

	state := ForkChoiceState{
		HeadHash:           *latest,
		SafeBlockHash:      *latest,
		FinalizedBlockHash: common.Hash{},
	}

	w := Withdrawal{
		Index:     "0x1",
		Validator: "0x1",
		Address:   common.Address{}.Hex(),
		Amount:    "0x1",
	}

	attrs := PayloadAttributes{
		Timestamp:             hexutil.Uint64(time.Now().UnixMilli()),
		PrevRandao:            common.Hash{},
		SuggestedFeeRecipient: common.Address{},
		Withdrawals:           []*Withdrawal{&w},
		//ParentBeaconBlockRoot: common.Hash{},
	}

	fcu1, err := ec0.ForkchoiceUpdatedV2(&state, &attrs)
	require.NoError(t, err)
	require.NotNil(t, fcu1)
	require.Equal(t, "VALID", fcu1.PayloadStatus.Status)
	require.NotEmpty(t, fcu1.PayloadId)

	payload1, err := ec0.GetPayloadV2(fcu1.PayloadId)
	require.NoError(t, err)
	require.NotNil(t, payload1)

	require.Equal(t, 1, len(payload1.ExecutionPayload.Transactions))

	// for each engine, execute the proposed block
	for _, e := range engines {
		engine, err := NewEngineClient(e.engineUrl, e.jwtFile)
		require.NoError(t, err)
		defer engine.Close()

		newPayload, err := engine.NewPayloadV2(payload1.ExecutionPayload)
		require.NoError(t, err)
		require.NotNil(t, newPayload)
		require.Equal(t, "VALID", newPayload.Status)
		require.Empty(t, newPayload.ValidationError)
		require.NotEmpty(t, newPayload.LatestValidHash)

		// forkchoice with new hashes
		state.FinalizedBlockHash = state.HeadHash
		state.HeadHash = common.HexToHash(newPayload.LatestValidHash)
		state.SafeBlockHash = common.HexToHash(newPayload.LatestValidHash)

		// apparently not used in block hash, but must be distinct from previous attrs
		attrs.Timestamp = hexutil.Uint64(time.Now().UnixMilli())

		fcu2, err := engine.ForkchoiceUpdatedV2(&state, &attrs)
		require.NoError(t, err)
		require.NotNil(t, fcu2)
		require.Equal(t, "VALID", fcu2.PayloadStatus.Status)
		require.NotEmpty(t, fcu2.PayloadId)
	}

	// All engines should have advanced and be at same block num/hash
	checkEngines(t, engines)

	// All toAddress balances should be expected value
	for _, e := range engines {
		c, err := ethclient.DialContext(context.TODO(), e.ethUrl)
		require.NoError(t, err)
		defer c.Close()

		newBalance, err := c.BalanceAt(context.TODO(), toAddress, nil)
		require.NoError(t, err)

		require.Equal(t, expectedBalanceAfter, newBalance)
	}

}

func TestEnginesAtSameBlock(t *testing.T) {
	t.SkipNow() // FIXME: make this work with the new docker-ized setup
	checkEngines(t, getEngineConfigs(4))
}

type EngineConfig struct {
	index     int
	ethUrl    string
	engineUrl string
	jwtFile   string
}

func getEngineConfigs(count int) []EngineConfig {
	var engines []EngineConfig
	for i := 0; i < count; i++ {
		engines = append(engines, EngineConfig{
			index:     i,
			ethUrl:    getEthUrl(i),
			engineUrl: getEngineUrl(i),
			jwtFile:   getJwtPath(i),
		})
	}
	return engines
}

func getEngineUrl(idx int) string {
	return fmt.Sprintf("http://127.0.0.1:855%d", idx)
}

func getEthUrl(idx int) string {
	return fmt.Sprintf("http://127.0.0.1:854%d", idx)
}

func getJwtPath(idx int) string {
	return fmt.Sprintf("./build/erigon%d/jwt.hex", idx)
}

func checkEngines(t *testing.T, engines []EngineConfig) {
	require.Greater(t, len(engines), 2)
	var nums []*big.Int
	var hashes []*common.Hash
	for i := 0; i < len(engines); i++ {
		e := engines[i]
		num, hash, err := getLatestBlockInfo(context.TODO(), e.ethUrl)
		require.NoError(t, err)
		nums = append(nums, num)
		hashes = append(hashes, hash)
	}
	for i := 0; i < len(nums)-1; i++ {
		j := i + 1
		require.Equal(t, nums[i], nums[j], "comparing %d and %d", i, j)
		require.Equal(t, hashes[i], hashes[j], "comparing %d and %d", i, j)
	}
}

func getLatestBlockInfo(ctx context.Context, ethUrl string) (*big.Int, *common.Hash, error) {
	eth, err := ethclient.Dial(ethUrl)
	if err != nil {
		return nil, nil, err
	}
	defer eth.Close()

	n, err := eth.BlockNumber(ctx)
	if err != nil {
		return nil, nil, err
	}
	number := big.NewInt(int64(n))

	block, err := eth.BlockByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	hash := block.Hash()
	return number, &hash, err
}
