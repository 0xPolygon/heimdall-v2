package keeper

import (
	"github.com/cosmos/gogoproto/grpc"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper      Keeper
	queryServer grpc.Server
	// legacySubspace exported.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper, queryServer grpc.Server) Migrator {
	return Migrator{
		keeper:      keeper,
		queryServer: queryServer,
		// legacySubspace: ss
	}
}
