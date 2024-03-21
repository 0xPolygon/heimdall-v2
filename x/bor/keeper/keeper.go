package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// Keeper stores all chainmanager related data
type Keeper struct {
	cdc            codec.BinaryCodec
	storeService   store.KVStoreService
	ck             types.ChainManagerKeeper
	sk             types.StakeKeeper
	contractCaller helper.ContractCaller
	Params         collections.Item[types.Params]
}

// NewKeeper creates a new instance of the bor Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	chainKeeper types.ChainManagerKeeper,
	stakingKeeper types.StakeKeeper,
	caller helper.ContractCaller,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	return Keeper{
		cdc:            cdc,
		storeService:   storeService,
		ck:             chainKeeper,
		sk:             stakingKeeper,
		contractCaller: caller,
		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetSpanKey appends prefix to start block
func GetSpanKey(id uint64) []byte {
	return append(types.SpanPrefixKey, []byte(strconv.FormatUint(id, 10))...)
}

// AddNewSpan adds new span for bor to store and updates last span
func (k *Keeper) AddNewSpan(ctx context.Context, span *types.Span) error {
	store := k.storeService.OpenKVStore(ctx)

	out, err := k.cdc.Marshal(span)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling span", "error", err)
		return err
	}

	// store set span id
	if err := store.Set(GetSpanKey(span.Id), out); err != nil {
		return err
	}

	// update last span
	return k.UpdateLastSpan(ctx, span.Id)
}

// AddNewRawSpan adds new span for bor to store
func (k *Keeper) AddNewRawSpan(ctx context.Context, span *types.Span) error {
	store := k.storeService.OpenKVStore(ctx)

	out, err := k.cdc.Marshal(span)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling span", "error", err)
		return err
	}

	return store.Set(GetSpanKey(span.Id), out)
}

// GetSpan fetches span indexed by id from store
func (k *Keeper) GetSpan(ctx context.Context, id uint64) (*types.Span, error) {
	store := k.storeService.OpenKVStore(ctx)
	spanKey := GetSpanKey(id)

	ok, err := store.Has(spanKey)
	if err != nil {
		return nil, err
	}

	// If we are starting from 0 there will be no spanKey present
	if !ok {
		return nil, fmt.Errorf("span not found for id: %v", id)
	}

	spanBytes, err := store.Get(spanKey)
	if err != nil {
		return nil, err
	}

	var span types.Span
	if err := k.cdc.Unmarshal(spanBytes, &span); err != nil {
		return nil, err
	}

	return &span, nil
}

func (k *Keeper) HasSpan(ctx context.Context, id uint64) (bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	spanKey := GetSpanKey(id)

	return store.Has(spanKey)
}

// GetAllSpans fetches all spans indexed by id from store
func (k *Keeper) GetAllSpans(ctx context.Context) (spans []*types.Span) {
	// iterate through spans and create span update array
	k.IterateSpansAndApplyFn(ctx, func(span types.Span) error {
		// append to list of validatorUpdates
		spans = append(spans, &span)
		return nil
	})

	return
}

