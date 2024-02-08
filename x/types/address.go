package types

// TODO HV2
// Check for address

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// AddrLen defines a valid address length
	AddrLen = 20
)

// Ensure that different address types implement the interface
// TODO Do we still need sdk.Address check
// var _ sdk.Address = HeimdallAddress{}
var _ yaml.Marshaler = HeimdallAddress{}

// ZeroHeimdallAddress represents zero address
// common.Address gives any empty bytes array of size 20
var ZeroHeimdallAddress = HeimdallAddress{common.Address{}.Bytes()}

// EthAddress get eth address
func (aa HeimdallAddress) EthAddress() common.Address {
	return common.Address(aa.GetAddress())
}

// Equals returns boolean for whether two AccAddresses are Equal
func (aa HeimdallAddress) Equals(aa2 sdk.Address) bool {
	if aa.Empty() && aa2.Empty() {
		return true
	}

	return bytes.Equal(aa.Bytes(), aa2.Bytes())
}

// Empty returns boolean for whether an AccAddress is empty
func (aa HeimdallAddress) Empty() bool {
	return bytes.Equal(aa.Bytes(), ZeroHeimdallAddress.Bytes())
}

// MarshalJSON marshals to JSON using Bech32.
func (aa HeimdallAddress) MarshalJSON() ([]byte, error) {
	return jsoniter.ConfigFastest.Marshal(aa.String())
}

// MarshalYAML marshals to YAML using Bech32.
func (aa HeimdallAddress) MarshalYAML() (interface{}, error) {
	return aa.String(), nil
}

// UnmarshalJSON unmarshals from JSON assuming Bech32 encoding.
func (aa *HeimdallAddress) UnmarshalJSON(data []byte) error {
	var s string
	if err := jsoniter.ConfigFastest.Unmarshal(data, &s); err != nil {
		return err
	}

	*aa = HexToHeimdallAddress(s)

	return nil
}

// UnmarshalYAML unmarshals from JSON assuming Bech32 encoding.
func (aa *HeimdallAddress) UnmarshalYAML(data []byte) error {
	var s string
	if err := yaml.Unmarshal(data, &s); err != nil {
		return err
	}

	*aa = HexToHeimdallAddress(s)

	return nil
}

// Bytes returns the raw address bytes.
func (aa HeimdallAddress) Bytes() []byte {
	return aa.GetAddress()
}

// String implements the Stringer interface.
func (aa HeimdallAddress) HexString() string {
	return "0x" + aa.String()
}

// Format implements the fmt.Formatter interface.
// nolint: errcheck
func (aa HeimdallAddress) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(aa.String()))
	case 'p':
		s.Write([]byte(fmt.Sprintf("%p", aa)))
	default:
		s.Write([]byte(fmt.Sprintf("%X", aa.Bytes())))
	}
}

//
// Address utils
//

// BytesToHeimdallAddress returns Address with value b.
func BytesToHeimdallAddress(b []byte) HeimdallAddress {
	return HeimdallAddress{b}
}

// HexToHeimdallAddress returns Address with value b.
func HexToHeimdallAddress(b string) HeimdallAddress {
	return HeimdallAddress{common.HexToAddress(b).Bytes()}
}

// AccAddressToHeimdallAddress returns Address with value b.
func AccAddressToHeimdallAddress(b sdk.AccAddress) HeimdallAddress {
	return BytesToHeimdallAddress(b[:])
}

// HeimdallAddressToAccAddress returns Address with value b.
func HeimdallAddressToAccAddress(b HeimdallAddress) sdk.AccAddress {
	return sdk.AccAddress(b.Bytes())
}

// SampleHeimdallAddress returns sample address
func SampleHeimdallAddress(s string) HeimdallAddress {
	return BytesToHeimdallAddress([]byte(s))
}
