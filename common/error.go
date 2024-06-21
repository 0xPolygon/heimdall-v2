package common

import (
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace = 1

	CodeInvalidMsg = 1400
	CodeOldTx      = 1401

	CodeInvalidProposerInput    = 1500
	CodeInvalidBlockInput       = 1501
	CodeInvalidACK              = 1502
	CodeNoACK                   = 1503
	CodeBadTimeStamp            = 1504
	CodeInvalidNoACK            = 1505
	CodeTooManyNoAck            = 1506
	CodeLowBal                  = 1507
	CodeNoCheckpoint            = 1508
	CodeOldCheckpoint           = 1509
	CodeDisContinuousCheckpoint = 1510
	CodeNoCheckpointBuffer      = 1511
	CodeCheckpointBuffer        = 1512
	CodeCheckpointAlreadyExists = 1513
	CodeInvalidNoAckProposer    = 1505

	CodeOldValidator        = 2500
	CodeNoValidator         = 2501
	CodeValSignerMismatch   = 2502
	CodeValidatorExitDeny   = 2503
	CodeValAlreadyUnbonded  = 2504
	CodeSignerSynced        = 2505
	CodeValSave             = 2506
	CodeValAlreadyJoined    = 2507
	CodeSignerUpdateError   = 2508
	CodeNoConn              = 2509
	CodeWaitFrConfirmation  = 2510
	CodeValPubkeyMismatch   = 2511
	CodeErrDecodeEvent      = 2512
	CodeNoSignerChangeError = 2513
	CodeNonce               = 2514

	CodeSpanNotContinuous   = 3501
	CodeUnableToFreezeSet   = 3502
	CodeSpanNotFound        = 3503
	CodeValSetMisMatch      = 3504
	CodeProducerMisMatch    = 3505
	CodeInvalidBorChainID   = 3506
	CodeInvalidSpanDuration = 3507

	CodeFetchCheckpointSigners       = 4501
	CodeErrComputeGenesisAccountRoot = 4503
	CodeAccountRootMismatch          = 4504

	CodeErrAccountRootHash     = 4505
	CodeErrSetCheckpointBuffer = 4506
	CodeErrAddCheckpoint       = 4507

	CodeInvalidReceipt         = 5501
	CodeSideTxValidationFailed = 5502

	CodeValSigningInfoSave     = 6501
	CodeErrValUnjail           = 6502
	CodeSlashInfoDetails       = 6503
	CodeTickNotInContinuity    = 6504
	CodeTickAckNotInContinuity = 6505

	CodeNoMilestone              = 7501
	CodeMilestoneNotInContinuity = 7502
	CodeMilestoneInvalid         = 7503
	CodeOldMilestone             = 7504
	CodeInvalidMilestoneTimeout  = 7505
	CodeTooManyMilestoneTimeout  = 7506
	CodeInvalidMilestoneIndex    = 7507
	CodePrevMilestoneInVoting    = 7508
)

// invalid msg

func ErrInvalidMsg(ModuleName string, format string) error {
	return errors.Register(ModuleName, CodeInvalidMsg, format)
}

// checkpoint Errors

func ErrBadProposerDetails(ModuleName string, proposer sdk.AccAddress) error {
	return errors.Register(ModuleName, CodeInvalidProposerInput, fmt.Sprintf("proposer is not valid, current proposer is %v", proposer.String()))
}

func ErrBadBlockDetails(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidBlockInput, "wrong root hash for given start and end block numbers")
}

func ErrSetCheckpointBuffer(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrSetCheckpointBuffer, "account root hash not added to checkpoint buffer")
}

func ErrAddCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrAddCheckpoint, "err in adding checkpoint to header blocks")
}

func ErrBadAccountRootHash(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrAccountRootHash, "wrong root hash for given dividend accounts")
}

func ErrBadAck(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidACK, "ack not valid")
}

func ErrOldCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldCheckpoint, "checkpoint already received for given start and end block")
}

func ErrDisContinuousCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeDisContinuousCheckpoint, "checkpoint not in continuity")
}

func ErrNoACK(ModuleName string, expiresAt uint64) error {
	return errors.Register(ModuleName, CodeNoACK, fmt.Sprintf("checkpoint already exists in buffer, ack expected, expires at %s", strconv.FormatUint(expiresAt, 10)))
}

func ErrNoConn(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoConn, "unable to connect to chain")
}

func ErrNoCheckpointFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoCheckpoint, "checkpoint not found")
}

func ErrCheckpointAlreadyExists(ModuleName string) error {
	return errors.Register(ModuleName, CodeCheckpointAlreadyExists, "checkpoint already exists")
}

func ErrNoCheckpointBufferFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoCheckpointBuffer, "checkpoint buffer not found")
}

func ErrCheckpointBufferFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeCheckpointBuffer, "checkpoint buffer found")
}

func ErrInvalidNoACK(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidNoACK, "invalid no ack -- waiting for last checkpoint ack")
}

func ErrInvalidNoACKProposer(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidNoAckProposer, "invalid no ack proposer")
}

func ErrTooManyNoACK(ModuleName string) error {
	return errors.Register(ModuleName, CodeTooManyNoAck, "too many no-acks")
}

func ErrBadTimeStamp(ModuleName string) error {
	return errors.Register(ModuleName, CodeBadTimeStamp, "invalid time stamp. It must be in near past.")
}

// Milestone Errors

func ErrNoMilestoneFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoMilestone, "milestone not found")
}

func ErrMilestoneNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeMilestoneNotInContinuity, "milestone not in continuity")
}

func ErrMilestoneInvalid(ModuleName string) error {
	return errors.Register(ModuleName, CodeMilestoneInvalid, "milestone msg invalid")
}

func ErrOldMilestone(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldMilestone, "milestone already exists")
}

func ErrInvalidMilestoneTimeout(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidMilestoneTimeout, "invalid milestone Timeout msg ")
}

func ErrTooManyMilestoneTimeout(ModuleName string) error {
	return errors.Register(ModuleName, CodeTooManyNoAck, "too many milestone timeout msg")
}

func ErrInvalidMilestoneIndex(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoMilestone, "invalid milestone index")
}

func ErrPrevMilestoneInVoting(ModuleName string) error {
	return errors.Register(ModuleName, CodePrevMilestoneInVoting, "previous milestone still in voting phase")
}

// Staking Errors

func ErrOldValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldValidator, "start epoch behind current epoch")
}

func ErrNoValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoValidator, "validator information not found")
}

func ErrNonce(ModuleName string) error {
	return errors.Register(ModuleName, CodeNonce, "incorrect validator nonce")
}

func ErrValSignerPubKeyMismatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValPubkeyMismatch, "signer pubkey mismatch between event and msg")
}

func ErrValSignerMismatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSignerMismatch, "signer address doesnt match pubkey address")
}

func ErrValIsNotCurrentVal(ModuleName string) error {
	return errors.Register(ModuleName, CodeValidatorExitDeny, "validator is not in validator set, exit not possible")
}

func ErrValUnbonded(ModuleName string) error {
	return errors.Register(ModuleName, CodeValAlreadyUnbonded, "validator already unbonded , cannot exit")
}

func ErrSignerUpdateError(ModuleName string) error {
	return errors.Register(ModuleName, CodeSignerUpdateError, "signer update error")
}

func ErrNoSignerChange(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoSignerChangeError, "new signer same as old signer")
}

func ErrOldTx(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldTx, "old txhash not allowed")
}

func ErrValidatorAlreadySynced(ModuleName string) error {
	return errors.Register(ModuleName, CodeSignerSynced, "no signer update found, invalid message")
}

func ErrValidatorSave(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSave, "cannot save validator")
}

