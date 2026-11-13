package heimdalld

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/server"
)

func protectedPeerIDs(ctx *server.Context, additional []string) ([]string, error) {
	persistent, err := persistentServingIDs(ctx.Config.P2P.PersistentPeers)
	if err != nil {
		return nil, err
	}
	ids := append([]string(nil), additional...)
	ids = append(ids, privateServingIDs(ctx)...)
	ids = append(ids, persistent...)
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

func privateServingIDs(ctx *server.Context) []string {
	var ids []string
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
	return ids
}

func persistentServingIDs(addresses string) ([]string, error) {
	var ids []string
	for _, address := range strings.Split(addresses, ",") {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		if _, suffix, ok := strings.Cut(address, "://"); ok {
			address = suffix
		}
		id, _, ok := strings.Cut(address, "@")
		if !ok {
			return nil, fmt.Errorf("persistent peer address must contain a node ID")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
