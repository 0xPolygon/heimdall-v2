package app

import (
	"encoding/json"
)

// GenesisState of the blockchain is represented here as a map of raw JSON
// messages keyed by an identifier string.
// The identifier is used to determine which module genesis information belongs to,
// so it may be appropriately routed during the init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager, which populates JSON from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage
