package common

import (
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO HV2: refactor and complete error codes to use them in the codebase

const (
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
	return errors.Register(ModuleName, CodeValSignerMismatch, "signer address doesn't match pubKey address")
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
