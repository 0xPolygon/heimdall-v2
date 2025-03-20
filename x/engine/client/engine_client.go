package client

import "context"

type ExecutionEngineClient interface {
	ForkchoiceUpdatedV2(ctx context.Context, state *ForkChoiceState, attrs *PayloadAttributes) (*ForkchoiceUpdatedResponse, error)
	GetPayloadV2(ctx context.Context, payloadId string) (*Payload, error)
	NewPayloadV2(ctx context.Context, payload ExecutionPayload) (*NewPayloadResponse, error)
	CheckCapabilities(ctx context.Context, requiredMethods []string) error
	Close()
}
