package common

import (
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultModuleName string = "1"

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

// -------- Invalid msg

func ErrInvalidMsg(ModuleName string, format string) error {
	return errors.Register(ModuleName, CodeInvalidMsg, format)
}

// -------- Checkpoint Errors

func ErrBadProposerDetails(ModuleName string, proposer sdk.AccAddress) error {
	return errors.Register(ModuleName, CodeInvalidProposerInput, fmt.Sprintf("Proposer is not valid, current proposer is %v", proposer.String()))
}

func ErrBadBlockDetails(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidBlockInput, "Wrong roothash for given start and end block numbers")
}

func ErrSetCheckpointBuffer(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrSetCheckpointBuffer, "Account Root Hash not added to Checkpoint Buffer")
}

func ErrAddCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrAddCheckpoint, "Err in adding checkpoint to header blocks")
}

func ErrBadAccountRootHash(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrAccountRootHash, "Wrong roothash for given dividend accounts")
}

func ErrBadAck(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidACK, "Ack Not Valid")
}

func ErrOldCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldCheckpoint, "Checkpoint already received for given start and end block")
}

func ErrDisContinuousCheckpoint(ModuleName string) error {
	return errors.Register(ModuleName, CodeDisContinuousCheckpoint, "Checkpoint not in continuity")
}

func ErrNoACK(ModuleName string, expiresAt uint64) error {
	return errors.Register(ModuleName, CodeNoACK, fmt.Sprintf("Checkpoint Already Exists In Buffer, ACK expected, expires at %s", strconv.FormatUint(expiresAt, 10)))
}

func ErrNoConn(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoConn, "Unable to connect to chain")
}

func ErrNoCheckpointFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoCheckpoint, "Checkpoint Not Found")
}

func ErrCheckpointAlreadyExists(ModuleName string) error {
	return errors.Register(ModuleName, CodeCheckpointAlreadyExists, "Checkpoint Already Exists")
}

func ErrNoCheckpointBufferFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoCheckpointBuffer, "Checkpoint buffer not found")
}

func ErrCheckpointBufferFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeCheckpointBuffer, "Checkpoint buffer found")
}

func ErrInvalidNoACK(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidNoACK, "Invalid No ACK -- Waiting for last checkpoint ACK")
}

func ErrInvalidNoACKProposer(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidNoAckProposer, "Invalid No ACK Proposer")
}

func ErrTooManyNoACK(ModuleName string) error {
	return errors.Register(ModuleName, CodeTooManyNoAck, "Too many no-acks")
}

func ErrBadTimeStamp(ModuleName string) error {
	return errors.Register(ModuleName, CodeBadTimeStamp, "Invalid time stamp. It must be in near past.")
}

// -----------Milestone Errors
func ErrNoMilestoneFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoMilestone, "Milestone Not Found")
}

func ErrMilestoneNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeMilestoneNotInContinuity, "Milestone not in continuity")
}

func ErrMilestoneInvalid(ModuleName string) error {
	return errors.Register(ModuleName, CodeMilestoneInvalid, "Milestone Msg Invalid")
}

func ErrOldMilestone(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldMilestone, "Milestone already exists")
}

func ErrInvalidMilestoneTimeout(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidMilestoneTimeout, "Invalid Milestone Timeout msg ")
}

func ErrTooManyMilestoneTimeout(ModuleName string) error {
	return errors.Register(ModuleName, CodeTooManyNoAck, "Too many milestone timeout msg")
}

func ErrInvalidMilestoneIndex(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoMilestone, "Invalid milestone index")
}

func ErrPrevMilestoneInVoting(ModuleName string) error {
	return errors.Register(ModuleName, CodePrevMilestoneInVoting, "Previous milestone still in voting phase")
}

// ----------- Staking Errors

func ErrOldValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldValidator, "Start Epoch behind Current Epoch")
}

func ErrNoValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoValidator, "Validator information not found")
}

func ErrNonce(ModuleName string) error {
	return errors.Register(ModuleName, CodeNonce, "Incorrect validator nonce")
}

func ErrValSignerPubKeyMismatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValPubkeyMismatch, "Signer Pubkey mismatch between event and msg")
}

func ErrValSignerMismatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSignerMismatch, "Signer Address doesnt match pubkey address")
}

func ErrValIsNotCurrentVal(ModuleName string) error {
	return errors.Register(ModuleName, CodeValidatorExitDeny, "Validator is not in validator set, exit not possible")
}

func ErrValUnbonded(ModuleName string) error {
	return errors.Register(ModuleName, CodeValAlreadyUnbonded, "Validator already unbonded , cannot exit")
}

func ErrSignerUpdateError(ModuleName string) error {
	return errors.Register(ModuleName, CodeSignerUpdateError, "Signer update error")
}

func ErrNoSignerChange(ModuleName string) error {
	return errors.Register(ModuleName, CodeNoSignerChangeError, "New signer same as old signer")
}

func ErrOldTx(ModuleName string) error {
	return errors.Register(ModuleName, CodeOldTx, "Old txhash not allowed")
}

func ErrValidatorAlreadySynced(ModuleName string) error {
	return errors.Register(ModuleName, CodeSignerSynced, "No signer update found, invalid message")
}

