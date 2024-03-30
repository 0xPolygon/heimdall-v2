package keeper

import (
	"context"
	"errors"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// AddMilestone adds milestone in the store
func (k *Keeper) AddMilestone(ctx context.Context, milestone types.Milestone) error {
	milestoneNumber := k.GetMilestoneCount(ctx) + 1 //GetCount gives the number of previous milestone

	key := types.GetMilestoneKey(milestoneNumber)
	if err := k.addMilestone(ctx, key, milestone); err != nil {
		return err
	}

	k.SetMilestoneCount(ctx, milestoneNumber)
	k.Logger(ctx).Info("Adding good milestone to state", "milestone", milestone, "milestoneNumber", milestoneNumber)

	return nil
}

// addMilestone adds milestone to store
func (k *Keeper) addMilestone(ctx context.Context, key []byte, milestone types.Milestone) error {
	store := k.storeService.OpenKVStore(ctx)

	// marshal milestone
	out, err := k.cdc.Marshal(&milestone)
	if err != nil {
		k.Logger(ctx).Error("Error marshalling milestone", "error", err)
		return err
	}

	// store in key provided
	store.Set(key, out)

	return nil
}

// GetMilestoneByNumber to get milestone by milestone number
func (k *Keeper) GetMilestoneByNumber(ctx context.Context, number uint64) (*types.Milestone, error) {
	store := k.storeService.OpenKVStore(ctx)
	milestoneKey := types.GetMilestoneKey(number)

	var _milestone types.Milestone

	milestoneBytes, err := store.Get(milestoneKey)

	if err != nil {
		return nil, errors.New("error while fetchig the milestone from the store")
	}

	if milestoneBytes == nil {
		return nil, types.ErrNoMilestoneFound
	}

	// unmarshall validator and return
	_milestone, err = types.UnMarshallMilestone(k.cdc, milestoneBytes)
	if err != nil {
		return nil, err
	}

	return &_milestone, nil
}

// GetLastMilestone gets last milestone, milestone number = GetCount()
func (k *Keeper) GetLastMilestone(ctx context.Context) (*types.Milestone, error) {
	store := k.storeService.OpenKVStore(ctx)
	Count := k.GetMilestoneCount(ctx)

	lastMilestoneKey := types.GetMilestoneKey(Count)

	var _milestone types.Milestone

	milestoneBytes, err := store.Get(lastMilestoneKey)

	if err != nil {
		return nil, errors.New("error while fetchig the milestone from the store")
	}

	if milestoneBytes == nil {
		return nil, types.ErrNoMilestoneFound
	}

	// unmarshall validator and return
	_milestone, err = types.UnMarshallMilestone(k.cdc, milestoneBytes)
	if err != nil {
		return nil, err
	}

	return &_milestone, nil
}

// SetCount set the count number
func (k *Keeper) SetMilestoneCount(ctx context.Context, number uint64) {
	store := k.storeService.OpenKVStore(ctx)
	// convert timestamp to bytes
	value := []byte(strconv.FormatUint(number, 10))
	// set no-ack
	store.Set(types.CountKey, value)
}

// GetCount returns milestone count
func (k *Keeper) GetMilestoneCount(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)

	countBytes, err := store.Get(types.CountKey)

	if err != nil {
		return uint64(0)
	}

	if countBytes == nil {
		return uint64(0)
	}

	result, err := strconv.ParseUint(string(countBytes), 10, 64)
	if err == nil {
		return result
	}

	return uint64(0)
}

// SetMilestoneBlockNumber set the block number when the latest milestone enter the handler
func (k *Keeper) SetMilestoneBlockNumber(ctx context.Context, number int64) {
	store := k.storeService.OpenKVStore(ctx)
	// convert block number to bytes
	value := []byte(strconv.FormatInt(number, 10))
	// set
	store.Set(types.BlockNumberKey, value)
}

// GetMilestoneBlockNumber returns the block number when the latest milestone enter the handler
func (k *Keeper) GetMilestoneBlockNumber(ctx context.Context) int64 {
	store := k.storeService.OpenKVStore(ctx)

	blockNumberBytes, err := store.Get(types.BlockNumberKey)

	if err != nil {
		return int64(0)
	}

	if blockNumberBytes == nil {
		return int64(0)
	}

	result, err := strconv.ParseInt(string(blockNumberBytes), 10, 64)
	if err == nil {
		return result
	}

	return int64(0)
}

// SetLastNoAck set last no-ack object
func (k *Keeper) SetNoAckMilestone(ctx context.Context, milestoneId string) {
	store := k.storeService.OpenKVStore(ctx)

	milestoneNoAckKey := types.GetMilestoneNoAckKey(milestoneId)
	value := []byte(milestoneId)

	// set no-ack-milestone
	store.Set(milestoneNoAckKey, value)
	store.Set(types.MilestoneLastNoAckKey, value)
}

// GetLastNoAckMilestone returns last no ack milestone
func (k *Keeper) GetLastNoAckMilestone(ctx context.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	// check if MilestoneLastNoAckKey key exists

	lastNoAckBytes, err := store.Get(types.MilestoneLastNoAckKey)

	if err != nil || lastNoAckBytes == nil {
		return ""
	}

	return string(lastNoAckBytes)
}

// GetLastNoAckMilestone returns last no ack milestone
func (k *Keeper) GetNoAckMilestone(ctx context.Context, milestoneId string) bool {
	store := k.storeService.OpenKVStore(ctx)
	// check if No Ack Milestone is there
	res, err := store.Has(types.GetMilestoneNoAckKey(milestoneId))

	if err != nil {
		return false
	}

	return res
}

// SetLastMilestoneTimeout set lastMilestone timeout time
func (k *Keeper) SetLastMilestoneTimeout(ctx context.Context, timestamp uint64) {
	store := k.storeService.OpenKVStore(ctx)
	// convert timestamp to bytes
	value := []byte(strconv.FormatUint(timestamp, 10))
	// set no-ack
	store.Set(types.LastMilestoneTimeout, value)
}

// GetLastMilestoneTimeout returns lastMilestone timeout time
func (k *Keeper) GetLastMilestoneTimeout(ctx context.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	//check if lastMilestoneTimeout key exists

	lastMilestoneBytes, err := store.Get(types.LastMilestoneTimeout)

	if err != nil {
		return uint64(0)
	}

	if lastMilestoneBytes == nil {
		return uint64(0)
	}

	result, err := strconv.ParseUint(string(lastMilestoneBytes), 10, 64)
	if err == nil {
		return result
	}

	return 0
}