// GetSpanList returns all spans with params like page and limit
func (k *Keeper) GetSpanList(ctx context.Context, page uint64, limit uint64) ([]types.Span, error) {
	store := k.storeService.OpenKVStore(ctx)

	// have max limit
	if limit > 20 {
		limit = 20
	}

	// get paginated iterator
	st := runtime.KVStoreAdapter(store)
	iterator := storetypes.KVStorePrefixIteratorPaginated(st, types.SpanPrefixKey, uint(page), uint(limit))

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

// GetLastSpan fetches last span using lastStartBlock
func (k *Keeper) GetLastSpan(ctx context.Context) (*types.Span, error) {
	store := k.storeService.OpenKVStore(ctx)

	var lastSpanID uint64

	ok, err := store.Has(types.LastSpanIDKey)
	if err != nil {
		return nil, err
	}

	if ok {
		// get last span id
		lastSpanBytes, err := store.Get(types.LastSpanIDKey)
		if err != nil {
			return nil, err
		}
		if lastSpanID, err = strconv.ParseUint(string(lastSpanBytes), 10, 64); err != nil {
			return nil, err
		}
	}

	return k.GetSpan(ctx, lastSpanID)
}

// FreezeSet freezes validator set for next span
func (k *Keeper) FreezeSet(ctx sdk.Context, id uint64, startBlock uint64, endBlock uint64, borChainID string, seed common.Hash) error {
	// select next producers
	newProducers, err := k.SelectNextProducers(ctx, seed)
	if err != nil {
		return err
	}

	// increment last eth block
	k.IncrementLastEthBlock(ctx)

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
func (k *Keeper) SelectNextProducers(ctx context.Context, seed common.Hash) (vals []types.Validator, err error) {
	// spanEligibleVals are current validators who are not getting deactivated in between next span
	spanEligibleVals := k.sk.GetSpanEligibleValidators(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	producerCount := params.ProducerCount

	// if producers to be selected is more than current validators no need to select/shuffle
	if len(spanEligibleVals) <= int(producerCount) {
		return spanEligibleVals, nil
	}

	// select next producers using seed as block header hash
	newProducersIds, err := SelectNextProducers(seed, spanEligibleVals, producerCount)
	if err != nil {
		return vals, err
	}

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
	vals = helper.SortValidatorByAddress(vals)

	return vals, nil
}

// UpdateLastSpan updates the last span start block
func (k *Keeper) UpdateLastSpan(ctx context.Context, id uint64) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.LastSpanIDKey, []byte(strconv.FormatUint(id, 10)))
}

// IncrementLastEthBlock increment last eth block
func (k *Keeper) IncrementLastEthBlock(ctx context.Context) error {
	store := k.storeService.OpenKVStore(ctx)
	lastEthBlock := big.NewInt(0)
	ok, err := store.Has(types.LastProcessedEthBlock)
	if err != nil {
		return err
	}
	if ok {
		lastEthBlockBytes, err := store.Get(types.LastProcessedEthBlock)
		if err != nil {
			return err
		}
		lastEthBlock = lastEthBlock.SetBytes(lastEthBlockBytes)
	}

	return store.Set(types.LastProcessedEthBlock, lastEthBlock.Add(lastEthBlock, big.NewInt(1)).Bytes())
}

// SetLastEthBlock sets last eth block number
func (k *Keeper) SetLastEthBlock(ctx context.Context, blockNumber *big.Int) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.LastProcessedEthBlock, blockNumber.Bytes())
}

// GetLastEthBlock gets last processed Eth block for seed
func (k *Keeper) GetLastEthBlock(ctx context.Context) (*big.Int, error) {
	store := k.storeService.OpenKVStore(ctx)
	lastEthBlock := big.NewInt(0)
	ok, err := store.Has(types.LastProcessedEthBlock)
	if err != nil {
		return nil, err
	}

	if ok {
		lastEthBlockBytes, err := store.Get(types.LastProcessedEthBlock)
		if err != nil {
			return nil, err
		}
		lastEthBlock = lastEthBlock.SetBytes(lastEthBlockBytes)
	}

	return lastEthBlock, nil
}

func (k *Keeper) GetNextSpanSeed(ctx context.Context) (common.Hash, error) {
	lastEthBlock, err := k.GetLastEthBlock(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	// increment last processed header block number
	newEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
	k.Logger(ctx).Debug("newEthBlock to generate seed", "newEthBlock", newEthBlock)

	// fetch block header from mainchain
	blockHeader, err := k.contractCaller.GetMainChainBlock(newEthBlock)
	if err != nil {
		k.Logger(ctx).Error("Error fetching block header from mainchain while calculating next span seed", "error", err)
		return common.Hash{}, err
	}

	return blockHeader.Hash(), nil
}

// -----------------------------------------------------------------------------
// Params

// SetParams sets the bor module's parameters.
func (k *Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

// GetParams gets the bor module's parameters.
func (k *Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}
	return params, nil
}

// IterateSpansAndApplyFn iterates spans and applies the given function.
func (k *Keeper) IterateSpansAndApplyFn(ctx context.Context, f func(span types.Span) error) {
	store := k.storeService.OpenKVStore(ctx)

	// get span iterator
	iterator, err := store.Iterator(types.SpanPrefixKey, storetypes.PrefixEndBytes(types.SpanPrefixKey))
	if err != nil {
		panic(fmt.Errorf("failed to create iterator: %v", err))
	}
	defer iterator.Close()

	// loop through spans to get valid spans
	for ; iterator.Valid(); iterator.Next() {
		// unmarshal span
		var result types.Span
		if err := k.cdc.Unmarshal(iterator.Value(), &result); err != nil {
			k.Logger(ctx).Error("Error Unmarshal", "error", err)
		}
		// call function and return if required
		if err := f(result); err != nil {
			return
		}
	}
}
