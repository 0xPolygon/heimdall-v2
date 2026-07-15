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

// boundedNestingTxDecoder wraps a TxDecoder with a cheap Any-nesting pre-scan.
func boundedNestingTxDecoder(inner sdk.TxDecoder) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		if err := checkTxNesting(txBytes); err != nil {
			return nil, err
		}
		return inner(txBytes)
	}
}

// checkTxNesting walks the transaction's message trees following Any chains and rejects
// nesting beyond maxAnyNestingDepth. A malformed prefix is left for the real decoder to
// report; this scan only ever adds a rejection, never accepts what the decoder rejects.
func checkTxNesting(txBytes []byte) error {
	// TxRaw: body_bytes = field 1, auth_info_bytes = field 2, both length-delimited.
	if body, ok := protoField(txBytes, 1); ok {
		if err := scanAnyNesting(body, 0); err != nil {
			return err
		}
	}
	if authInfo, ok := protoField(txBytes, 2); ok {
		if err := scanAnyNesting(authInfo, 0); err != nil {
			return err
		}
	}
	return nil
}

// scanAnyNesting descends only through Any values, incrementing depth per unwrap. It
// never copies: protowire returns sub-slices of the input, so the whole scan is bounded
// by maxAnyNestingDepth times the message size.
func scanAnyNesting(msg []byte, depth int) error {
	if depth > maxAnyNestingDepth {
		return sdkerrors.ErrTxDecode.Wrapf("message Any nesting exceeds max depth %d", maxAnyNestingDepth)
	}
	return forEachLenField(msg, func(_ protowire.Number, v []byte) error {
		if inner, ok := anyValue(v); ok {
			return scanAnyNesting(inner, depth+1)
		}
		return nil
	})
}

// protoField returns the value of the first length-delimited field numbered want.
func protoField(b []byte, want protowire.Number) ([]byte, bool) {
	var found []byte
	got := false
	_ = forEachLenField(b, func(num protowire.Number, v []byte) error {
		if num == want && !got {
			found, got = v, true
		}
		return nil
	})
	return found, got
}

// anyValue reports whether v is a google.protobuf.Any and returns its value bytes. An Any
// carries only field 1 (type_url, a "/"-prefixed string) and field 2 (value, bytes), both
// length-delimited; any other field number or a non length-delimited field disqualifies
// the match, so plain scalar fields are not mistaken for Any and descended into.
func anyValue(v []byte) ([]byte, bool) {
	var typeURL, value []byte
	rest := v
	for len(rest) > 0 {
		num, val, n, ok := nextLenField(rest)
		if !ok || val == nil {
			return nil, false
		}
		rest = rest[n:]
		switch num {
		case 1:
			typeURL = val
		case 2:
			value = val
		default:
			return nil, false
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
