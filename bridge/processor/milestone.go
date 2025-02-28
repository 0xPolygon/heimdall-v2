package processor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// MilestoneProcessor - process milestone related events
type MilestoneProcessor struct {
	BaseProcessor

	// header listener subscription
	cancelMilestoneService context.CancelFunc
}

// MilestoneContext represents milestone context
type MilestoneContext struct {
	ChainmanagerParams *chainmanagerTypes.Params
}

// Start starts new block subscription
func (mp *MilestoneProcessor) Start() error {
	// mp.Logger.Info("Starting")

	// // create cancellable context
	// milestoneCtx, cancelMilestoneService := context.WithCancel(context.Background())

	// mp.cancelMilestoneService = cancelMilestoneService

	// // start polling for milestone
	// mp.Logger.Info("Start polling for milestone", "milestoneLength", helper.MilestoneLength, "pollInterval", helper.GetConfig().MilestonePollInterval)

	// go mp.startPolling(milestoneCtx, helper.MilestoneLength, helper.GetConfig().MilestonePollInterval)
	// go mp.startPollingMilestoneTimeout(milestoneCtx, 2*helper.GetConfig().MilestonePollInterval)

	return nil
}

// RegisterTasks - nil
func (mp *MilestoneProcessor) RegisterTasks() {
}

// startPolling - polls heimdall and checks if new milestone needs to be proposed
func (mp *MilestoneProcessor) startPolling(ctx context.Context, milestoneLength uint64, interval time.Duration) {
	ticker := time.NewTicker(interval)
	// stop ticker when everything done
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := mp.checkAndPropose(ctx, milestoneLength)
			if err != nil {
				mp.Logger.Error("Error in proposing the milestone", "error", err)
			}
		case <-ctx.Done():
			mp.Logger.Info("Polling stopped")
			return
		}
	}
}

// sendMilestoneToHeimdall - handles header block from bor
// 1. check if i am the proposer for next milestone
// 2. check if milestone has to be proposed
// 3. if so, propose milestone to heimdall.
func (mp *MilestoneProcessor) checkAndPropose(ctx context.Context, milestoneLength uint64) (err error) {
	// fetch milestone context
	milestoneContext, err := mp.getMilestoneContext()
	if err != nil {
		return err
	}

	// check whether the node is current milestone proposer or not
	isProposer, err := util.IsMilestoneProposer(mp.cliCtx.Codec)
	if err != nil {
		mp.Logger.Error("Error checking isProposer in HeaderBlock handler", "error", err)
		return err
	}

	if isProposer {
		result, err := util.GetMilestoneCount(mp.cliCtx.Codec)
		if err != nil {
			return err
		}

		start := helper.GetMilestoneBorBlockHeight()

		if result != 0 {
			// fetch latest milestone
			latestMilestone, err := util.GetLatestMilestone(mp.cliCtx.Codec)
			if err != nil {
				return err
			}

			if latestMilestone == nil {
				return errors.New("got nil result while fetching latest milestone")
			}

			// start block number should be continuous to the end block of lasted stored milestone
			start = latestMilestone.EndBlock + 1
		}

		// send the milestone to heimdall chain
		if err := mp.createAndSendMilestoneToHeimdall(ctx, milestoneContext, start, milestoneLength); err != nil {
			mp.Logger.Error("Error sending milestone to heimdall", "error", err)
			return err
		}
	} else {
		mp.Logger.Info("I am not the current milestone proposer")
	}

	return nil
}

// sendMilestoneToHeimdall - creates milestone msg and broadcasts to heimdall
func (mp *MilestoneProcessor) createAndSendMilestoneToHeimdall(ctx context.Context, milestoneContext *MilestoneContext, startNum uint64, milestoneLength uint64) error {
	mp.Logger.Debug("Initiating milestone to Heimdall", "start", startNum, "milestoneLength", milestoneLength)

	blocksConfirmation := helper.BorChainMilestoneConfirmation

	// Get latest bor block
	block, err := mp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		return err
	}

	latestNum := block.Number.Uint64()

	if latestNum < startNum+milestoneLength+blocksConfirmation-1 {
		mp.Logger.Debug(fmt.Sprintf("less than milestoneLength  start=%v latest block=%v milestonelength=%v borchainconfirmation=%v", startNum, latestNum, milestoneLength, blocksConfirmation))
		return nil
	}

	endNum := latestNum - blocksConfirmation

	// fetch the endBlock+1 number instead of endBlock so that we can directly get the hash of endBlock using parent hash
	block, err = mp.contractCaller.GetBorChainBlock(ctx, big.NewInt(int64(endNum+1)))
	if err != nil {
		return fmt.Errorf("error while fetching %d block %w", endNum+1, err)
	}

	endHash := block.ParentHash

	newRandUuid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	addressString, err := helper.GetAddressString()
	if err != nil {
		return fmt.Errorf("error converting address to string: %w", err)
	}

	milestoneId := fmt.Sprintf("%s - %s", newRandUuid.String(), addressString)

	mp.Logger.Info("End block hash", common.Bytes2Hex(endHash[:]))

	mp.Logger.Info("✅ Creating and broadcasting new milestone",
		"start", startNum,
		"end", endNum,
		"hash", common.Bytes2Hex(endHash[:]),
		"milestoneId", milestoneId,
		"milestoneLength", milestoneLength,
	)

	chainParams := milestoneContext.ChainmanagerParams.ChainParams

	address, err := helper.GetAddressString()
	if err != nil {
		return fmt.Errorf("error converting address to string: %w", err)
	}

	// create and send milestone message
	msg := milestoneTypes.NewMsgMilestoneBlock(
		address,
		startNum,
		endNum,
		endHash[:],
		chainParams.BorChainId,
		milestoneId,
	)

	// broadcast to heimdall
	txRes, err := mp.txBroadcaster.BroadcastToHeimdall(msg, nil) //nolint:contextcheck
	if err != nil {
		mp.Logger.Error("Error while broadcasting milestone to heimdall", "error", err)
		return err
	}

	if txRes.Code != abci.CodeTypeOK {
		mp.Logger.Error("milestone tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
		return fmt.Errorf("milestone tx failed, tx response code: %v", txRes.Code)
	}

	return nil
}

