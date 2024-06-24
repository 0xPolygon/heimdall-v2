package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Keeper stores all chainmanager related data
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	ck           types.ChainManagerKeeper
	sk           types.StakeKeeper
	// TODO HV2: uncomment when contractCaller is implemented
	// contractCaller helper.ContractCaller

	Schema       collections.Schema
	spans        collections.Map[uint64, *types.Span]
	latestSpan   collections.Item[uint64]
	lastEthBlock collections.Item[[]byte]
	Params       collections.Item[types.Params]
}

// NewKeeper creates a new instance of the bor Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	chainKeeper types.ChainManagerKeeper,
	stakingKeeper types.StakeKeeper,
	// caller helper.ContractCaller,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		ck:           chainKeeper,
		sk:           stakingKeeper,
		// TODO HV2: uncomment when contractCaller is implemented
		// contractCaller: caller,
		spans:        collections.NewMap(sb, types.SpanPrefixKey, "span", collections.Uint64Key, codec.CollInterfaceValue[*types.Span](cdc)),
		latestSpan:   collections.NewItem(sb, types.LastSpanIDKey, "lastSpanId", collections.Uint64Value),
		lastEthBlock: collections.NewItem(sb, types.LastProcessedEthBlock, "lastEthBlock", collections.BytesValue),
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
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

// GetSpanKey appends prefix to start block
func GetSpanKey(id uint64) []byte {
	return append(types.SpanPrefixKey, sdk.Uint64ToBigEndian(id)...)
}

// AddNewSpan adds new span for bor to store and updates last span
func (k *Keeper) AddNewSpan(ctx context.Context, span *types.Span) error {
	logger := k.Logger(ctx)
	// store := k.storeService.OpenKVStore(ctx)

	// out, err := k.cdc.Marshal(span)
	// if err != nil {
	// 	k.Logger(ctx).Error("Error marshalling span", "error", err)
	// 	return err
	// }

	// // store set span id
	// if err := store.Set(GetSpanKey(span.Id), out); err != nil {
	// 	return err
	// }

	if err := k.AddNewRawSpan(ctx, span); err != nil {
		// TODO HV2: should we panic here instead ?
		logger.Error("error setting span", "error", err, "span", span)
		return err
	}

	return k.UpdateLastSpan(ctx, span.Id)
}

// AddNewRawSpan adds new span for bor to store
func (k *Keeper) AddNewRawSpan(ctx context.Context, span *types.Span) error {
	// store := k.storeService.OpenKVStore(ctx)

	// out, err := k.cdc.Marshal(span)
	// if err != nil {
	// 	k.Logger(ctx).Error("Error marshalling span", "error", err)
	// 	return err
	// }

	// return store.Set(GetSpanKey(span.Id), out)

	return k.spans.Set(ctx, span.Id, span)
}

// GetSpan fetches span indexed by id from store
func (k *Keeper) GetSpan(ctx context.Context, id uint64) (*types.Span, error) {
	ok, err := k.spans.Has(ctx, id)
	if err != nil {
		return nil, err
	}

	// If we are starting from 0 there will be no spanKey present
	if !ok {
		return nil, fmt.Errorf("span not found for id: %v", id)
	}

	span, err := k.spans.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return span, nil
}

func (k *Keeper) HasSpan(ctx context.Context, id uint64) (bool, error) {
	// store := k.storeService.OpenKVStore(ctx)
	// spanKey := GetSpanKey(id)

	// return store.Has(spanKey)
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

	defer func(iter collections.Iterator[uint64, *types.Span]) {
		err := iter.Close()
		if err != nil {
			logger.Error("error closing span iterator", "err", err)
			return
		}
	}(iter)

	spans, err := iter.Values()
	if err != nil {
		logger.Error("error getting spans from iterator", "err", err)
		return nil, err
	}

	return spans, err
}

// GetSpanList returns all spans with params like page and limit
func (k *Keeper) FetchSpanList(ctx context.Context, page uint64, limit uint64) ([]types.Span, error) {
	store := k.storeService.OpenKVStore(ctx)

	// have max limit
	if limit > 20 {
		limit = 20
	}

	// get paginated iterator
	st := runtime.KVStoreAdapter(store)
	iterator := storetypes.KVStorePrefixIteratorPaginated(st, types.SpanPrefixKey, uint(page), uint(limit))
	defer iterator.Close()

	// loop through validators to get valid validators
	var spans []types.Span

	for ; iterator.Valid(); iterator.Next() {
		var span types.Span
		if err := k.cdc.Unmarshal(iterator.Value(), &span); err == nil {
			spans = append(spans, span)
		}
	}

	return spans, nil
}

// GetLastSpan fetches last span from store
func (k *Keeper) GetLastSpan(ctx context.Context) (*types.Span, error) {
	ok, err := k.latestSpan.Has(ctx)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("last span not found")
	}

	// get last span id
	lastSpanId, err := k.latestSpan.Get(ctx)
	if err != nil {
		return nil, err
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

	// generate new span
	newSpan := &types.Span{
		Id:                id,
		StartBlock:        startBlock,
		EndBlock:          endBlock,
		ValidatorSet:      k.sk.GetValidatorSet(ctx),
		SelectedProducers: newProducers,
		ChainId:           borChainID,
	}

	return k.AddNewSpan(ctx, newSpan)
}

// SelectNextProducers selects producers for next span
func (k *Keeper) SelectNextProducers(ctx context.Context, seed common.Hash) ([]types.Validator, error) {
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

	vals := make([]types.Validator, 0, len(newProducersIds))

	IDToPower := make(map[uint64]uint64)
	for _, ID := range newProducersIds {
		IDToPower[ID] = IDToPower[ID] + 1
	}

	for key, value := range IDToPower {
		if val, ok := k.sk.GetValidatorFromValID(ctx, key); ok {
			val.VotingPower = int64(value)
			vals = append(vals, val)
		}
	}

	// sort by address
	// TODO HV2: uncomment when helper is merged
	// vals = helper.SortValidatorByAddress(vals)

	return vals, nil
}

// UpdateLastSpan updates the last span
func (k *Keeper) UpdateLastSpan(ctx context.Context, id uint64) error {
	// store := k.storeService.OpenKVStore(ctx)
	// return store.Set(types.LastSpanIDKey, []byte(strconv.FormatUint(id, 10)))
	return k.latestSpan.Set(ctx, id)
}

// IncrementLastEthBlock increment last eth block
func (k *Keeper) IncrementLastEthBlock(ctx context.Context) error {
	// store := k.storeService.OpenKVStore(ctx)
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
	// store := k.storeService.OpenKVStore(ctx)
	// return store.Set(types.LastProcessedEthBlock, blockNumber.Bytes())
	return k.lastEthBlock.Set(ctx, blockNumber.Bytes())
}

// GetLastEthBlock gets last processed Eth block for seed
func (k *Keeper) GetLastEthBlock(ctx context.Context) (*big.Int, error) {
	// store := k.storeService.OpenKVStore(ctx)
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

// GetNextSpanSeed gets the eth block hash which serves as the seed for random selection of producer set
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

	// TODO HV2: uncomment when contractCaller is implemented
	// fetch block header from mainchain
	// blockHeader, err := k.contractCaller.GetMainChainBlock(newEthBlock)
	blockHeader := &ethtypes.Header{Number: newEthBlock} // dummy block header to avoid panic
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

// GetParams gets the bor module's parameters.
func (k *Keeper) FetchParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}
	return params, nil
}
