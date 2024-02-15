package types

import (
	"bytes"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	cosmosCryto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	_ "cosmossdk.io/x/nft"
	cosmosTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewValidator func creates a new validator,
// the HeimdallAddress field is generated using Address i.e. [20]byte
func NewValidator(
	id uint64,
	startEpoch uint64,
	endEpoch uint64,
	nonce uint64,
	power int64,
	pubKey cryptotypes.PubKey,
	signer string,
) *HeimdallValidator {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return nil
	}

	return &HeimdallValidator{
		ValId:       id,
		StartEpoch:  startEpoch,
		EndEpoch:    endEpoch,
		Nonce:       nonce,
		VotingPower: power,
		PubKey:      pkAny,
		Signer:      signer,
	}
}

// SortValidatorByAddress sorts a slice of validators by address
// to sort it we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
func SortValidatorByAddress(a []HeimdallValidator) []HeimdallValidator {
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(strings.ToLower(a[i].Signer), strings.ToLower(a[j].Signer)) < 0
	})

	return a
}

// IsCurrentValidator checks if validator is in current validator set
func (v *HeimdallValidator) IsCurrentValidator(ackCount uint64) bool {
	// current epoch will be ack count + 1
	currentEpoch := ackCount + 1

	// validator hasn't initialised unstake
	if !v.Jailed && v.StartEpoch <= currentEpoch && (v.EndEpoch == 0 || v.EndEpoch > currentEpoch) && v.VotingPower > 0 {
		return true
	}

	return false
}

// Validates validator
func (v *HeimdallValidator) ValidateBasic() bool {
	pk, ok := v.PubKey.GetCachedValue().(cryptotypes.PubKey)

	if !ok {
		return false
	}
	if bytes.Equal(pk.Bytes(), ZeroPubKey.Bytes()) {
		return false
	}

	zeroAddress := common.Address{}.String()

	if strings.Compare(strings.ToLower(v.Signer), strings.ToLower(zeroAddress)) == 0 {
		return false
	}

	return true
}

// amino marshall validator
func MarshallValidator(cdc codec.BinaryCodec, validator HeimdallValidator) (bz []byte, err error) {
	bz, err = cdc.Marshal(&validator)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// amono unmarshall validator
func UnmarshallValidator(cdc codec.BinaryCodec, value []byte) (HeimdallValidator, error) {
	var validator HeimdallValidator

	if err := cdc.Unmarshal(value, &validator); err != nil {
		return validator, err
	}

	return validator, nil
}

// Copy creates a new copy of the validator so we can mutate accum.
// Panics if the validator is nil.
func (v *HeimdallValidator) Copy() *HeimdallValidator {
	vCopy := *v
	return &vCopy
}

// CompareProposerPriority returns the one with higher ProposerPriority.
func (v *HeimdallValidator) CompareProposerPriority(other *HeimdallValidator) *HeimdallValidator {
	if v == nil {
		return other
	}

	switch {
	case v.ProposerPriority > other.ProposerPriority:
		return v
	case v.ProposerPriority < other.ProposerPriority:
		return other
	default:
		result := strings.Compare(strings.ToLower(v.Signer), strings.ToLower(other.Signer))

		switch {
		case result < 0:
			return v
		case result > 0:
			return other
		default:
			panic("Cannot compare identical validators")
		}
	}
}

// Bytes computes the unique encoding of a validator with a given voting power.
// These are the bytes that gets hashed in consensus. It excludes address
// as its redundant with the pubkey. This also excludes ProposerPriority
// which changes every round.
func (v *HeimdallValidator) Bytes() []byte {
	result := make([]byte, 64)

	copy(result[12:], []byte(v.Signer))
	copy(result[32:], new(big.Int).SetInt64(v.VotingPower).Bytes())

	return result
}

// UpdatedAt returns block number of last validator update
func (v *HeimdallValidator) UpdatedAt() string {
	return v.LastUpdated
}

// MinimalVal returns block number of last validator update
func (v *HeimdallValidator) MinimalVal() MinimalVal {
	return MinimalVal{
		ID:          v.ValId,
		VotingPower: uint64(v.VotingPower),
		Signer:      v.Signer,
	}
}

// GetBondedTokens implements types.ValidatorI.
func (v *HeimdallValidator) GetBondedTokens() math.Int {
	return math.NewInt(v.VotingPower)
}

// GetOperator implements types.ValidatorI.
func (v *HeimdallValidator) GetOperator() string {
	return v.Signer
}

// --------

// Bytes get bytes of validatorID
func ValIDToBytes(valID uint64) []byte {
	return []byte(strconv.FormatUint(valID, 10))
}

// --------

// MinimalVal is the minimal validator representation
// Used to send validator information to bor validator contract
type MinimalVal struct {
	ID          uint64 `json:"ID"`
	VotingPower uint64 `json:"power"` // TODO add 10^-18 here so that we dont overflow easily
	Signer      string `json:"signer"`
}

func (v HeimdallValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(v.PubKey, &pk)
}

///////Following functions are implemented to support cosmos validator interface/////////

// ConsPubKey implements types.ValidatorI.
func (v *HeimdallValidator) ConsPubKey() (cryptotypes.PubKey, error) {
	panic("unimplemented")
}

// GetCommission implements types.ValidatorI.
func (v *HeimdallValidator) GetCommission() math.LegacyDec {
	panic("unimplemented")
}

// GetConsAddr implements types.ValidatorI.
func (v *HeimdallValidator) GetConsAddr() ([]byte, error) {
	panic("unimplemented")
}

// GetConsensusPower implements types.ValidatorI.
func (v *HeimdallValidator) GetConsensusPower(math.Int) int64 {
	panic("unimplemented")
}

// GetDelegatorShares implements types.ValidatorI.
func (v *HeimdallValidator) GetDelegatorShares() math.LegacyDec {
	panic("unimplemented")
}

// GetMinSelfDelegation implements types.ValidatorI.
func (v *HeimdallValidator) GetMinSelfDelegation() math.Int {
	panic("unimplemented")
}

// GetMoniker implements types.ValidatorI.
func (v *HeimdallValidator) GetMoniker() string {
	panic("unimplemented")
}

// GetStatus implements types.ValidatorI.
func (v *HeimdallValidator) GetStatus() cosmosTypes.BondStatus {
	panic("unimplemented")
}

// GetTokens implements types.ValidatorI.
func (v *HeimdallValidator) GetTokens() math.Int {
	panic("unimplemented")
}

// IsBonded implements types.ValidatorI.
func (v *HeimdallValidator) IsBonded() bool {
	panic("unimplemented")
}

// IsJailed implements types.ValidatorI.
func (v *HeimdallValidator) IsJailed() bool {
	panic("unimplemented")
}

// IsUnbonded implements types.ValidatorI.
func (v *HeimdallValidator) IsUnbonded() bool {
	panic("unimplemented")
}

// IsUnbonding implements types.ValidatorI.
func (v *HeimdallValidator) IsUnbonding() bool {
	panic("unimplemented")
}

// SharesFromTokens implements types.ValidatorI.
func (v *HeimdallValidator) SharesFromTokens(amt math.Int) (math.LegacyDec, error) {
	panic("unimplemented")
}

// SharesFromTokensTruncated implements types.ValidatorI.
func (v *HeimdallValidator) SharesFromTokensTruncated(amt math.Int) (math.LegacyDec, error) {
	panic("unimplemented")
}

// TmConsPublicKey implements types.ValidatorI.
func (v *HeimdallValidator) TmConsPublicKey() (cosmosCryto.PublicKey, error) {
	panic("unimplemented")
}

// TokensFromShares implements types.ValidatorI.
func (v *HeimdallValidator) TokensFromShares(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}

// TokensFromSharesRoundUp implements types.ValidatorI.
func (v *HeimdallValidator) TokensFromSharesRoundUp(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}

func (v *HeimdallValidator) TokensFromSharesTruncated(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}
