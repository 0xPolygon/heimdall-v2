package keeper

import (
	"context"
	"math/big"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bk "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	// TODO HV2: enable stakeKeeper, chainKeeper and contractCaller when implemented in heimdall-v2
	// "github.com/0xPolygon/heimdall-v2/chainmanager/keeper"
	// "github.com/0xPolygon/heimdall-v2/helper"
	// "github.com/0xPolygon/heimdall-v2/stake/keeper"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

// Keeper stores all chainmanager related data
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	schema       collections.Schema

	bankKeeper bk.Keeper

	// TODO HV2: enable stakeKeeper, chainKeeper and contractCaller when implemented in heimdall-v2
	// stakingKeeper sk.Keeper
	// chainKeeper ck.Keeper
	// IContractCaller helper.IContractCaller

	sequences        collections.Map[string, bool]
	dividendAccounts collections.Map[string, hTypes.DividendAccount]
}

// NewKeeper create new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	bankKeeper bk.Keeper,
	/*
		 TODO HV2: enable stakeKeeper, chainKeeper and contractCaller when implemented
		    stake sk.Keeper,
			chainKeeper ck.Keeper,
			contractCaller helper.IContractCaller,
	*/
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		bankKeeper: bankKeeper,
		/* TODO HV2: enable stakeKeeper, chainKeeper and contractCaller when implemented in heimdall-v2
		stakingKeeper: 	stakingKeeper,
		chainKeeper:   	chainKeeper,
		contractCaller: contractCaller,
		*/

		// TODO HV2: in heimdall-v1, the keys are always prefixed with the key, then removed when getters are invoked
		//  in heimdall-v2, I am only using plain keys, without the prefix. To double check.
		sequences:        collections.NewMap(sb, types.TopupSequencePrefixKey, "topup_sequence", collections.StringKey, collections.BoolValue),
		dividendAccounts: collections.NewMap(sb, types.DividendAccountMapKey, "dividend_account", collections.StringKey, codec.CollValue[hTypes.DividendAccount](cdc)),
	}

	// build the schema and set it in the keeper
	s, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.schema = s

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAllTopupSequences returns all the topup sequences
func (k *Keeper) GetAllTopupSequences(ctx sdk.Context) ([]string, error) {
	// get the sequences iterator
	iter, err := k.sequences.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	// defer closing the iterator
	defer func(iter collections.Iterator[string, bool]) {
		err := iter.Close()
		if err != nil {
			k.Logger(ctx).Error("error closing topup sequences iterator", "err", err)
		}
	}(iter)
	// iterate over sequences' keys, and return them
	sequences, err := iter.Keys()
	if err != nil {
		k.Logger(ctx).Error("error getting topup sequences from the iterator", "err", err)
		return nil, err
	}
	return sequences, nil
}

// SetTopupSequence sets the topup sequence value in the store for the given key
func (k *Keeper) SetTopupSequence(ctx sdk.Context, sequence string) error {
	err := k.sequences.Set(ctx, sequence, types.DefaultTopupSequenceValue)
	if err != nil {
		k.Logger(ctx).Error("error setting topup sequence", "sequence", sequence, "err", err)
		return err
	}
	k.Logger(ctx).Debug("topup sequence set", "sequence", sequence)
	return nil
}

// HasTopupSequence checks if the topup sequence exists
func (k *Keeper) HasTopupSequence(ctx sdk.Context, sequence string) (bool, error) {
	isSequencePresent, err := k.sequences.Has(ctx, sequence)
	if err != nil {
		k.Logger(ctx).Error("error checking if topup sequence exists", "sequence", sequence, "err", err)
		return false, err
	}
	k.Logger(ctx).Debug("topup sequence exists", "sequence", sequence, "isSequencePresent", isSequencePresent)
	return isSequencePresent, nil
}

// GetAllDividendAccounts returns all the dividend accounts
func (k *Keeper) GetAllDividendAccounts(ctx sdk.Context) ([]hTypes.DividendAccount, error) {
	// get the dividend accounts iterator
	iter, err := k.dividendAccounts.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	// defer closing the iterator
	defer func(iter collections.Iterator[string, hTypes.DividendAccount]) {
		err := iter.Close()
		if err != nil {
			k.Logger(ctx).Error("error closing dividend accounts iterator", "err", err)
		}
	}(iter)
	// iterate over dividend accounts' values, and return them
	dividendAccounts, err := iter.Values()
	if err != nil {
		k.Logger(ctx).Error("error getting dividend accounts from the iterator", "err", err)
		return nil, err
	}
	return dividendAccounts, nil
}

// SetDividendAccount sets the dividend account in the store for the given dividendAccount.User
func (k *Keeper) SetDividendAccount(ctx sdk.Context, dividendAccount hTypes.DividendAccount) error {
	err := k.dividendAccounts.Set(ctx, dividendAccount.User, dividendAccount)
	if err != nil {
		k.Logger(ctx).Error("error adding dividend account", "dividendAccount", dividendAccount, "err", err)
		return err
	}
	k.Logger(ctx).Debug("dividend account added", "dividendAccount", dividendAccount)
	return nil
}

// HasDividendAccount checks if the dividend account exists
func (k *Keeper) HasDividendAccount(ctx sdk.Context, user string) (bool, error) {
	isDividendAccountPresent, err := k.dividendAccounts.Has(ctx, user)
	if err != nil {
		k.Logger(ctx).Error("error checking if dividend account exists", "user", user, "err", err)
		return false, err
	}
	k.Logger(ctx).Debug("dividend account exists", "user", user, "isDividendAccountPresent", isDividendAccountPresent)
	return isDividendAccountPresent, nil
}

// GetDividendAccount returns the dividend account for the given user
func (k *Keeper) GetDividendAccount(ctx sdk.Context, user string) (hTypes.DividendAccount, error) {
	dividendAccount, err := k.dividendAccounts.Get(ctx, user)
	if err != nil {
		k.Logger(ctx).Error("error getting dividend account", "user", user, "err", err)
		return hTypes.DividendAccount{}, err
	}
	k.Logger(ctx).Debug("dividend account retrieved", "user", user, "dividendAccount", dividendAccount)
	return dividendAccount, nil
}

// AddFeeToDividendAccount adds the fee to the dividend account for the given user
func (k *Keeper) AddFeeToDividendAccount(ctx sdk.Context, user string, fee *big.Int) error {
	// check if dividendAccount exists
	exist, err := k.HasDividendAccount(ctx, user)
	if err != nil {
		return err
	}
	var dividendAccount hTypes.DividendAccount
	if !exist {
		// create a new dividend account
		k.Logger(ctx).Debug("dividend account not found, creating one", "user", user)
		dividendAccount = hTypes.DividendAccount{
			User:      user,
			FeeAmount: big.NewInt(0).String(),
		}
	} else {
		// get the dividend account
		dividendAccount, err = k.GetDividendAccount(ctx, user)
		if err != nil {
			return err
		}
	}
	// update the fee
	oldFee, _ := big.NewInt(0).SetString(dividendAccount.FeeAmount, 10)
	totalFee := big.NewInt(0).Add(oldFee, fee).String()
	dividendAccount.FeeAmount = totalFee
	k.Logger(ctx).Info("fee added to dividend account", "user", user, "oldFee", oldFee, "addedFee", fee, "totalFee", totalFee)

	// set the updated dividend account
	err = k.SetDividendAccount(ctx, dividendAccount)
	if err != nil {
		k.Logger(ctx).Error("error adding fee to dividend account", "user", user, "fee", fee, "err", err)
		return err
	}
	return nil
}
