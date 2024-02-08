package types

import (
	"bytes"
	"math/big"
	"sort"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewValidator func creates a new validator,
// the HeimdallAddress field is generated using Address i.e. [20]byte
func NewValidator(
	id ValidatorID,
	startEpoch uint64,
	endEpoch uint64,
	nonce uint64,
	power int64,
	pubKey cryptotypes.PubKey,
	signer HeimdallAddress,
) *Validator {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return nil
	}
	return &Validator{
		ID:          id,
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
func SortValidatorByAddress(a []Validator) []Validator {
	sort.Slice(a, func(i, j int) bool {
		return bytes.Compare(a[i].Signer.GetAddress(), a[j].Signer.GetAddress()) < 0
	})

	return a
}

// IsCurrentValidator checks if validator is in current validator set
func (v *Validator) IsCurrentValidator(ackCount uint64) bool {
	// current epoch will be ack count + 1
	currentEpoch := ackCount + 1

	// validator hasn't initialised unstake
	if !v.Jailed && v.StartEpoch <= currentEpoch && (v.EndEpoch == 0 || v.EndEpoch > currentEpoch) && v.VotingPower > 0 {
		return true
	}

	return false
}

// Validates validator
func (v *Validator) ValidateBasic() bool {
	pk, ok := v.PubKey.GetCachedValue().(cryptotypes.PubKey)

	if !ok {
		return false
	}
	if bytes.Equal(pk.Bytes(), ZeroPubKey.Bytes()) {
		return false
	}

	zeroAddress := HeimdallAddress{(common.Address{}).Bytes()}.Address

	if bytes.Equal(v.Signer.GetAddress(), zeroAddress) {
		return false
	}

	return true
}

// amino marshall validator
func MarshallValidator(cdc codec.BinaryCodec, validator Validator) (bz []byte, err error) {
	bz, err = cdc.Marshal(&validator)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// amono unmarshall validator
func UnmarshallValidator(cdc codec.BinaryCodec, value []byte) (Validator, error) {
	var validator Validator

	if err := cdc.Unmarshal(value, &validator); err != nil {
		return validator, err
	}

	return validator, nil
}

// Copy creates a new copy of the validator so we can mutate accum.
// Panics if the validator is nil.
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
		result := bytes.Compare(v.Signer.GetAddress(), other.Signer.GetAddress())

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
func (v *Validator) Bytes() []byte {
	result := make([]byte, 64)

	copy(result[12:], v.Signer.GetAddress())
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
		ID:          v.ID,
		VotingPower: uint64(v.VotingPower),
		Signer:      v.Signer,
	}
}

// --------

// NewValidatorID generate new validator ID
func NewValidatorID(id uint64) ValidatorID {
	return ValidatorID{id}
}

// Bytes get bytes of validatorID
func (valID ValidatorID) Bytes() []byte {
	return []byte(strconv.FormatUint(valID.Uint64(), 10))
}

// Int converts validator ID to int
func (valID ValidatorID) Int() int {
	return int(valID.GetID())
}

// Uint64 converts validator ID to int
func (valID ValidatorID) Uint64() uint64 {
	return uint64(valID.GetID())
}

// --------

// MinimalVal is the minimal validator representation
// Used to send validator information to bor validator contract
type MinimalVal struct {
	ID          ValidatorID     `json:"ID"`
	VotingPower uint64          `json:"power"` // TODO add 10^-18 here so that we dont overflow easily
	Signer      HeimdallAddress `json:"signer"`
}

func (v Validator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(v.PubKey, &pk)
}