// startPolling - polls heimdall and checks if new milestoneTimeout needs to be proposed
func (mp *MilestoneProcessor) startPollingMilestoneTimeout(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	// stop ticker when everything done
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := mp.checkAndProposeMilestoneTimeout(ctx)
			if err != nil {
				mp.Logger.Error("Error in proposing the MilestoneTimeout msg", "error", err)
			}
		case <-ctx.Done():
			mp.Logger.Info("Polling stopped")
			ticker.Stop()

			return
		}
	}
}

// sendMilestoneToHeimdall - handles header block from bor
// 1. check if i am the proposer for next milestone
// 2. check if milestone has to be proposed
// 3. if so, propose milestone to heimdall.
func (mp *MilestoneProcessor) checkAndProposeMilestoneTimeout(ctx context.Context) (err error) {
	isMilestoneTimeoutRequired, err := mp.checkIfMilestoneTimeoutIsRequired(ctx)
	if err != nil {
		mp.Logger.Debug("Error checking sMilestoneTimeoutRequired while proposing Milestone Timeout ", "error", err)
		return
	}

	if isMilestoneTimeoutRequired {
		var isProposer bool

		// check if the node is the proposer list or not.
		if isProposer, err = util.IsInMilestoneProposerList(10, mp.cliCtx.Codec); err != nil {
			mp.Logger.Error("Error checking IsInMilestoneProposerList while proposing Milestone Timeout ", "error", err)
			return
		}

		// if i am the proposer and NoAck is required, then propose No-Ack
		if isProposer {
			// send Checkpoint No-Ack to heimdall
			//nolint:contextcheck
			if err = mp.createAndSendMilestoneTimeoutToHeimdall(); err != nil {
				mp.Logger.Error("Error proposing Milestone-Timeout ", "error", err)
				return
			}
		}
	}

	return nil
}

// sendMilestoneTimeoutToHeimdall - creates milestone-timeout msg and broadcasts to heimdall
func (mp *MilestoneProcessor) createAndSendMilestoneTimeoutToHeimdall() error {
	mp.Logger.Debug("Initiating milestone timeout to Heimdall")

	mp.Logger.Info("✅ Creating and broadcasting milestone-timeout")

	address, err := helper.GetAddressString()
	if err != nil {
		return fmt.Errorf("error converting address to string: %w", err)
	}

	// create and send milestone message
	msg := milestoneTypes.NewMsgMilestoneTimeout(
		address,
	)

	// return broadcast to heimdall
	txRes, err := mp.txBroadcaster.BroadcastToHeimdall(msg, nil)
	if err != nil {
		mp.Logger.Error("Error while broadcasting milestone timeout to heimdall", "error", err)
		return err
	}

	if txRes.Code != abci.CodeTypeOK {
		mp.Logger.Error("milestone timeout tx failed on heimdall", "txHash", txRes.TxHash, "code", txRes.Code)
		return fmt.Errorf("milestone timeout tx failed, tx response code: %v", txRes.Code)
	}

	return nil
}

func (mp *MilestoneProcessor) checkIfMilestoneTimeoutIsRequired(ctx context.Context) (bool, error) {
	latestMilestone, err := util.GetLatestMilestone(mp.cliCtx.Codec)
	if err != nil || latestMilestone == nil {
		return false, err
	}

	lastMilestoneEndBlock := latestMilestone.EndBlock
	currentChildBlockNumber, err := mp.getCurrentChildBlock(ctx)
	if err != nil {
		return false, err
	}

	if (currentChildBlockNumber - lastMilestoneEndBlock) > helper.MilestoneBufferLength {
		return true, nil
	}

	return false, nil
}

// getCurrentChildBlock gets the current child block
func (mp *MilestoneProcessor) getCurrentChildBlock(ctx context.Context) (uint64, error) {
	childBlock, err := mp.contractCaller.GetBorChainBlock(ctx, nil)
	if err != nil {
		return 0, err
	}

	return childBlock.Number.Uint64(), nil
}

func (mp *MilestoneProcessor) getMilestoneContext() (*MilestoneContext, error) {
	chainmanagerParams, err := util.GetChainmanagerParams(mp.cliCtx.Codec)
	if err != nil {
		mp.Logger.Error("Error while fetching chain manager params", "error", err)
		return nil, err
	}

	return &MilestoneContext{
		ChainmanagerParams: chainmanagerParams,
	}, nil
}

// Stop stops all necessary go routines
func (mp *MilestoneProcessor) Stop() {
	// cancel milestone polling
	mp.cancelMilestoneService()
}
