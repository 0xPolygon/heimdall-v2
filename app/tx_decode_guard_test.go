package app

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

const govProposalTypeURL = "/cosmos.gov.v1.MsgSubmitProposal"

// wrapAnyMessage builds bytes for a message with a single field-1 Any whose value is inner:
//
//	Msg{ messages: [ Any{ type_url, value: inner } ] }
//
// TxBody.messages and MsgSubmitProposal.messages are both field-1 repeated Any, so chaining
// wrapAnyMessage models the nested-Any self-chain.
func wrapAnyMessage(inner []byte, typeURL string) []byte {
	var any []byte
	any = protowire.AppendTag(any, 1, protowire.BytesType)
	any = protowire.AppendString(any, typeURL)
	any = protowire.AppendTag(any, 2, protowire.BytesType)
	any = protowire.AppendBytes(any, inner)

	var msg []byte
	msg = protowire.AppendTag(msg, 1, protowire.BytesType)
	msg = protowire.AppendBytes(msg, any)
	return msg
}

// nestedAnyBody chains depth wraps over an empty leaf, yielding a TxBody whose Any nesting is depth.
func nestedAnyBody(depth int) []byte {
	cur := []byte{}
	for range depth {
		cur = wrapAnyMessage(cur, govProposalTypeURL)
	}
	return cur
}

// txRawWithBody wraps body as TxRaw.body_bytes (field 1).
func txRawWithBody(body []byte) []byte {
	var tx []byte
	tx = protowire.AppendTag(tx, 1, protowire.BytesType)
	tx = protowire.AppendBytes(tx, body)
	return tx
}

func lenField(num protowire.Number, val []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, val)
	return b
}

func varintField(num protowire.Number, val uint64) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.VarintType)
	b = protowire.AppendVarint(b, val)
	return b
}

func TestCheckTxNestingDepthBoundary(t *testing.T) {
	tests := []struct {
		name    string
		depth   int
		wantErr bool
	}{
		{name: "flat body no nesting", depth: 0, wantErr: false},
		{name: "shallow nesting", depth: 3, wantErr: false},
		{name: "at max depth", depth: maxAnyNestingDepth, wantErr: false},
		{name: "one past max depth", depth: maxAnyNestingDepth + 1, wantErr: true},
		{name: "deeply nested", depth: 5000, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTxNesting(txRawWithBody(nestedAnyBody(tc.depth)))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScanAnyNestingBoundary(t *testing.T) {
	require.NoError(t, scanAnyNesting(nestedAnyBody(maxAnyNestingDepth), 0, 0))
	require.Error(t, scanAnyNesting(nestedAnyBody(maxAnyNestingDepth+1), 0, 0))
	// Starting anyDepth is honored: an otherwise-shallow body one level from the cap trips.
	require.Error(t, scanAnyNesting(nestedAnyBody(2), maxAnyNestingDepth, 0))
}

func TestCheckTxNestingScansAuthInfo(t *testing.T) {
	// A deep chain carried in auth_info_bytes (field 2) must also be rejected.
	tx := lenField(2, nestedAnyBody(maxAnyNestingDepth+1))
	require.Error(t, checkTxNesting(tx))
}

func TestCheckTxNestingIgnoresNonProtoAndEmpty(t *testing.T) {
	require.NoError(t, checkTxNesting(nil))
	require.NoError(t, checkTxNesting([]byte{0xff, 0xff, 0xff}))
}

func TestAnyValue(t *testing.T) {
	validAny := func() []byte {
		b := lenField(1, []byte(govProposalTypeURL))
		return append(b, lenField(2, []byte("payload"))...)
	}
	tests := []struct {
		name    string
		in      []byte
		wantOK  bool
		wantVal string
	}{
		{name: "valid any", in: validAny(), wantOK: true, wantVal: "payload"},
		// An extra field on the envelope must not disqualify the Any, else it hides a chain from the guard.
		{name: "extra field tolerated", in: append(validAny(), lenField(1024, []byte("x"))...), wantOK: true, wantVal: "payload"},
		{name: "type url without slash", in: lenField(1, []byte("cosmos.gov.v1.Msg")), wantOK: false},
		{name: "empty type url", in: lenField(1, []byte{}), wantOK: false},
		{name: "value only, no type url", in: lenField(2, []byte("payload")), wantOK: false},
		{name: "type url as non length-delimited field", in: varintField(1, 7), wantOK: false},
		{name: "malformed", in: []byte{0x0a, 0x05, 0x01}, wantOK: false},
		{name: "empty", in: []byte{}, wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := anyValue(tc.in)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, tc.wantVal, string(val))
			}
		})
	}
}

