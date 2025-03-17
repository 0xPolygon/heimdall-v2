package engine

import (
	"context"

	gethEngine "github.com/ethereum/go-ethereum/beacon/engine"
)

type ExecutionEngineClient interface {
	ForkchoiceUpdatedV2(ctx context.Context, state *gethEngine.ForkchoiceStateV1, attrs *gethEngine.PayloadAttributes) (*gethEngine.ForkChoiceResponse, error)
	GetPayloadV2(ctx context.Context, payloadId string) (*gethEngine.ExecutionPayloadEnvelope, error)
	NewPayloadV2(ctx context.Context, payload gethEngine.ExecutableData) (*gethEngine.PayloadStatusV1, error)
	CheckCapabilities(ctx context.Context, requiredMethods []string) error
	Close()
}
