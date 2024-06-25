package testutil

import (
	"crypto/rand"
	"math/big"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	stakingKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// GenRandomVals generate random validators
func GenRandomVals(count int, startBlock uint64, power int64, timeAlive uint64, randomise bool, startID uint64) (validators []types.Validator) {
	for i := 0; i < count; i++ {
		pubKey := secp256k1.GenPrivKey().PubKey()
		addr := strings.ToLower(pubKey.Address().String())

		pkAny, err := codectypes.NewAnyWithValue(pubKey)
		if err != nil {
			return
		}

		if randomise {
			startBlock = generateRandNumber(10)
			power = int64(generateRandNumber(100))
		}

		newVal := types.Validator{
			ValId:            startID + uint64(i),
			StartEpoch:       startBlock,
			EndEpoch:         startBlock + timeAlive,
			VotingPower:      power,
			Signer:           addr,
			PubKey:           pkAny,
			ProposerPriority: 0,
		}
		validators = append(validators, newVal)
	}

	return
}

// LoadRandomValidatorSet loads random validator set
func LoadRandomValidatorSet(require *require.Assertions, count int, keeper *stakingKeeper.Keeper, ctx sdk.Context, randomise bool, timeAlive int) types.ValidatorSet {
	var valSet types.ValidatorSet

	validators := GenRandomVals(count, 0, 10, uint64(timeAlive), randomise, 1)
	for _, validator := range validators {
		err := keeper.AddValidator(ctx, validator)
		require.NoError(err, "Unable to set validator, Error: %v", err)

		err = valSet.UpdateWithChangeSet([]*types.Validator{&validator})
		require.NoError(err)
	}

	valSet.IncrementProposerPriority(1)

	err := keeper.UpdateValidatorSetInStore(ctx, valSet)
	require.NoError(err, "Unable to update validator set")

	vals := keeper.GetAllValidators(ctx)
	require.NotNil(vals)

	return valSet
}

func generateRandNumber(max int64) uint64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 1
	}

	return nBig.Uint64()
}