func TestIsProtoMessage(t *testing.T) {
	require.True(t, isProtoMessage(lenField(1, []byte("x"))))
	require.True(t, isProtoMessage(varintField(1, 7)))
	require.True(t, isProtoMessage(append(lenField(1, []byte("a")), varintField(2, 9)...)))
	require.False(t, isProtoMessage(nil))
	require.False(t, isProtoMessage([]byte{}))
	require.False(t, isProtoMessage([]byte{0xff}))                   // malformed tag
	require.False(t, isProtoMessage([]byte{0x0a, 0x05, 0x01, 0x02})) // truncated LEN value
}

func TestIsTypeURL(t *testing.T) {
	require.True(t, isTypeURL([]byte("/cosmos.gov.v1.MsgSubmitProposal")))
	require.False(t, isTypeURL([]byte("cosmos.gov.v1.MsgSubmitProposal")))
	require.False(t, isTypeURL([]byte{}))
	require.False(t, isTypeURL(nil))
}

func TestNextLenField(t *testing.T) {
	t.Run("length-delimited field", func(t *testing.T) {
		in := lenField(1, []byte("abc"))
		num, val, n, ok := nextLenField(in)
		require.True(t, ok)
		require.Equal(t, protowire.Number(1), num)
		require.Equal(t, "abc", string(val))
		require.Equal(t, len(in), n)
	})
	t.Run("non length-delimited field returns nil value", func(t *testing.T) {
		in := varintField(1, 7)
		num, val, n, ok := nextLenField(in)
		require.True(t, ok)
		require.Equal(t, protowire.Number(1), num)
		require.Nil(t, val)
		require.Equal(t, len(in), n)
	})
	t.Run("truncated tag", func(t *testing.T) {
		_, _, _, ok := nextLenField([]byte{0xff})
		require.False(t, ok)
	})
	t.Run("truncated length-delimited value", func(t *testing.T) {
		// field 1, LEN, declared length 5, but only 2 bytes follow.
		_, _, _, ok := nextLenField([]byte{0x0a, 0x05, 0x01, 0x02})
		require.False(t, ok)
	})
	t.Run("truncated varint value", func(t *testing.T) {
		// field 1, varint, continuation bit set but no next byte.
		_, _, _, ok := nextLenField([]byte{0x08, 0xff})
		require.False(t, ok)
	})
}

func TestForEachLenField(t *testing.T) {
	in := append(lenField(1, []byte("a")), varintField(2, 9)...)
	in = append(in, lenField(3, []byte("c"))...)

	var seen []protowire.Number
	require.NoError(t, forEachLenField(in, func(num protowire.Number, _ []byte) error {
		seen = append(seen, num)
		return nil
	}))
	require.Equal(t, []protowire.Number{1, 3}, seen) // varint field 2 skipped

	// Callback error propagates and stops iteration.
	stop := errors.New("stop")
	calls := 0
	err := forEachLenField(in, func(protowire.Number, []byte) error {
		calls++
		return stop
	})
	require.ErrorIs(t, err, stop)
	require.Equal(t, 1, calls)

	// Malformed input stops cleanly without error and without a callback.
	calls = 0
	require.NoError(t, forEachLenField([]byte{0xff}, func(protowire.Number, []byte) error {
		calls++
		return nil
	}))
	require.Equal(t, 0, calls)
}

