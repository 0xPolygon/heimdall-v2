package types

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/assert"
)

// TestSortValidatorByAddress populates only the signer as that is the only value used in sorting
func TestSortValidatorByAddress(t *testing.T) {
	t.Parallel()

	tc := []struct {
		in  []Validator
		out []Validator
		msg string
	}{
		{
			in: []Validator{
				{Signer: "dummyAddress3"},
				{Signer: "dummyAddress2"},
				{Signer: "dummyAddress1"},
			},
			out: []Validator{
				{Signer: "dummyAddress1"},
				{Signer: "dummyAddress2"},
				{Signer: "dummyAddress3"},
			},
			msg: "reverse sorting of validator objects",
		},
	}
	for i, c := range tc {
		out := SortValidatorByAddress(c.in)
		assert.Equal(t, c.out, out, fmt.Sprintf("i: %v, case: %v", i, c.msg))
	}
}

func TestValidateBasic(t *testing.T) {
	t.Parallel()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	accounts := simulation.RandomAccounts(r1, n)

	pks := make([][]byte, n)
	for i, acc := range accounts {
		pks[i] = acc.PubKey.Bytes()
	}

	tc := []struct {
		in      Validator
		expFail bool
		msg     string
	}{
		{
			in:      Validator{StartEpoch: 1, EndEpoch: 5, Nonce: 0, PubKey: pks[0], Signer: accounts[0].PubKey.Address().String()},
			expFail: false,
			msg:     "Valid basic validator test",
		},
		{
			in:      Validator{StartEpoch: 1, EndEpoch: 1, Nonce: 0, PubKey: pks[3], Signer: ""},
			expFail: true,
			msg:     "invalid",
		},
	}

	for _, c := range tc {
		fmt.Println(c.in.Signer)
		out := c.in.ValidateBasic()
		if c.expFail {
			assert.NotNil(t, out)
			assert.Contains(t, out.Error(), c.msg)
		} else {
			assert.Equal(t, nil, out, c.msg)
		}
	}
}
