package keeper

import (
	"reflect"
	"testing"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var TestValidators = []staketypes.Validator{
	{
		ValId:       3,
		StartEpoch:  0,
		EndEpoch:    0,
		VotingPower: 10000,
		PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
		Signer:      "0x1c4f0f054a0d6a1415382dc0fd83c6535188b220",
		LastUpdated: "0",
	},
	{
		ValId:       4,
		StartEpoch:  0,
		EndEpoch:    0,
		VotingPower: 10000,
		PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
		Signer:      "0x461295d3d9249215e758e939a150ab180950720b",
		LastUpdated: "0",
	},
	{
		ValId:       5,
		StartEpoch:  0,
		EndEpoch:    0,
		VotingPower: 10000,
		PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
		Signer:      "0x836fe3e3dd0a5f77d9d5b0f67e48048aaafcd5a0",
		LastUpdated: "0",
	},
	{
		ValId:       1,
		StartEpoch:  0,
		EndEpoch:    0,
		VotingPower: 10000,
		PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
		Signer:      "0x925a91f8003aaeabea6037103123b93c50b86ca3",
		LastUpdated: "0",
	},
	{
		ValId:       2,
		StartEpoch:  0,
		EndEpoch:    0,
		VotingPower: 10000,
		PubKey:      secp256k1.GenPrivKey().PubKey().Bytes(),
		Signer:      "0xc787af4624cb3e80ee23ae7faac0f2acea2be34c",
		LastUpdated: "0",
	},
}

func TestSelectNextProducers(t *testing.T) {
	t.Parallel()

	type producerSelectionTestCase struct {
		seed            string
		producerCount   uint64
		resultSlots     int64
		resultProducers int64
	}

	testcases := []producerSelectionTestCase{
		{"0x8f5bab218b6bb34476f51ca588e9f4553a3a7ce5e13a66c660a5283e97e9a85a", 10, 5, 5},
		{"0x8f5bab218b6bb34476f51ca588e9f4553a3a7ce5e13a66c660a5283e97e9a85a", 5, 5, 5},
		{"0xe09cc356df20c7a2dd38cb85b680a16ec29bd8b3e1ecc1b20f2e5603d5e7ee85", 10, 5, 5},
		{"0xe09cc356df20c7a2dd38cb85b680a16ec29bd8b3e1ecc1b20f2e5603d5e7ee85", 5, 5, 5},
		{"0x8f5bab218b6bb34476f51ca588e9f4553a3a7ce5e13a66c660a5283e97e9a85a", 4, 4, 3},
		{"0xe09cc356df20c7a2dd38cb85b680a16ec29bd8b3e1ecc1b20f2e5603d5e7ee85", 4, 4, 4},
	}

	for i, testcase := range testcases {
		seed := common.HexToHash(testcase.seed)

		producerIds := selectNextProducers(seed, TestValidators, testcase.producerCount)
		require.NoError(t, nil)

		producers, slots := getSelectedValidatorsFromIDs(TestValidators, producerIds)
		require.Equal(t, testcase.resultSlots, slots, "Total slots should be %v (Testcase %v)", testcase.resultSlots, i+1)
		require.Equal(t, int(testcase.resultProducers), len(producers), "Total producers should be %v (Testcase %v)", testcase.resultProducers, i+1)
	}
}

func getSelectedValidatorsFromIDs(validators []staketypes.Validator, producerIds []uint64) ([]staketypes.Validator, int64) {
	var vals []staketypes.Validator

	IDToPower := make(map[uint64]uint64)
	for _, ID := range producerIds {
		IDToPower[ID] = IDToPower[ID] + 1
	}

	var slots int64

	for key, value := range IDToPower {
		if val, ok := findValidatorByID(validators, key); ok {
			val.VotingPower = int64(value)
			vals = append(vals, val)
			slots = slots + int64(value)
		}
	}

	return vals, slots
}

func findValidatorByID(validators []staketypes.Validator, id uint64) (val staketypes.Validator, ok bool) {
	for _, v := range validators {
		if v.ValId == id {
			return v, true
		}
	}

	return
}

func TestCreateWeightedRanges(t *testing.T) {
	t.Parallel()

	type args struct {
		vals []uint64
	}

	tests := []struct {
		name        string
		args        args
		ranges      []uint64
		totalWeight uint64
	}{
		{
			args: args{
				vals: []uint64{30, 20, 50, 50, 1},
			},
			ranges:      []uint64{30, 50, 100, 150, 151},
			totalWeight: 151,
		},
		{
			args: args{
				vals: []uint64{1, 2, 1, 2, 1},
			},
			ranges:      []uint64{1, 3, 4, 6, 7},
			totalWeight: 7,
		},
		{
			args: args{
				vals: []uint64{10, 1, 20, 1, 2},
			},
			ranges:      []uint64{10, 11, 31, 32, 34},
			totalWeight: 34,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ranges, totalWeight := createWeightedRanges(tt.args.vals)
			if !reflect.DeepEqual(ranges, tt.ranges) {
				t.Errorf("createWeightedRange() got ranges = %v, want %v", ranges, tt.ranges)
			}

			if totalWeight != tt.totalWeight {
				t.Errorf("createWeightedRange() got totalWeight = %v, want %v", totalWeight, tt.totalWeight)
			}
		})
	}
}

func TestBinarySearch(t *testing.T) {
	t.Parallel()

	type args struct {
		array  []uint64
		search uint64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			args: args{
				array:  []uint64{},
				search: 0,
			},
			want: -1,
		},
		{
			args: args{
				array:  []uint64{1},
				search: 100,
			},
			want: 0,
		},
		{
			args: args{
				array:  []uint64{1, 1000},
				search: 100,
			},
			want: 1,
		},
		{
			args: args{
				array:  []uint64{1, 100, 1000},
				search: 2,
			},
			want: 1,
		},
		{
			args: args{
				array:  []uint64{1, 100, 1000, 1000},
				search: 1001,
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := binarySearch(tt.args.array, tt.args.search); got != tt.want {
				t.Errorf("binarySearch() = %v, want %v", got, tt.want)
			}
		})
	}
}