func TestBoundedNestingTxDecoder(t *testing.T) {
	sentinel := errors.New("inner reached")
	inner := sdk.TxDecoder(func([]byte) (sdk.Tx, error) { return nil, sentinel })
	dec := boundedNestingTxDecoder(inner)

	// A deeply nested tx is rejected by the guard, before the inner decoder runs.
	_, err := dec(txRawWithBody(nestedAnyBody(maxAnyNestingDepth + 1)))
	require.Error(t, err)
	require.NotErrorIs(t, err, sentinel)

	// A shallow tx passes the guard and reaches the inner decoder.
	_, err = dec(txRawWithBody(nestedAnyBody(2)))
	require.ErrorIs(t, err, sentinel)
}

// wrapAnyMessageExtra is wrapAnyMessage with a throwaway extra field on the Any envelope.
func wrapAnyMessageExtra(inner []byte) []byte {
	var anyB []byte
	anyB = protowire.AppendTag(anyB, 1, protowire.BytesType)
	anyB = protowire.AppendString(anyB, govProposalTypeURL)
	anyB = protowire.AppendTag(anyB, 2, protowire.BytesType)
	anyB = protowire.AppendBytes(anyB, inner)
	anyB = protowire.AppendTag(anyB, 1024, protowire.BytesType) // non-critical extra field
	anyB = protowire.AppendBytes(anyB, []byte("x"))

	var msg []byte
	msg = protowire.AppendTag(msg, 1, protowire.BytesType)
	msg = protowire.AppendBytes(msg, anyB)
	return msg
}

// An extra field on every Any envelope must not let the chain slip past the guard.
func TestCheckTxNesting_ExtraFieldAnyChainNotBypassed(t *testing.T) {
	body := func(d int) []byte {
		cur := []byte{}
		for range d {
			cur = wrapAnyMessageExtra(cur)
		}
		return cur
	}
	require.NoError(t, checkTxNesting(txRawWithBody(body(maxAnyNestingDepth))))
	require.Error(t, checkTxNesting(txRawWithBody(body(maxAnyNestingDepth+1))))
}

// A duplicated body_bytes must not let a shallow decoy hide a deep chain: the inner decoder
// applies last-field-wins, so every occurrence has to be scanned.
func TestCheckTxNesting_DuplicateBodyBytesNotBypassed(t *testing.T) {
	shallow := nestedAnyBody(1)
	deep := nestedAnyBody(maxAnyNestingDepth + 1)
	require.Error(t, checkTxNesting(append(lenField(1, shallow), lenField(1, deep)...)))
	require.Error(t, checkTxNesting(append(lenField(1, deep), lenField(1, shallow)...)))
}

// wrapAnyBehindNonAny nests the next level as: Msg{ Any{ value: NonAny{ field7: inner } } },
// so each Any is reached through a non-Any wrapper.
func wrapAnyBehindNonAny(inner []byte) []byte {
	var wrap []byte
	wrap = protowire.AppendTag(wrap, 7, protowire.BytesType)
	wrap = protowire.AppendBytes(wrap, inner)

	var anyB []byte
	anyB = protowire.AppendTag(anyB, 1, protowire.BytesType)
	anyB = protowire.AppendString(anyB, govProposalTypeURL)
	anyB = protowire.AppendTag(anyB, 2, protowire.BytesType)
	anyB = protowire.AppendBytes(anyB, wrap)

	var msg []byte
	msg = protowire.AppendTag(msg, 1, protowire.BytesType)
	msg = protowire.AppendBytes(msg, anyB)
	return msg
}

// An Any chain diluted with non-Any wrapper layers must still be counted.
func TestCheckTxNesting_AnyBehindNonAnyStillCounted(t *testing.T) {
	body := func(d int) []byte {
		cur := []byte{}
		for range d {
			cur = wrapAnyBehindNonAny(cur)
		}
		return cur
	}
	require.NoError(t, checkTxNesting(txRawWithBody(body(maxAnyNestingDepth))))
	require.Error(t, checkTxNesting(txRawWithBody(body(maxAnyNestingDepth+1))))
}
