package types

// TODO HV2: remove unused constants

// Checkpoint tags
var (
	EventTypeCheckpoint       = "checkpoint"
	EventTypeCheckpointAdjust = "checkpoint-adjust"
	EventTypeCheckpointAck    = "checkpoint-ack"
	EventTypeCheckpointNoAck  = "checkpoint-noack"

	AttributeKeyProposer    = "proposer"
	AttributeKeyStartBlock  = "start-block"
	AttributeKeyEndBlock    = "end-block"
	AttributeKeyHeaderIndex = "header-index"
	AttributeKeyNewProposer = "new-proposer"
	AttributeKeyRootHash    = "root-hash"
	AttributeKeyAccountHash = "account-hash"
	AttributeKeyHash        = "hash"
	AttributeValueCategory  = ModuleName
)
