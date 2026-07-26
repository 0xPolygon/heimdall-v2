package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/protobuf/encoding/protowire"
)

// maxAnyNestingDepth bounds how many nested google.protobuf.Any unwraps a transaction
// may carry. It sits above the SDK's own Any-unpack recursion limit (10), so it never
// rejects a transaction the decoder would otherwise accept — it only lets us discard a
// pathologically nested message before the decoder's unknown-field pre-pass walks and
// copies every level. Any-typed message fields (e.g. gov MsgSubmitProposal.messages)
// can self-chain, and that pre-pass copies each level's Any value, so an unbounded chain
// costs work proportional to depth times size; this cap keeps a rejected transaction
// cheap to reject on every decode path (CheckTx, PrepareProposal, ProcessProposal).
const maxAnyNestingDepth = 16

// maxScanDepth bounds the guard's own recursion over the message tree so the scan cost
// stays proportional to transaction size regardless of shape. It is comfortably above
// maxAnyNestingDepth: an Any chain long enough to matter trips the Any cap first, while
// deep non-Any nesting (which the decoder does not amplify) is simply not descended past
// this ceiling.
const maxScanDepth = 64

// boundedNestingTxDecoder wraps a TxDecoder with a cheap Any-nesting pre-scan.
func boundedNestingTxDecoder(inner sdk.TxDecoder) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		if err := checkTxNesting(txBytes); err != nil {
			return nil, err
		}
		return inner(txBytes)
	}
}

// checkTxNesting scans every TxRaw.body_bytes (field 1) and auth_info_bytes (field 2) for
// excessive Any nesting. It walks all occurrences, not just the first: the inner decoder
// applies last-field-wins to these singular fields, so a duplicated field must not let a
// shallow decoy hide a deep chain behind it.
func checkTxNesting(txBytes []byte) error {
	return forEachLenField(txBytes, func(num protowire.Number, v []byte) error {
		if num == 1 || num == 2 {
			return scanAnyNesting(v, 0, 0)
		}
		return nil
	})
}

// scanAnyNesting walks a message tree, counting Any unwraps in anyDepth and total recursion
// in scanDepth. It descends into every sub-message, so an Any reached through non-Any
// wrappers is still counted; anyDepth (not scanDepth) is what rejects, so deep non-Any
// nesting is never mistaken for a nested-Any chain. protowire returns sub-slices, so the
// scan never copies.
func scanAnyNesting(msg []byte, anyDepth, scanDepth int) error {
	if anyDepth > maxAnyNestingDepth {
		return sdkerrors.ErrTxDecode.Wrapf("message Any nesting exceeds max depth %d", maxAnyNestingDepth)
	}
	if scanDepth >= maxScanDepth {
		return nil
	}
	return forEachLenField(msg, func(_ protowire.Number, v []byte) error {
		if inner, ok := anyValue(v); ok {
			return scanAnyNesting(inner, anyDepth+1, scanDepth+1)
		}
		if isProtoMessage(v) {
			return scanAnyNesting(v, anyDepth, scanDepth+1)
		}
		return nil
	})
}

// anyValue reports whether v is shaped like a google.protobuf.Any and returns its value
// bytes. An Any carries a "/"-prefixed type_url in field 1 and a value in field 2; any
// other field is ignored — the decoder tolerates non-critical extra fields on the envelope,
// so the guard must too, or a throwaway field would hide a chain from it.
func anyValue(v []byte) ([]byte, bool) {
	var typeURL, value []byte
	rest := v
	for len(rest) > 0 {
		num, val, n, ok := nextLenField(rest)
		if !ok {
			return nil, false
		}
		rest = rest[n:]
		switch num {
		case 1:
			typeURL = val
		case 2:
			value = val
		}
	}
	if !isTypeURL(typeURL) {
		return nil, false
	}
	return value, true
}

// isTypeURL reports whether b is a non-empty, "/"-prefixed proto type URL.
func isTypeURL(b []byte) bool {
	return len(b) > 0 && b[0] == '/'
}

// isProtoMessage reports whether b parses cleanly as a sequence of protobuf fields, so the
// guard descends into genuine sub-messages and not into scalar string/bytes payloads.
func isProtoMessage(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	rest := b
	for len(rest) > 0 {
		_, _, n, ok := nextLenField(rest)
		if !ok {
			return false
		}
		rest = rest[n:]
	}
	return true
}

// forEachLenField invokes fn for every length-delimited field (number, value) in b,
// skipping other wire types. It stops without error on the first malformed byte, leaving
// canonical decode errors to the real decoder.
func forEachLenField(b []byte, fn func(protowire.Number, []byte) error) error {
	rest := b
	for len(rest) > 0 {
		num, val, n, ok := nextLenField(rest)
		if !ok {
			return nil
		}
		rest = rest[n:]
		if val == nil {
			continue
		}
		if err := fn(num, val); err != nil {
			return err
		}
	}
	return nil
}

// nextLenField consumes one proto field from b, returning its number, its value (nil for
// non length-delimited fields), the bytes consumed, and ok=false on malformed input.
func nextLenField(b []byte) (protowire.Number, []byte, int, bool) {
	num, typ, n := protowire.ConsumeTag(b)
	if n < 0 {
		return 0, nil, 0, false
	}
	var val []byte
	var m int
	if typ == protowire.BytesType {
		val, m = protowire.ConsumeBytes(b[n:])
	} else {
		m = protowire.ConsumeFieldValue(num, typ, b[n:])
	}
	if m < 0 {
		return 0, nil, 0, false
	}
	return num, val, n + m, true
}
