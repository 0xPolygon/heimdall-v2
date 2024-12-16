package keeper

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// Keeper stores all bor module related data
type Keeper struct {
	cdc            codec.BinaryCodec
	storeService   store.KVStoreService
	authority      string
	ck             types.ChainManagerKeeper
	sk             types.StakeKeeper
	contractCaller helper.IContractCaller

	Schema       collections.Schema
	spans        collections.Map[uint64, types.Span]
	latestSpan   collections.Item[uint64]
	lastEthBlock collections.Item[[]byte]
	Params       collections.Item[types.Params]
}

// NewKeeper creates a new instance of the bor Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	authority string,
	chainKeeper types.ChainManagerKeeper,
	stakingKeeper types.StakeKeeper,
	caller helper.IContractCaller,
) Keeper {

	bz, err := address.NewHexCodec().StringToBytes(authority)
	if err != nil {
		panic(fmt.Errorf("invalid bor authority address: %w", err))
	}

	// ensure only gov has the authority to update the params
	if !bytes.Equal(bz, authtypes.NewModuleAddress(govtypes.ModuleName)) {
		panic(fmt.Errorf("invalid bor authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:            cdc,
		storeService:   storeService,
		authority:      authority,
		ck:             chainKeeper,
		sk:             stakingKeeper,
		contractCaller: caller,
		spans:          collections.NewMap(sb, types.SpanPrefixKey, "span", collections.Uint64Key, codec.CollValue[types.Span](cdc)),
		latestSpan:     collections.NewItem(sb, types.LastSpanIDKey, "lastSpanId", collections.Uint64Value),
		lastEthBlock:   collections.NewItem(sb, types.LastProcessedEthBlock, "lastEthBlock", collections.BytesValue),
		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema
	return k
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns x/bor module's authority
func (k Keeper) GetAuthority() string {
	return k.authority
}

// TODO HV2: delete this function if not needed

// GetSpanKey appends prefix to start block
func GetSpanKey(id uint64) []byte {
	return append(types.SpanPrefixKey, sdk.Uint64ToBigEndian(id)...)
}

// AddNewSpan adds new span for bor to store and updates last span
func (k *Keeper) AddNewSpan(ctx context.Context, span *types.Span) error {
	logger := k.Logger(ctx)
	if err := k.AddNewRawSpan(ctx, span); err != nil {
		// TODO HV2: should we panic here instead ?
		logger.Error("error setting span", "error", err, "span", span)
		return err
	}

	return k.UpdateLastSpan(ctx, span.Id)
}

// AddNewRawSpan adds new span for bor to store
func (k *Keeper) AddNewRawSpan(ctx context.Context, span *types.Span) error {
	return k.spans.Set(ctx, span.Id, *span)
}

// GetSpan fetches span indexed by id from store
func (k *Keeper) GetSpan(ctx context.Context, id uint64) (types.Span, error) {
	ok, err := k.spans.Has(ctx, id)
	if err != nil {
		return types.Span{}, err
	}

	// If we are starting from 0 there will be no spanKey present
	if !ok {
		return types.Span{}, fmt.Errorf("span not found for id: %v", id)
	}

	span, err := k.spans.Get(ctx, id)
	if err != nil {
		return types.Span{}, err
	}

	return span, nil
}

func (k *Keeper) HasSpan(ctx context.Context, id uint64) (bool, error) {
	return k.spans.Has(ctx, id)
}

// GetAllSpans fetches all spans indexed by id from store
func (k *Keeper) GetAllSpans(ctx context.Context) ([]*types.Span, error) {
	logger := k.Logger(ctx)

	// get span iterator
	iter, err := k.spans.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func(iter collections.Iterator[uint64, types.Span]) {
		err := iter.Close()
		if err != nil {
			logger.Error("error closing span iterator", "err", err)
			return
		}
	}(iter)

	res, err := iter.Values()
	if err != nil {
		logger.Error("error getting spans from iterator", "err", err)
		return nil, err
	}

	spans := make([]*types.Span, 0, len(res))
	for _, span := range res {
		spans = append(spans, &span)
	}
	return spans, err
}

// GetLastSpan fetches last span from store
func (k *Keeper) GetLastSpan(ctx context.Context) (types.Span, error) {
	ok, err := k.latestSpan.Has(ctx)
	if err != nil {
		return types.Span{}, err
	}

	if !ok {
		return types.Span{}, fmt.Errorf("last span not found")
	}

	// get last span id
	lastSpanId, err := k.latestSpan.Get(ctx)
	if err != nil {
		return types.Span{}, err
	}

	return k.GetSpan(ctx, lastSpanId)
}

// FreezeSet freezes validator set for next span
func (k *Keeper) FreezeSet(ctx sdk.Context, id uint64, startBlock uint64, endBlock uint64, borChainID string, seed common.Hash) error {
	// select next producers
	newProducers, err := k.SelectNextProducers(ctx, seed)
	if err != nil {
		return err
	}

	// increment last eth block
	if err := k.IncrementLastEthBlock(ctx); err != nil {
		return err
	}

	valSet, err := k.sk.GetValidatorSet(ctx)
	if err != nil {
		return err
	}

	// generate new span
	newSpan := &types.Span{
		Id:                id,
		StartBlock:        startBlock,
		EndBlock:          endBlock,
		ValidatorSet:      valSet,
		SelectedProducers: newProducers,
		BorChainId:        borChainID,
	}

	return k.AddNewSpan(ctx, newSpan)
}

// SelectNextProducers selects producers for next span
func (k *Keeper) SelectNextProducers(ctx context.Context, seed common.Hash) ([]staketypes.Validator, error) {
	// spanEligibleVals are current validators who are not getting deactivated in between next span
	spanEligibleVals := k.sk.GetSpanEligibleValidators(ctx)
	params, err := k.FetchParams(ctx)
	if err != nil {
		return nil, err
	}
	producerCount := params.ProducerCount

	// if producers to be selected is more than current validators no need to select/shuffle
	if len(spanEligibleVals) <= int(producerCount) {
		return spanEligibleVals, nil
	}

	// select next producers using seed as block header hash
	newProducersIds, err := selectNextProducers(seed, spanEligibleVals, producerCount)
	if err != nil {
		return nil, err
	}

	vals := make([]staketypes.Validator, 0, len(newProducersIds))

	IDToPower := make(map[uint64]uint64)
	for _, ID := range newProducersIds {
		IDToPower[ID] = IDToPower[ID] + 1
	}

	for key, value := range IDToPower {
		val, err := k.sk.GetValidatorFromValID(ctx, key)
		if err != nil {
			return nil, err
		}

		val.VotingPower = int64(value)
		vals = append(vals, val)
	}

	// sort by address
	vals = types.SortValidatorByAddress(vals)

	return vals, nil
}

// UpdateLastSpan updates the last span
func (k *Keeper) UpdateLastSpan(ctx context.Context, id uint64) error {
	return k.latestSpan.Set(ctx, id)
}

// IncrementLastEthBlock increment last eth block
func (k *Keeper) IncrementLastEthBlock(ctx context.Context) error {
	lastEthBlock := big.NewInt(0)
	ok, err := k.lastEthBlock.Has(ctx)
	if err != nil {
		return err
	}
	if ok {
		lastEthBlockBytes, err := k.lastEthBlock.Get(ctx)
		if err != nil {
			return err
		}
		lastEthBlock = lastEthBlock.SetBytes(lastEthBlockBytes)
	}

	return k.lastEthBlock.Set(ctx, lastEthBlock.Add(lastEthBlock, big.NewInt(1)).Bytes())
}

// SetLastEthBlock sets last eth block number
func (k *Keeper) SetLastEthBlock(ctx context.Context, blockNumber *big.Int) error {
	return k.lastEthBlock.Set(ctx, blockNumber.Bytes())
}

// GetLastEthBlock gets last processed Eth block for seed
func (k *Keeper) GetLastEthBlock(ctx context.Context) (*big.Int, error) {
	lastEthBlock := big.NewInt(0)
	ok, err := k.lastEthBlock.Has(ctx)
	if err != nil {
		return nil, err
	}

	if ok {
		lastEthBlockBytes, err := k.lastEthBlock.Get(ctx)
		if err != nil {
			return nil, err
		}
		lastEthBlock = lastEthBlock.SetBytes(lastEthBlockBytes)
	}

	return lastEthBlock, nil
}

// FetchNextSpanSeed gets the eth block hash which serves as seed for random selection of producer set
// for the next span
func (k *Keeper) FetchNextSpanSeed(ctx context.Context) (common.Hash, error) {
	logger := k.Logger(ctx)
	lastEthBlock, err := k.GetLastEthBlock(ctx)
	if err != nil {
		logger.Error("error fetching last eth block while calculating next span seed", "error", err)
		return common.Hash{}, err
	}

	// increment last processed header block number
	newEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
	logger.Debug("newEthBlock to generate seed", "newEthBlock", newEthBlock)

	// fetch block header from mainchain
	blockHeader, err := k.contractCaller.GetMainChainBlock(newEthBlock)
	if err != nil {
		logger.Error("error fetching block header from mainchain while calculating next span seed", "error", err)
		return common.Hash{}, err
	}

	return blockHeader.Hash(), nil
}

// SetParams sets the bor module's parameters.
func (k *Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

// FetchParams gets the bor module's parameters.
func (k *Keeper) FetchParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}
	return params, nil
}