func ErrValidatorNotDeactivated(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSave, "validator not deactivated")
}

func ErrValidatorAlreadyJoined(ModuleName string) error {
	return errors.Register(ModuleName, CodeValAlreadyJoined, "validator already joined")
}

// bor Errors --------------------------------

func ErrInvalidBorChainID(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidBorChainID, "invalid bor chain id")
}

func ErrSpanNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeSpanNotContinuous, "span not continuous")
}

func ErrInvalidSpanDuration(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidSpanDuration, "wrong span duration")
}

func ErrSpanNotFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeSpanNotFound, "span not found")
}

func ErrUnableToFreezeValSet(ModuleName string) error {
	return errors.Register(ModuleName, CodeUnableToFreezeSet, "unable to freeze validator set for next span")
}

func ErrValSetMisMatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSetMisMatch, "validator set mismatch")
}

func ErrProducerMisMatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeProducerMisMatch, "producer set mismatch")
}

func CodeToDefaultMsg(code uint32) string {
	switch code {
	case CodeInvalidMsg:
		return "invalid message"
	case CodeInvalidProposerInput:
		return "proposer is not valid"
	case CodeInvalidBlockInput:
		return "wrong root hash for given start and end block numbers"
	case CodeInvalidACK:
		return "ack not valid"
	case CodeNoACK:
		return "checkpoint already exists in buffer, ack expected"
	case CodeBadTimeStamp:
		return "invalid time stamp. It must be in near past."
	case CodeInvalidNoACK:
		return "invalid no ack -- waiting for last checkpoint ack"
	case CodeTooManyNoAck:
		return "too many no-acks"
	case CodeLowBal:
		return "insufficient balance"
	case CodeNoCheckpoint:
		return "checkpoint not found"
	case CodeOldCheckpoint:
		return "checkpoint already received for given start and end block"
	case CodeDisContinuousCheckpoint:
		return "checkpoint not in continuity"
	case CodeNoCheckpointBuffer:
		return "checkpoint buffer not found"
	case CodeOldValidator:
		return "start epoch behind current epoch"
	case CodeNoValidator:
		return "validator information not found"
	case CodeValSignerMismatch:
		return "signer address doesnt match pubkey address"
	case CodeValidatorExitDeny:
		return "validator is not in validator set, exit not possible"
	case CodeValAlreadyUnbonded:
		return "validator already unbonded , cannot exit"
	case CodeSignerSynced:
		return "no signer update found, invalid message"
	case CodeValSave:
		return "cannot save validator"
	case CodeValAlreadyJoined:
		return "validator already joined"
	case CodeSignerUpdateError:
		return "signer update error"
	case CodeNoConn:
		return "unable to connect to chain"
	case CodeWaitFrConfirmation:
		return "wait for confirmation time before sending transaction"
	case CodeValPubkeyMismatch:
		return "signer pubkey mismatch between event and msg"
	case CodeSpanNotContinuous:
		return "span not continuous"
	case CodeUnableToFreezeSet:
		return "unable to freeze validator set for next span"
	case CodeSpanNotFound:
		return "span not found"
	case CodeValSetMisMatch:
		return "validator set mismatch"
	case CodeProducerMisMatch:
		return "producer set mismatch"
	case CodeInvalidBorChainID:
		return "invalid bor chain id"
	default:
		return "default error"
	}
}

// Slashing errors

func ErrValidatorSigningInfoSave(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSigningInfoSave, "cannot save validator signing info")
}

func ErrUnjailValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrValUnjail, "error while unjail validator")
}

func ErrSlashInfoDetails(ModuleName string) error {
	return errors.Register(ModuleName, CodeSlashInfoDetails, "wrong slash info details")
}

func ErrTickNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeTickNotInContinuity, "tick not in continuity")
}

func ErrTickAckNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeTickAckNotInContinuity, "tick-ack not in continuity")
}
