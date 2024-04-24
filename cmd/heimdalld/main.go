package main

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/cmd/heimdalld/service"
)

func main() {
	service.NewHeimdallService(context.Background(), nil)
}