func ErrValidatorSave(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSave, "Cannot save validator")
}

func ErrValidatorNotDeactivated(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSave, "Validator Not Deactivated")
}

func ErrValidatorAlreadyJoined(ModuleName string) error {
	return errors.Register(ModuleName, CodeValAlreadyJoined, "Validator already joined")
}

// Bor Errors --------------------------------

func ErrInvalidBorChainID(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidBorChainID, "Invalid Bor chain id")
}

func ErrSpanNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeSpanNotContinuous, "Span not continuous")
}

func ErrInvalidSpanDuration(ModuleName string) error {
	return errors.Register(ModuleName, CodeInvalidSpanDuration, "wrong span duration")
}

func ErrSpanNotFound(ModuleName string) error {
	return errors.Register(ModuleName, CodeSpanNotFound, "Span not found")
}

func ErrUnableToFreezeValSet(ModuleName string) error {
	return errors.Register(ModuleName, CodeUnableToFreezeSet, "Unable to freeze validator set for next span")
}

func ErrValSetMisMatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSetMisMatch, "Validator set mismatch")
}

func ErrProducerMisMatch(ModuleName string) error {
	return errors.Register(ModuleName, CodeProducerMisMatch, "Producer set mismatch")
}

// TODO HV2 - These are not needed as we don't return any error from side tx
// //
// // Side-tx errors
// //

// // ErrorSideTx represents side-tx error
// func ErrorSideTx(ModuleName string, code uint32) (res abci.ResponseDeliverSideTx) {
// 	res.Code = uint32(code)
// 	res.Codespace = string(codespace)
// 	res.Result = voteTypes.Vote_VOTE_SKIP // skip side-tx vote in-case of error

// 	return
// }

// func ErrSideTxValidation(ModuleName string) error {
// 	return errors.Register(ModuleName, CodeSideTxValidationFailed, "External call majority validation failed. ")
// }

//
// Private methods
//

func CodeToDefaultMsg(code uint32) string {
	switch code {
	// case CodeInvalidBlockInput:
	// 	return "Invalid Block Input"
	case CodeInvalidMsg:
		return "Invalid Message"
	case CodeInvalidProposerInput:
		return "Proposer is not valid"
	case CodeInvalidBlockInput:
		return "Wrong roothash for given start and end block numbers"
	case CodeInvalidACK:
		return "Ack Not Valid"
	case CodeNoACK:
		return "Checkpoint Already Exists In Buffer, ACK expected"
	case CodeBadTimeStamp:
		return "Invalid time stamp. It must be in near past."
	case CodeInvalidNoACK:
		return "Invalid No ACK -- Waiting for last checkpoint ACK"
	case CodeTooManyNoAck:
		return "Too many no-acks"
	case CodeLowBal:
		return "Insufficient balance"
	case CodeNoCheckpoint:
		return "Checkpoint Not Found"
	case CodeOldCheckpoint:
		return "Checkpoint already received for given start and end block"
	case CodeDisContinuousCheckpoint:
		return "Checkpoint not in continuity"
	case CodeNoCheckpointBuffer:
		return "Checkpoint buffer Not Found"
	case CodeOldValidator:
		return "Start Epoch behind Current Epoch"
	case CodeNoValidator:
		return "Validator information not found"
	case CodeValSignerMismatch:
		return "Signer Address doesnt match pubkey address"
	case CodeValidatorExitDeny:
		return "Validator is not in validator set, exit not possible"
	case CodeValAlreadyUnbonded:
		return "Validator already unbonded , cannot exit"
	case CodeSignerSynced:
		return "No signer update found, invalid message"
	case CodeValSave:
		return "Cannot save validator"
	case CodeValAlreadyJoined:
		return "Validator already joined"
	case CodeSignerUpdateError:
		return "Signer update error"
	case CodeNoConn:
		return "Unable to connect to chain"
	case CodeWaitFrConfirmation:
		return "wait for confirmation time before sending transaction"
	case CodeValPubkeyMismatch:
		return "Signer Pubkey mismatch between event and msg"
	case CodeSpanNotContinuous:
		return "Span not continuous"
	case CodeUnableToFreezeSet:
		return "Unable to freeze validator set for next span"
	case CodeSpanNotFound:
		return "Span not found"
	case CodeValSetMisMatch:
		return "Validator set mismatch"
	case CodeProducerMisMatch:
		return "Producer set mismatch"
	case CodeInvalidBorChainID:
		return "Invalid Bor chain id"
	default:
		return "Default error"
	}
}

// Slashing errors
func ErrValidatorSigningInfoSave(ModuleName string) error {
	return errors.Register(ModuleName, CodeValSigningInfoSave, "Cannot save validator signing info")
}

func ErrUnjailValidator(ModuleName string) error {
	return errors.Register(ModuleName, CodeErrValUnjail, "Error while unJail validator")
}

func ErrSlashInfoDetails(ModuleName string) error {
	return errors.Register(ModuleName, CodeSlashInfoDetails, "Wrong slash info details")
}

func ErrTickNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeTickNotInContinuity, "Tick not in continuity")
}

func ErrTickAckNotInContinuity(ModuleName string) error {
	return errors.Register(ModuleName, CodeTickAckNotInContinuity, "Tick-ack not in continuity")
}
