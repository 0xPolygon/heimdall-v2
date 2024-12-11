package stake

import (
	"errors"

	"github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	cmttypes "github.com/cometbft/cometbft/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateGenesis validates the provided stake genesis state to ensure that listed
// validators and staking sequences are valid
func ValidateGenesis(data *types.GenesisState) error {
	for _, validator := range data.Validators {
		if !validator.ValidateBasic() {
			return errors.New("invalid validator")
		}
	}

	for _, sq := range data.StakingSequences {
		if sq == "" {
			return errors.New("invalid sequence")
		}
	}

	return nil
}

// WriteValidators returns a slice of comet genesis validators.
func WriteValidators(ctx sdk.Context, keeper *keeper.Keeper) (vals []cmttypes.GenesisValidator, returnErr error) {
	validators := keeper.GetAllValidators(ctx)
	for _, validator := range validators {
		pk, err := validator.ConsPubKey()
		if err != nil {
			returnErr = err
			return
		}
		pubKey := secp256k1.PubKey{Key: pk}
		cmtPk, err := cryptocodec.ToCmtPubKeyInterface(&pubKey)
		if err != nil {
			returnErr = err
			return
		}
		if cmtPk == nil {
			returnErr = errors.New("invalid public key")
			return
		}

		vals = append(vals, cmttypes.GenesisValidator{
			Address: sdk.ConsAddress(cmtPk.Address()).Bytes(),
			PubKey:  cmtPk,
			Power:   validator.GetVotingPower(),
			Name:    validator.Signer,
		})
	}

	return
}
