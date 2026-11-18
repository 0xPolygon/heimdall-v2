package app

import (
	"bytes"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

const (
	maxSignedVESize = 6 * 1024

	// nonRpExtensionHashField is the field number of sidetxs.VoteExtension.non_rp_extension_hash.
	nonRpExtensionHashField protowire.Number = 5
)

func nonRpExtensionHash(height int64, extension []byte) []byte {
	if !helper.IsPurley(height) {
		return nil
	}
	return crypto.Keccak256(extension)
}

// validateVoteEnvelope uses only the signed payload and the vote's authenticated metadata.
// It is also safe after Commit, when checkpoint state and the SDK context have changed.
func validateVoteEnvelope(req *abci.RequestVerifyVoteExtension) (*sidetxs.VoteExtension, error) {
	if len(req.VoteExtension) < minVESize || len(req.VoteExtension) > maxSignedVESize {
		return nil, fmt.Errorf("vote extension size out of bounds: %d", len(req.VoteExtension))
	}
	if len(req.NonRpVoteExtension) < minNonRpVoteExtensionSizeAt(req.Height) || len(req.NonRpVoteExtension) > maxNonRpVoteExtensionSize {
		return nil, fmt.Errorf("non-rp extension size out of bounds: %d", len(req.NonRpVoteExtension))
	}
	if err := rejectVoteEnvelopeFields(req.VoteExtension, req.Height); err != nil {
		return nil, err
	}
	var extension sidetxs.VoteExtension
	if err := extension.Unmarshal(req.VoteExtension); err != nil {
		return nil, fmt.Errorf("decode vote extension: %w", err)
	}
	// Purley extensions omit the height and carry a block hash prefix; req.Height is authenticated by
	// the extension signature, and req.Hash is either the voted block or a prefix pinned by the commit.
	if err := validateVoteExtensionHeader(&extension, req.Height); err != nil {
		return nil, err
	}
	if !bytes.Equal(extension.BlockHash, voteExtensionBlockHash(req.Height, req.Hash)) {
		return nil, fmt.Errorf("vote extension block hash mismatch")
	}
	if !bytes.Equal(extension.NonRpExtensionHash, nonRpExtensionHash(req.Height, req.NonRpVoteExtension)) {
		return nil, fmt.Errorf("non-rp extension hash mismatch")
	}
	if _, err := validateSideTxResponses(extension.SideTxResponses); err != nil {
		return nil, err
	}
	return &extension, nil
}

// VerifyVoteExtension handles Purley envelopes before BaseApp constructs a context.
// In particular, the initial block's InitChain context is gone after Commit.
func (app *HeimdallApp) VerifyVoteExtension(req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	if helper.IsPurley(req.Height) {
		return verifyVoteEnvelope(req), nil
	}
	if req.Height == app.LastBlockHeight() {
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
	return app.BaseApp.VerifyVoteExtension(req)
}

func verifyVoteEnvelope(req *abci.RequestVerifyVoteExtension) *abci.ResponseVerifyVoteExtension {
	start := time.Now()
	defer metrics.RecordABCIHandlerDuration(metrics.VerifyVoteExtensionDuration, start)
	status := abci.ResponseVerifyVoteExtension_ACCEPT
	if _, err := validateVoteEnvelope(req); err != nil {
		status = abci.ResponseVerifyVoteExtension_REJECT
		metrics.RecordVoteExtensionRejected("invalid_envelope")
	}
	return &abci.ResponseVerifyVoteExtension{Status: status}
}

func rejectVoteEnvelopeFields(raw []byte, height int64) error {
	if err := rejectUnknownVoteExtFields(raw); err != nil {
		return err
	}
	if helper.IsPurley(height) {
		return nil
	}
	return forEachLenField(raw, func(num protowire.Number, _ []byte) error {
		if num == nonRpExtensionHashField {
			return fmt.Errorf("non-rp hash field before activation")
		}
		return nil
	})
}
