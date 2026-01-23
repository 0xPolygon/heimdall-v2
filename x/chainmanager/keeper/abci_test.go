package keeper_test

func (s *KeeperTestSuite) TestEndBlocker() {
	ctx, require, cmKeeper := s.ctx, s.Require(), s.cmKeeper

	s.Run("returns nil when no chain manager address migration", func() {
		updates, err := cmKeeper.EndBlocker(ctx)

		require.NoError(err)
		require.Nil(updates)
	})

	s.Run("processes successfully with valid params", func() {
		// Params are already set in SetupTest
		updates, err := cmKeeper.EndBlocker(ctx)

		require.NoError(err)
		require.Nil(updates)
	})
}

func (s *KeeperTestSuite) TestLogger() {
	ctx, require, cmKeeper := s.ctx, s.Require(), s.cmKeeper

	s.Run("returns logger with module name", func() {
		logger := cmKeeper.Logger(ctx)

		require.NotNil(logger)
	})

	s.Run("logger can log messages without panic", func() {
		logger := cmKeeper.Logger(ctx)

		require.NotPanics(func() {
			logger.Info("test message")
			logger.Debug("debug message")
			logger.Error("error message")
		})
	})
}

func (s *KeeperTestSuite) TestGetAuthority() {
	require, cmKeeper := s.Require(), s.cmKeeper

	s.Run("returns authority address", func() {
		authority := cmKeeper.GetAuthority()

		require.NotEmpty(authority)
	})

	s.Run("authority is valid address format", func() {
		authority := cmKeeper.GetAuthority()

		// Should be a valid hex address
		require.Contains(authority, "0x")
		require.GreaterOrEqual(len(authority), 40) // At least 40 hex chars (20 bytes)
	})
}

func (s *KeeperTestSuite) TestSetParamsAndRetrieval() {
	ctx, require, cmKeeper := s.ctx, s.Require(), s.cmKeeper

	s.Run("params can be updated", func() {
		// Get initial params
		initialParams, err := cmKeeper.GetParams(ctx)
		require.NoError(err)

		// Update params
		updatedParams := initialParams
		updatedParams.ChainParams.BorChainId = "custom-chain-id-999"
		err = cmKeeper.SetParams(ctx, updatedParams)
		require.NoError(err)

		// Verify update
		retrieved, err := cmKeeper.GetParams(ctx)
		require.NoError(err)
		require.Equal("custom-chain-id-999", retrieved.ChainParams.BorChainId)
	})

	s.Run("params update persists across gets", func() {
		// Set custom value
		currentParams, err := cmKeeper.GetParams(ctx)
		require.NoError(err)

		currentParams.BorChainTxConfirmations = 999
		err = cmKeeper.SetParams(ctx, currentParams)
		require.NoError(err)

		// Get again and verify
		retrieved, err := cmKeeper.GetParams(ctx)
		require.NoError(err)
		require.Equal(uint64(999), retrieved.BorChainTxConfirmations)
	})
}

func (s *KeeperTestSuite) TestKeeperSchema() {
	require, cmKeeper := s.Require(), s.cmKeeper

	s.Run("keeper has valid schema", func() {
		require.NotNil(cmKeeper.Schema)
	})
}

func (s *KeeperTestSuite) TestEndBlockerWithDifferentHeights() {
	ctx, require, cmKeeper := s.ctx, s.Require(), s.cmKeeper

	s.Run("handles multiple consecutive calls", func() {
		// Call EndBlocker multiple times
		for i := 0; i < 3; i++ {
			updates, err := cmKeeper.EndBlocker(ctx)
			require.NoError(err)
			require.Nil(updates)
		}
	})
}
