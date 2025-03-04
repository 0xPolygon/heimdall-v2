package types

import (
	"encoding/hex"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Milestone tags
var (
	AttributeKeyProposer      = "proposer"
	AttributeKeyStartBlock    = "start-block"
	AttributeKeyEndBlock      = "end-block"
	AttributeKeyNewProposer   = "new-proposer"
	AttributeKeyHash          = "hash"
	AttributeKeyMilestoneID   = "milestone-id"
	AttributeValueCategory    = ModuleName
	EventTypeMilestone        = "milestone"
	EventTypeMilestoneTimeout = "milestone-timeout"
)

// NewMilestoneEvent construct a new coin minted sdk.Event
func NewMilestoneEvent(milestone Milestone, milestoneNumber uint64) sdk.Event {
	return sdk.NewEvent(
		EventTypeMilestone,
		sdk.NewAttribute("hash", string(milestone.Hash)),
		sdk.NewAttribute("proposer", milestone.Proposer),
		sdk.NewAttribute("hash", hex.EncodeToString(milestone.Hash)),
		sdk.NewAttribute("start_block", strconv.FormatUint(milestone.StartBlock, 10)),
		sdk.NewAttribute("end_block", strconv.FormatUint(milestone.EndBlock, 10)),
		sdk.NewAttribute("bor_chain_id", milestone.BorChainId),
		sdk.NewAttribute("milestone_id", milestone.MilestoneId),
		sdk.NewAttribute("timestamp", strconv.FormatUint(milestone.Timestamp, 10)),
		sdk.NewAttribute("number", strconv.FormatUint(milestoneNumber, 10)),
	)
}
