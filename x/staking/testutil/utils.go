package testutil

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/0xPolygon/heimdall-v2/x/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenRandomVal generate random validators
func GenRandomVal(count int, startBlock uint64, power int64, timeAlive uint64, randomise bool, startID uint64) (validators []types.Validator) {
	for i := 0; i < count; i++ {
		pubKey := secp256k1.GenPrivKey().PubKey()
		addr := sdk.ValAddress(pubKey.Address())

		pkAny, err := codectypes.NewAnyWithValue(pubKey)
		if err != nil {
			fmt.Errorf("Error in generating the pubKey")
			return
		}

		if randomise {
			startBlock = generateRandNumber(10)
			power = int64(generateRandNumber(100))
		}

		newVal := types.Validator{
			ID:               types.NewValidatorID(startID + uint64(i)),
			StartEpoch:       startBlock,
			EndEpoch:         startBlock + timeAlive,
			VotingPower:      power,
			Signer:           types.HeimdallAddress{addr},
			PubKey:           pkAny,
			ProposerPriority: 0,
		}
		validators = append(validators, newVal)
	}

	return
}

func generateRandNumber(max int64) uint64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 1
	}

	return nBig.Uint64()
}
