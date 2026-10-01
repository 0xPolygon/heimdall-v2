package app

import (
	"errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// maxTxNestingRecursion bounds the same recursive traversal the real tx decoder's own
// unknown-field pre-pass (unknownproto.RejectUnknownFieldsStrict/RejectUnknownFields, called
// from DefaultTxDecoder) performs -- resolving each google.protobuf.Any via the app's
// InterfaceRegistry and recursing into every message-typed field the resolved type's real
// descriptor declares, Any or not. That pass's own default recursion limit is 10,000: a
// transaction whose Any resolves to a self-referential registered type (either directly, or
// via an ordinary embedded message field one or more levels down -- e.g. a TxBody.messages Any
// wrapping a registered /cosmos.tx.v1beta1.Tx, whose own Body field is an ordinary embedded
// TxBody that can wrap another such Any) can cost real work proportional to that depth before
// the real decoder rejects it. Capping the SAME traversal at this much lower bound, ahead of
// the real (unbounded-by-default) decode, rejects such a transaction at this bound's cost
// instead of paying up to 10,000 levels' worth of resolution and recursion first. 16 matches
// the depth this guard originally shipped with (when it only counted Any resolutions): every
// legitimate self-chain reachable in this app today (gov proposal messages, multisig member
// keys) costs exactly one unit of this same recursion count per level, so the historically
// reviewed and accepted headroom is unchanged; a chain that must additionally hop through an
// ordinary embedded message field per level (like Tx.Body) is bounded even tighter in real
// conceptual levels, which only strengthens the DoS bound.
const maxTxNestingRecursion = 16

// hasOverNestedTx reports whether, at/after the Kyoto height, any of txs is over-nested.
// Gated so the accept/reject decision is uniform across validators from each fork boundary;
// before Kyoto it is a no-op and decode behavior is unchanged.
//
// Between Kyoto and Lugano, txs are scanned by checkTxNestingLegacy -- the original
// byte-level heuristic, kept alive here (not just left in git history) so upgrading past
// Kyoto never leaves a network with zero guard coverage while waiting for Lugano to
// activate. At/after Lugano, txs are scanned by the real descriptor-driven checkTxNesting,
// which closes the legacy heuristic's ordinary-message-descent gap (see checkTxNestingLegacy's
// doc comment). Splitting on Lugano rather than swapping checkTxNesting's algorithm in place
// under the already-fully-activated Kyoto gate is deliberate: Kyoto is long past its
// activation height on every network, so a same-fork algorithm swap would take effect the
// instant each validator's binary upgrades, with no coordinated cutover -- exactly the
// mixed-version window a fork gate exists to avoid for a decision that must be uniform
// across validators.
func hasOverNestedTx(registry codectypes.InterfaceRegistry, height int64, txs [][]byte) bool {
	if !helper.IsKyoto(height) {
		return false
	}
	lugano := helper.IsLugano(height)
	for _, txBytes := range txs {
		var overNested bool
		if lugano {
			overNested = checkTxNesting(registry, txBytes) != nil
		} else {
			overNested = checkTxNestingLegacy(txBytes) != nil
		}
		if overNested {
			return true
		}
	}
	return false
}

// checkTxNesting runs the real decoder's own field-descriptor-driven traversal (via
// unknownproto), capped at maxTxNestingRecursion, over TxRaw's body_bytes and auth_info_bytes
// -- the same two fields, and the same allowUnknownNonCriticals choice per field, that
// DefaultTxDecoder itself passes to unknownproto ahead of its own (unbounded-by-default)
// pass. Any error other than hitting that cap (a malformed tx, a genuinely unknown field, an
// unresolvable type URL) is left for the real decode path to reject on its own terms with its
// own error, unaffected by this guard -- checkTxNesting only ever reports the nesting bound
// itself, never substitutes for or changes any other decode-time rejection.
func checkTxNesting(registry codectypes.InterfaceRegistry, txBytes []byte) error {
	var raw sdktx.TxRaw
	if err := raw.Unmarshal(txBytes); err != nil {
		return nil //nolint:nilerr // malformed TxRaw is left for the real decode path to reject with its own error, per this func's doc comment
	}

	var body sdktx.TxBody
	if err := rejectOverNested(raw.BodyBytes, &body, true, registry); err != nil {
		return err
	}

	var authInfo sdktx.AuthInfo
	return rejectOverNested(raw.AuthInfoBytes, &authInfo, false, registry)
}

// rejectOverNested wraps unknownproto.RejectUnknownFieldsWithRecursionLimit, translating only
// its ErrRecursionLimitReached into this guard's own rejection error. Every other error
// (malformed input, genuinely unknown fields, unresolvable Any type URLs) is swallowed here so
// the real decode path -- which runs the identical check with its own, much larger limit --
// remains the sole source of truth for every rejection reason besides nesting depth.
func rejectOverNested(bz []byte, msg proto.Message, allowUnknownNonCriticals bool, registry codectypes.InterfaceRegistry) error {
	_, err := unknownproto.RejectUnknownFieldsWithRecursionLimit(bz, msg, allowUnknownNonCriticals, registry, maxTxNestingRecursion)
	if err != nil && errors.Is(err, unknownproto.ErrRecursionLimitReached) {
		return sdkerrors.ErrTxDecode.Wrap("transaction exceeds the message nesting bound")
	}
	return nil
}

// maxAnyNestingDepth bounds checkTxNestingLegacy's byte-level Any-unwrap count. This is the
// guard's original implementation, kept only for the Kyoto-to-Lugano window -- see
// hasOverNestedTx's doc comment for why it isn't simply replaced in place.
const maxAnyNestingDepth = 16

// checkTxNestingLegacy is the guard's original implementation: a byte-level heuristic that
// walks TxRaw.body_bytes (field 1) and auth_info_bytes (field 2) for occurrences of
// google.protobuf.Any's wire shape (see anyValue) and counts only direct Any-to-Any unwraps.
//
// It has a real, known gap that checkTxNesting's descriptor-driven traversal closes: nesting
// that hops through an ORDINARY (non-Any) embedded message field between Any unwraps --
// e.g. a TxBody.messages Any resolving to the real, registered /cosmos.tx.v1beta1.Tx, whose
// own Body field is a plain embedded TxBody, which can itself wrap another such Any -- is
// invisible to this heuristic. anyValue only recognizes a field as "descend further" when its
// raw bytes happen to parse as {field 1: "/"-prefixed bytes, field 2: bytes}; an ordinary
// embedded message's raw bytes essentially never satisfy that shape (its own field 1 is
// whatever the real schema says, not a type-URL string), so forEachLenField stops recursing
// there and the walk never reaches the Any nested one ordinary hop further in -- regardless of
// how many such hops the payload actually contains. A transaction built entirely from that
// shape costs the real decoder work proportional to its true depth while this heuristic
// reports it as shallow. checkTxNesting does not have this gap because it resolves each Any
// through the app's real InterfaceRegistry and recurses into every message-typed field the
// resolved type's real descriptor declares, ordinary or Any.
func checkTxNestingLegacy(txBytes []byte) error {
	return forEachLenField(txBytes, func(num protowire.Number, v []byte) error {
		if num == 1 || num == 2 {
			return scanAnyNesting(v, 0)
		}
		return nil
	})
}

// scanAnyNesting follows Any chains, incrementing depth per unwrap, and rejects nesting
// beyond maxAnyNestingDepth. It descends only into values matching the Any envelope shape
// (anyValue), so recursion depth equals the Any-unwrap count and a scalar field is walked
// only if its bytes happen to mimic that shape. protowire returns sub-slices, so the scan
// never copies.
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

// anyValue reports whether v is shaped like a google.protobuf.Any and returns its value
// bytes. An Any carries a "/"-prefixed type_url in field 1 and a value in field 2; any
// other field is ignored -- the decoder tolerates non-critical extra fields on the envelope,
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
