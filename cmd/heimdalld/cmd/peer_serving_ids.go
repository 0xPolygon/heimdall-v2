package heimdalld

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/server"
)

func protectedPeerIDs(ctx *server.Context, additional []string) ([]string, error) {
	ids := append([]string(nil), additional...)
	for _, id := range strings.Split(ctx.Config.P2P.PrivatePeerIDs, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := hex.DecodeString(id); err != nil || len(id) != 40 {
			ctx.Logger.Warn("Ignoring invalid private peer ID for reserved serving")
			continue
		}
		ids = append(ids, id)
	}
	for _, address := range strings.Split(ctx.Config.P2P.PersistentPeers, ",") {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		id, _, ok := strings.Cut(address, "@")
		if !ok {
			return nil, fmt.Errorf("persistent peer address must contain a node ID")
		}
		ids = append(ids, id)
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]bool)
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	return unique, nil
}
