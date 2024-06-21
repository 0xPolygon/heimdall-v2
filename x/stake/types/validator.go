package types

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	cmtprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cosmosCryto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewValidator func creates a new validator
func NewValidator(
	id uint64,
	startEpoch uint64,
	endEpoch uint64,
	nonce uint64,
	power int64,
	pubKey cryptotypes.PubKey,
	signer string,
) (*Validator, error) {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return nil, err
	}
	return &Validator{
		ValId:       id,
		StartEpoch:  startEpoch,
		EndEpoch:    endEpoch,
		Nonce:       nonce,
		VotingPower: power,
		PubKey:      pkAny,
		Signer:      signer,
	}, nil
}

// SortValidatorByAddress sorts a slice of validators by address
func SortValidatorByAddress(a []Validator) []Validator {
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(strings.ToLower(a[i].Signer), strings.ToLower(a[j].Signer)) < 0
	})

	return a
}

// IsCurrentValidator checks if validator is in current validator set
func (v *Validator) IsCurrentValidator(ackCount uint64) bool {
	// current epoch will be ack count + 1
	currentEpoch := ackCount + 1

	// validator hasn't initialised unstake
	return !v.Jailed && v.StartEpoch <= currentEpoch && (v.EndEpoch == 0 || v.EndEpoch > currentEpoch) && v.VotingPower > 0
}

// ValidateBasic validates a validator struct
func (v *Validator) ValidateBasic() bool {
	pk, ok := v.PubKey.GetCachedValue().(cryptotypes.PubKey)

	if !ok {
		return false
	}
	if bytes.Equal(pk.Bytes(), ZeroPubKey[:]) {
		return false
	}

	if strings.Compare(strings.ToLower(v.Signer), strings.ToLower(common.Address{}.String())) == 0 {
		return false
	}

	return true
}

// MarshallValidator is responsible for marshalling validator
func MarshallValidator(cdc codec.BinaryCodec, validator Validator) (bz []byte, err error) {
	return cdc.Marshal(&validator)
}

// UnmarshallValidator is responsible for unmarshalling validator
func UnmarshallValidator(cdc codec.BinaryCodec, value []byte) (Validator, error) {
	var validator Validator
	err := cdc.Unmarshal(value, &validator)

	return validator, err
}

// Copy creates a new copy of the validator, so we can mutate accum
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// CompareProposerPriority returns the one with higher ProposerPriority.
func (v *Validator) CompareProposerPriority(other *Validator) *Validator {
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
			panic("cannot compare identical validators")
		}
	}
}

// ConsPubKey returns the validator PubKey as a cryptotypes.PubKey.
func (v Validator) ConsPubKey() (cryptotypes.PubKey, error) {
	pk, ok := v.PubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, fmt.Errorf("expecting cryptotypes.PubKey, got %T", pk)
	}

	return pk, nil
}

// Bytes computes the unique encoding of a validator with a given voting power.
// These are the bytes that gets hashed in consensus. It excludes address
// as it's redundant with the pubKey. This also excludes ProposerPriority
// which changes every round.
func (v *Validator) Bytes() []byte {
	result := make([]byte, 64)

	copy(result[12:], v.Signer)
	copy(result[32:], new(big.Int).SetInt64(v.VotingPower).Bytes())

	return result
}

// UpdatedAt returns block number of last validator update
func (v *Validator) UpdatedAt() string {
	return v.LastUpdated
}

// MinimalVal returns block number of last validator update
func (v *Validator) MinimalVal() MinimalVal {
	return MinimalVal{
		ID:          v.ValId,
		VotingPower: uint64(v.VotingPower),
		Signer:      v.Signer,
	}
}

// GetBondedTokens implements types.ValidatorI.
func (v *Validator) GetBondedTokens() math.Int {
	return math.NewInt(v.VotingPower)
}

// GetOperator implements types.ValidatorI.
func (v *Validator) GetOperator() string {
	return v.Signer
}

// ValIDToBytes get the bytes from a validatorID
func ValIDToBytes(valID uint64) []byte {
	return []byte(strconv.FormatUint(valID, 10))
}

// CmtConsPublicKey casts Validator.ConsensusPubkey to cmtprotocrypto.PubKey.
func (v Validator) CmtConsPublicKey() (cmtprotocrypto.PublicKey, error) {
	pk, err := v.ConsPubKey()
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	tmPk, err := cryptocodec.ToCmtProtoPublicKey(pk)
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	return tmPk, nil
}

// MinimalVal is the minimal validator representation
// Used to send validator information to bor validator contract
type MinimalVal struct {
	ID          uint64 `json:"ID"`
	VotingPower uint64 `json:"power"` // TODO add 10^-18 here so that we dont overflow easily
	Signer      string `json:"signer"`
}

func (v Validator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(v.PubKey, &pk)
}

// Following functions are implemented to support cosmos validator interface

// GetCommission implements types.ValidatorI.
func (*Validator) GetCommission() math.LegacyDec {
	panic("unimplemented")
}

// GetConsAddr implements types.ValidatorI.
func (*Validator) GetConsAddr() ([]byte, error) {
	panic("unimplemented")
}

// GetConsensusPower implements types.ValidatorI.
func (*Validator) GetConsensusPower(math.Int) int64 {
	panic("unimplemented")
}

// GetDelegatorShares implements types.ValidatorI.
func (*Validator) GetDelegatorShares() math.LegacyDec {
	panic("unimplemented")
}

// GetMinSelfDelegation implements types.ValidatorI.
func (*Validator) GetMinSelfDelegation() math.Int {
	panic("unimplemented")
}

// GetMoniker implements types.ValidatorI.
func (*Validator) GetMoniker() string {
	panic("unimplemented")
}

// GetStatus implements types.ValidatorI.
func (*Validator) GetStatus() cosmosTypes.BondStatus {
	panic("unimplemented")
}

// GetTokens implements types.ValidatorI.
func (*Validator) GetTokens() math.Int {
	panic("unimplemented")
}

// IsBonded implements types.ValidatorI.
func (*Validator) IsBonded() bool {
	panic("unimplemented")
}

// IsJailed implements types.ValidatorI.
func (*Validator) IsJailed() bool {
	panic("unimplemented")
}

// IsUnbonded implements types.ValidatorI.
func (*Validator) IsUnbonded() bool {
	panic("unimplemented")
}

// IsUnbonding implements types.ValidatorI.
func (*Validator) IsUnbonding() bool {
	panic("unimplemented")
}

// SharesFromTokens implements types.ValidatorI.
func (*Validator) SharesFromTokens(amt math.Int) (math.LegacyDec, error) {
	panic("unimplemented")
}

// SharesFromTokensTruncated implements types.ValidatorI.
func (*Validator) SharesFromTokensTruncated(amt math.Int) (math.LegacyDec, error) {
	panic("unimplemented")
}

// TmConsPublicKey implements types.ValidatorI.
func (*Validator) TmConsPublicKey() (cosmosCryto.PublicKey, error) {
	panic("unimplemented")
}

// TokensFromShares implements types.ValidatorI.
func (*Validator) TokensFromShares(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}

// TokensFromSharesRoundUp implements types.ValidatorI.
func (*Validator) TokensFromSharesRoundUp(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}

func (*Validator) TokensFromSharesTruncated(math.LegacyDec) math.LegacyDec {
	panic("unimplemented")
}
