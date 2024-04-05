package clerk_test

// TODO HV2 - uncomment after the app tests are fixed
/*
// GenesisTestSuite integrate test suite context object
type GenesisTestSuite struct {
	suite.Suite

	app *app.HeimdallApp
	ctx sdk.Context
}

// SetupTest setup necessary things for genesis test
func (suite *GenesisTestSuite) SetupTest() {
	suite.app = setupClerkGenesis(suite.T())
	suite.ctx = suite.app.BaseApp.NewContext(true)
}

// TestGenesisTestSuite
func TestGenesisTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GenesisTestSuite))
}

// TestInitExportGenesis test import and export genesis state
func (suite *GenesisTestSuite) TestInitExportGenesis() {
	t, happ, ctx := suite.T(), suite.app, suite.ctx
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	recordSequences := make([]string, 5)
	eventRecords := make([]*types.EventRecord, 1)

	for i := range recordSequences {
		recordSequences[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}

	// TODO HV2 - use real and meaningful data
	hAddr := strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	hHash, _ := hexCodec.NewHexCodec().StringToBytes(strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000)))
	testEventRecord := types.NewEventRecord(hmTypes.HeimdallHash{Hash: hHash}, uint64(0), uint64(0), hAddr, hmTypes.HexBytes{HexBytes: make([]byte, 0)}, strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000)), time.Now())
	eventRecords[0] = &testEventRecord

	genesisState := types.GenesisState{
		EventRecords:    eventRecords,
		RecordSequences: recordSequences,
	}
	clerk.InitGenesis(ctx, &happ.ClerkKeeper, &genesisState)

	actualParams := clerk.ExportGenesis(ctx, &happ.ClerkKeeper)

	require.Equal(t, len(recordSequences), len(actualParams.RecordSequences))
	require.Equal(t, len(eventRecords), len(actualParams.EventRecords))
}
*/
