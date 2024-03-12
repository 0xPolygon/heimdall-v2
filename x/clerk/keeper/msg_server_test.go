package keeper_test

// TODO HV2 - uncomment and fix unit tests

// func (suite *KeeperTestSuite) TestHandleMsgEventRecord() {
// 	// TODO HV2 - uncomment when heimdall app PR is merged
// 	// t, app, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r
// 	t, _, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r

// 	addr1 := sdk.AccAddress([]byte("addr1"))

// 	id := r.Uint64()
// 	logIndex := r.Uint64()
// 	blockNumber := r.Uint64()

// 	// successful message
// 	msg := types.NewMsgEventRecord(
// 		addr1,
// 		// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
// 		// hexCodec.StringToBytes("123"),
// 		hmTypes.HeimdallHash{},
// 		logIndex,
// 		blockNumber,
// 		id,
// 		addr1,
// 		hmTypes.HexBytes{},
// 		chainID,
// 	)

// 	t.Run("Success", func(t *testing.T) {
// 		result := suite.handler(ctx, msg)
// 		require.True(t, result.IsOK(), "expected msg record to be ok, got %v", result)

// 		// there should be no stored event record
// 		storedEventRecord, err := app.ClerkKeeper.GetEventRecord(ctx, id)
// 		require.Nil(t, storedEventRecord)
// 		require.Error(t, err)
// 	})

// 	t.Run("ExistingRecord", func(t *testing.T) {
// 		// TODO HV2 - uncomment when heimdall app PR is merged
// 		// // store event record in keeper
// 		// tempTime := time.Now()
// 		// err := app.ClerkKeeper.SetEventRecord(ctx,
// 		// 	types.NewEventRecord(
// 		// 		msg.TxHash,
// 		// 		msg.LogIndex,
// 		// 		msg.ID,
// 		// 		msg.ContractAddress,
// 		// 		msg.Data,
// 		// 		msg.ChainID,
// 		// 		tempTime,
// 		// 	),
// 		// )
// 		// require.NoError(t, err)

// 		result, err := suite.msgServer.HandleMsgEventRecord(ctx, &msg)
// 		require.False(t, result.IsOK(), "should fail due to existent event record but succeeded")
// 		require.Equal(t, types.CodeEventRecordAlreadySynced, result.Code)
// 	})

// 	t.Run("EventSizeExceed", func(t *testing.T) {
// 		suite.contractCaller = mocks.IContractCaller{}

// 		const letterBytes = "abcdefABCDEF"
// 		b := make([]byte, helper.LegacyMaxStateSyncSize+3)
// 		for i := range b {
// 			b[i] = letterBytes[rand.Intn(len(letterBytes))]
// 		}

// 		msg.Data = b

// 		err := msg.ValidateBasic()
// 		require.Error(t, err)
// 	})
// }

// func (suite *KeeperTestSuite) TestHandleMsgEventRecordSequence() {
// 	// TODO HV2 - uncomment when heimdall app PR is merged
// 	// t, app, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r
// 	t, _, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r

// 	addr1 := sdk.AccAddress([]byte("addr1"))

// 	msg := types.NewMsgEventRecord(
// 		addr1,
// 		// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
// 		// hexCodec.StringToBytes("123"),
// 		hmTypes.HeimdallHash{},
// 		r.Uint64(),
// 		r.Uint64(),
// 		r.Uint64(),
// 		addr1,
// 		hmTypes.HexBytes{},
// 		chainID,
// 	)

// 	// sequence id
// 	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
// 	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
// 	app.ClerkKeeper.SetRecordSequence(ctx, sequence.String())

// 	result, err := suite.msgServer.HandleMsgEventRecord(ctx, &msg)
// 	require.False(t, result.IsOK(), "should fail due to existent sequence but succeeded")
// 	require.Equal(t, common.CodeOldTx, result.Code)
// }

// func (suite *KeeperTestSuite) TestHandleMsgEventRecordChainID() {
// 	// TODO HV2 - uncomment when heimdall app PR is merged
// 	// t, app, ctx, r := suite.T(), suite.app, suite.ctx, suite.r
// 	t, _, ctx, r := suite.T(), suite.app, suite.ctx, suite.r

// 	addr1 := sdk.AccAddress([]byte("addr1"))

// 	id := r.Uint64()

// 	// wrong chain id
// 	msg := types.NewMsgEventRecord(
// 		addr1,
// 		// TODO HV2 - uncomment when auth PR is merged and hexCodec is implemented
// 		// hexCodec.StringToBytes("123"),
// 		hmTypes.HeimdallHash{},
// 		r.Uint64(),
// 		r.Uint64(),
// 		id,
// 		addr1,
// 		hmTypes.HexBytes{},
// 		"random chain id",
// 	)
// 	result, err := suite.msgServer.HandleMsgEventRecord(ctx, &msg)
// 	require.False(t, result.IsOK(), "error invalid bor chain id %v", result.Code)
// 	require.Equal(t, common.CodeInvalidBorChainID, result.Code)

// 	// TODO HV2 - uncomment when heimdall app PR is merged
// 	// // there should be no stored event record
// 	// storedEventRecord, err := app.ClerkKeeper.GetEventRecord(ctx, id)
// 	// require.Nil(t, storedEventRecord)
// 	// require.Error(t, err)
// }
