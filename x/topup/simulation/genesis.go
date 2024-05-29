package simulation

import (
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/0xPolygon/heimdall-v2/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

var SequenceNumber = "sequence_number"

// GenSequenceNumber returns a random sequence number
func GenSequenceNumber(r *rand.Rand) string {
	return strconv.Itoa(simulation.RandIntBetween(r, 0, math.MaxInt32))
}

// RandomizeGenState returns a simulated topup genesis
func RandomizeGenState(simState *module.SimulationState) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	minAccounts := 1
	maxAccounts := 50
	numAccounts := rand.Intn(maxAccounts-minAccounts) + minAccounts
	accounts := simulation.RandomAccounts(r1, numAccounts)

	var (
		sequences        = make([]string, 5)
		dividendAccounts = make([]types.DividendAccount, 5)
		sequenceNumber   string
	)

	for i := 0; i < 5; i++ {

		simState.AppParams.GetOrGenerate(SequenceNumber, &sequenceNumber, simState.Rand, func(r *rand.Rand) {
			sequenceNumber = GenSequenceNumber(r)
		})

		sequences[i] = sequenceNumber

		// create dividend account for validator
		dividendAccounts[i] = types.DividendAccount{
			User:      accounts[i].Address.String(),
			FeeAmount: big.NewInt(0).String(),
		}
	}

	topupGenesis := topupTypes.NewGenesisState(sequences, dividendAccounts)
	simState.GenState[topupTypes.ModuleName] = simState.Cdc.MustMarshalJSON(topupGenesis)
}
