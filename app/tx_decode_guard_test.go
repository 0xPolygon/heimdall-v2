package app

import (
	"testing"

	"cosmossdk.io/x/tx/signing"
	codectypes "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/std"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	sdktypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/0xPolygon/heimdall-v2/helper"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

const govProposalTypeURL = "/cosmos.gov.v1.MsgSubmitProposal"
const legacyAminoPubKeyTypeURL = "/cosmos.crypto.multisig.LegacyAminoPubKey"
const txTypeURL = "/cosmos.tx.v1beta1.Tx"

// newTestInterfaceRegistry builds the same InterfaceRegistry NewHeimdallApp does (std types --
// Tx, accounts, gov v1/v1beta1, multisig -- plus this app's own clerk module), so tests resolve
// google.protobuf.Any exactly as the real decoder and this guard's real traversal would.
func newTestInterfaceRegistry(t *testing.T) sdktypes.InterfaceRegistry {
	registry, err := sdktypes.NewInterfaceRegistryWithOptions(sdktypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          codectypes.HexCodec{},
			ValidatorAddressCodec: codectypes.HexCodec{},
		},
	})
	require.NoError(t, err)
	std.RegisterInterfaces(registry)
	govv1.RegisterInterfaces(registry)
	govv1beta1.RegisterInterfaces(registry)
	clerktypes.RegisterInterfaces(registry)
	return registry
}

// lenField encodes a length-delimited (bytes/message) field.
func lenField(num protowire.Number, val []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, val)
	return b
}

// anyEnvelope builds a raw google.protobuf.Any encoding: field 1 = type_url, field 2 = value.
func anyEnvelope(typeURL string, value []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, typeURL)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, value)
	return b
}

// txRawWithBody wraps body as TxRaw.body_bytes (field 1).
func txRawWithBody(body []byte) []byte {
	return lenField(1, body)
}

// nestedAnyBodyField chains depth self-references of typeURL through the single field its
// real descriptor declares as a further Any (field, e.g. gov MsgSubmitProposal.messages = 1,
// multisig LegacyAminoPubKey.public_keys = 2), yielding a TxBody-shaped byte string (since
// TxBody.messages is also field 1, repeated Any) whose real-decoder resolution cost is exactly
// depth Any-resolutions deep.
func nestedAnyBodyField(depth int, typeURL string, field protowire.Number) []byte {
	cur := []byte{}
	for range depth {
		cur = lenField(field, anyEnvelope(typeURL, cur))
	}
	return cur
}

// nestedAnyBody chains depth self-references of gov's MsgSubmitProposal (field 1, matching
// both its own messages field and TxBody.messages), yielding a TxBody whose Any-resolution
// depth is exactly depth.
func nestedAnyBody(depth int) []byte {
	return nestedAnyBodyField(depth, govProposalTypeURL, 1)
}

// nestedTxInAny chains depth self-references through the real, registered
// /cosmos.tx.v1beta1.Tx: each level is TxBody{messages: [Any{Tx, value: Tx{body: <next
// level's TxBody>}}]} -- Tx.Body is an ORDINARY embedded TxBody field (not itself an Any), so
// the real decoder's traversal hops through it via ordinary (non-Any) descent before reaching
// the next level's Any again. This is exactly the shape a false economy in the guard's design
// would miss: closing over-nesting through direct Any self-chains alone is not enough, because
// the real decoder also recurses through ordinary message-typed fields at any depth, not just
// before the first Any.
func nestedTxInAny(depth int) []byte {
	cur := []byte{} // innermost TxBody: no messages
	for range depth {
		txMsg := lenField(1, cur)          // Tx{body: cur} (field 1, ordinary TxBody field)
		anyTx := anyEnvelope(txTypeURL, txMsg)
		cur = lenField(1, anyTx) // TxBody{messages: [anyTx]} (field 1, repeated Any)
	}
	return cur
}

func TestHasOverNestedTx(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	origKyoto := helper.GetKyotoHeight()
	origLugano := helper.GetLuganoHeight()
	t.Cleanup(func() {
		helper.SetKyotoHeight(origKyoto)
		helper.SetLuganoHeight(origLugano)
	})

	// A direct repeated-Any self-chain: caught by BOTH checkTxNestingLegacy (Kyoto..Lugano)
	// and checkTxNesting (Lugano+), since it's exactly the shape the legacy heuristic was
	// built to detect.
	bomb := txRawWithBody(nestedAnyBody(maxTxNestingRecursion * 4))
	shallow := txRawWithBody(nestedAnyBody(1))

	tests := []struct {
		name         string
		kyotoHeight  int64
		luganoHeight int64
		height       int64
		txs          [][]byte
		want         bool
	}{
		{name: "kyoto disabled ignores bomb", kyotoHeight: 0, luganoHeight: 0, height: 1000, txs: [][]byte{bomb}, want: false},
		{name: "before kyoto ignores bomb", kyotoHeight: 100, luganoHeight: 200, height: 99, txs: [][]byte{bomb}, want: false},
		{name: "at kyoto, before lugano: legacy catches bomb", kyotoHeight: 100, luganoHeight: 200, height: 100, txs: [][]byte{bomb}, want: true},
		{name: "between kyoto and lugano: legacy catches bomb", kyotoHeight: 100, luganoHeight: 200, height: 150, txs: [][]byte{shallow, bomb}, want: true},
		{name: "between kyoto and lugano: legacy passes shallow", kyotoHeight: 100, luganoHeight: 200, height: 150, txs: [][]byte{shallow}, want: false},
		{name: "at lugano: real traversal catches bomb", kyotoHeight: 100, luganoHeight: 200, height: 200, txs: [][]byte{bomb}, want: true},
		{name: "after lugano: real traversal passes shallow", kyotoHeight: 100, luganoHeight: 200, height: 201, txs: [][]byte{shallow}, want: false},
		{name: "no txs", kyotoHeight: 100, luganoHeight: 200, height: 201, txs: nil, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper.SetKyotoHeight(tc.kyotoHeight)
			helper.SetLuganoHeight(tc.luganoHeight)
			require.Equal(t, tc.want, hasOverNestedTx(registry, tc.height, tc.txs))
		})
	}
}

// TestHasOverNestedTx_LuganoClosesLegacyGap is the reason hasOverNestedTx switches
// algorithms at Lugano instead of just improving checkTxNesting under the (already fully
// activated) Kyoto gate in place: checkTxNestingLegacy cannot see nesting that hops through
// an ordinary (non-Any) embedded message field between Any unwraps (see
// checkTxNestingLegacy's doc comment), so a deep nestedTxInAny chain sails through it
// completely undetected -- confirmed here at a depth (500) far beyond both
// maxAnyNestingDepth and maxTxNestingRecursion. At/after Lugano, the same payload is caught
// by the real descriptor-driven checkTxNesting.
func TestHasOverNestedTx_LuganoClosesLegacyGap(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	origKyoto := helper.GetKyotoHeight()
	origLugano := helper.GetLuganoHeight()
	t.Cleanup(func() {
		helper.SetKyotoHeight(origKyoto)
		helper.SetLuganoHeight(origLugano)
	})
	helper.SetKyotoHeight(100)
	helper.SetLuganoHeight(200)

	deepTxInAny := txRawWithBody(nestedTxInAny(500))

	require.False(t, hasOverNestedTx(registry, 150, [][]byte{deepTxInAny}),
		"legacy heuristic (Kyoto..Lugano) does not see nesting through an ordinary embedded field -- this is the known, accepted gap during that window")
	require.True(t, hasOverNestedTx(registry, 200, [][]byte{deepTxInAny}),
		"real descriptor-driven traversal (Lugano+) must catch what the legacy heuristic misses")
}

// TestCheckTxNestingDepthBoundary pins the boundary for a pure gov self-chain: gov
// MsgSubmitProposal self-chains directly via its own Any-typed messages field (no ordinary
// hop in between). unknownproto's own recursion accounting costs one extra unit beyond the
// literal Any-resolution count for reasons internal to that package (each Any field is
// typechecked as a bare google.protobuf.Any before being resolved and recursed into again),
// so the boundary is pinned empirically here rather than re-derived from that internal detail
// -- maxTxNestingRecursion-1 is the deepest gov self-chain this guard accepts.
func TestCheckTxNestingDepthBoundary(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	tests := []struct {
		name    string
		depth   int
		wantErr bool
	}{
		{name: "flat body no nesting", depth: 0, wantErr: false},
		{name: "shallow nesting", depth: 3, wantErr: false},
		{name: "at max depth", depth: maxTxNestingRecursion - 1, wantErr: false},
		{name: "one past max depth", depth: maxTxNestingRecursion, wantErr: true},
		{name: "deeply nested", depth: 500, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTxNesting(registry, txRawWithBody(nestedAnyBody(tc.depth)))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckTxNestingLegacyDepthBoundary pins checkTxNestingLegacy's own depth boundary for a
// direct repeated-Any self-chain -- the one shape it actually detects. Unlike
// TestCheckTxNestingDepthBoundary's real-decoder-derived off-by-one, this heuristic counts
// only literal Any-unwraps, so the boundary is exactly maxAnyNestingDepth.
func TestCheckTxNestingLegacyDepthBoundary(t *testing.T) {
	tests := []struct {
		name    string
		depth   int
		wantErr bool
	}{
		{name: "flat body no nesting", depth: 0, wantErr: false},
		{name: "at max depth", depth: maxAnyNestingDepth, wantErr: false},
		{name: "one past max depth", depth: maxAnyNestingDepth + 1, wantErr: true},
		{name: "deeply nested", depth: 500, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTxNestingLegacy(txRawWithBody(nestedAnyBody(tc.depth)))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckTxNestingLegacy_TxInAnyGap documents (rather than merely reproduces via a probe)
// the known, accepted gap in the Kyoto..Lugano window: nesting through an ordinary embedded
// message field between Any unwraps is invisible to this heuristic at any depth, because
// anyValue only recurses when a field's raw bytes happen to parse as the Any wire shape, and
// an ordinary message's own fields essentially never do. This is exactly the class
// checkTxNesting's descriptor-driven traversal was written to close -- see
// TestCheckTxNesting_TxInAnyNotUnderCounted immediately below for the same payload shape
// caught by the real traversal.
func TestCheckTxNestingLegacy_TxInAnyGap(t *testing.T) {
	require.NoError(t, checkTxNestingLegacy(txRawWithBody(nestedTxInAny(500))),
		"known gap: checkTxNestingLegacy cannot see nesting through an ordinary embedded message field")
}

// TestCheckTxNesting_TxInAnyNotUnderCounted is the regression test for the ordinary-descent
// bypass: /cosmos.tx.v1beta1.Tx is a real, registered type resolvable from any Any slot typed
// for an interface it satisfies, and its own Body field is an ordinary (non-Any) embedded
// TxBody that can wrap another Any right back to another Tx. Depths are chosen well clear of
// the exact boundary (rather than pinned to it) since each level here costs the real decoder
// two recursion units (one Any resolution into Tx, one ordinary descent into its Body), not
// one -- unlike gov/multisig's direct self-chains -- so getting the exact boundary depth right
// isn't the point; closing the under-counting bug class is.
func TestCheckTxNesting_TxInAnyNotUnderCounted(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	require.NoError(t, checkTxNesting(registry, txRawWithBody(nestedTxInAny(2))))
	require.Error(t, checkTxNesting(registry, txRawWithBody(nestedTxInAny(maxTxNestingRecursion))))
}

// TestCheckTxNesting_MalformedTxRawIgnored proves a TxRaw that fails to unmarshal is left
// entirely to the real decode path, which rejects it on its own terms -- not merely that
// checkTxNesting happens to return nil for it. Trailing garbage after an otherwise well-formed,
// deeply nested body_bytes field still populates raw.BodyBytes before Unmarshal's own error
// surfaces (gogoproto's generated Unmarshal sets fields as it decodes them, not atomically at
// the end), so this is only a genuine test of the malformed-input guard, not a case that
// happens to be moot because nothing got parsed.
func TestCheckTxNesting_MalformedTxRawIgnored(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	trailingGarbageAfterDeepBody := append(txRawWithBody(nestedAnyBody(maxTxNestingRecursion+50)), 0xff, 0xff)
	require.NoError(t, checkTxNesting(registry, trailingGarbageAfterDeepBody))
}

// nonCriticalUnknownField builds a length-delimited field whose number has bit 10 set (field
// 1024), which unknownproto's own bit11NonCritical convention treats as non-critical --
// tolerated when allowUnknownNonCriticals is true, rejected (like any other unknown field)
// when it's false.
func nonCriticalUnknownField() []byte {
	return lenField(1024, []byte{0x01})
}

// TestCheckTxNesting_BodyAllowsNonCriticalFields pins checkTxNesting's own choice of
// allowUnknownNonCriticals for TxBody (true, matching DefaultTxDecoder's identical choice for
// body_bytes): a non-critical unknown field placed before the real, deep chain must be
// tolerated and skipped over, not treated as a decode failure that would mask the chain behind
// it. Getting this flag wrong in the other direction would make checkTxNesting stop scanning
// (and so never detect the real over-nesting) the instant any such throwaway field appears
// anywhere earlier in a transaction's body.
func TestCheckTxNesting_BodyAllowsNonCriticalFields(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	bodyWithNonCritical := append(nonCriticalUnknownField(), nestedAnyBody(maxTxNestingRecursion+50)...)
	require.Error(t, checkTxNesting(registry, txRawWithBody(bodyWithNonCritical)))
}

// TestCheckTxNesting_AuthInfoRejectsNonCriticalFields pins checkTxNesting's own choice of
// allowUnknownNonCriticals for AuthInfo (false, matching DefaultTxDecoder's identical strict
// choice for auth_info_bytes): a non-critical unknown field placed before a real, deep chain
// is itself treated as a decode failure and swallowed as such -- left for the real, strict
// decode path to reject on its own terms -- rather than skipped over to keep scanning deeper.
func TestCheckTxNesting_AuthInfoRejectsNonCriticalFields(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	publicKeyAny := anyEnvelope(legacyAminoPubKeyTypeURL, nestedAnyBodyField(maxTxNestingRecursion+50, legacyAminoPubKeyTypeURL, 2))
	signerInfo := lenField(1, publicKeyAny)
	authInfoContent := append(nonCriticalUnknownField(), lenField(1, signerInfo)...)
	require.NoError(t, checkTxNesting(registry, lenField(2, authInfoContent)))
}

// AuthInfo (auth_info_bytes, field 2) is scanned exactly like TxBody -- proven by
// TestCheckTxNesting_SignerInfoPublicKeyNotBypassed, which builds a genuinely AuthInfo-shaped
// deep chain (via AuthInfo.signer_infos -> SignerInfo.public_key) and confirms it is rejected.
// nestedAnyBody itself is TxBody/gov-proposal-shaped (field 1 = repeated Any), not
// AuthInfo-shaped (field 1 = repeated SignerInfo, an ordinary message) -- placing it directly
// as auth_info_bytes would exercise a different, accidental field-1 reinterpretation rather
// than a real AuthInfo nesting scenario.

func TestCheckTxNestingIgnoresNonProtoAndEmpty(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	require.NoError(t, checkTxNesting(registry, nil))
	require.NoError(t, checkTxNesting(registry, []byte{0xff, 0xff, 0xff}))
}

// TestCheckTxNesting_DuplicateBodyBytesUsesLastOccurrence exercises TxRaw's own field-1
// (body_bytes) semantics against a duplicated occurrence. proto.Unmarshal implements
// last-field-wins for a singular field exactly like the real decoder's own cdc.Unmarshal does,
// so whichever occurrence comes last is the only one that matters -- a shallow decoy AFTER a
// deep chain is what the real decoder actually processes (and must not be rejected for), while
// a shallow decoy BEFORE a deep chain must not hide it (the deep one is what's actually used).
func TestCheckTxNesting_DuplicateBodyBytesUsesLastOccurrence(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	shallow := nestedAnyBody(1)
	deep := nestedAnyBody(maxTxNestingRecursion + 1)

	require.Error(t, checkTxNesting(registry, append(lenField(1, shallow), lenField(1, deep)...)),
		"a shallow decoy before the real, deep body_bytes must not hide it")
	require.NoError(t, checkTxNesting(registry, append(lenField(1, deep), lenField(1, shallow)...)),
		"the real decoder only ever sees the last occurrence, so a deep decoy before the real, shallow body_bytes must not be rejected")
}

// signerInfoAuthInfoBodyWithNestedPublicKey builds TxRaw.auth_info_bytes shaped as
// AuthInfo{signer_infos: [SignerInfo{public_key: Any{LegacyAminoPubKey, value: <depth-1-deep
// multisig self-chain>}}]}, reproducing the originally reported bypass: a deep Any chain
// reachable only by first descending through the ordinary (non-Any) AuthInfo.signer_infos ->
// SignerInfo edge before the first Any (SignerInfo's own public_key field) ever resolves.
// depth is the total Any-resolution count: the outer LegacyAminoPubKey envelope is one
// resolution, so the inner chain needs depth-1 more.
func signerInfoAuthInfoBodyWithNestedPublicKey(depth int) []byte {
	publicKeyAny := anyEnvelope(legacyAminoPubKeyTypeURL, nestedAnyBodyField(depth-1, legacyAminoPubKeyTypeURL, 2))
	signerInfo := lenField(1, publicKeyAny) // SignerInfo.public_key, field 1
	authInfo := lenField(1, signerInfo)     // AuthInfo.signer_infos, field 1
	return lenField(2, authInfo)            // TxRaw.auth_info_bytes, field 2
}

// TestCheckTxNesting_SignerInfoPublicKeyNotBypassed reproduces the originally reported bypass:
// a deep Any chain hidden behind the ordinary AuthInfo.signer_infos -> SignerInfo edge (with
// SignerInfo's own Any-typed public_key field where the chain actually starts) must still be
// caught, exactly like a chain hidden directly under TxBody/AuthInfo. The extra ordinary hop
// before the first Any costs one extra unit of maxTxNestingRecursion versus a bare gov
// self-chain -- pinned empirically, per TestCheckTxNestingDepthBoundary's comment on
// unknownproto's own recursion accounting.
func TestCheckTxNesting_SignerInfoPublicKeyNotBypassed(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	require.NoError(t, checkTxNesting(registry, signerInfoAuthInfoBodyWithNestedPublicKey(maxTxNestingRecursion-2)))
	require.Error(t, checkTxNesting(registry, signerInfoAuthInfoBodyWithNestedPublicKey(maxTxNestingRecursion-1)))
}

// applicationMsgWithScalarField builds TxRaw.body_bytes shaped as
// TxBody{messages: [Any{typeURL, value: Msg{fieldNum: scalarValue}}]}.
func applicationMsgWithScalarField(typeURL string, fieldNum protowire.Number, scalarValue []byte) []byte {
	msg := lenField(fieldNum, scalarValue)
	msgAny := anyEnvelope(typeURL, msg)
	return txRawWithBody(lenField(1, msgAny)) // TxBody.messages, field 1
}

// TestCheckTxNesting_ScalarFieldOfResolvedMsgNotDescended is the regression test for a real
// PoS-portal-confirmed false-positive-rejection path: MsgEventRecord.Data carries the raw L1
// state-sync event payload, which RootChainManager._depositFor / ERC1155Predicate.lockTokens
// (maticnetwork/pos-portal) forward from an attacker-supplied deposit calldata argument
// completely verbatim and unbounded. Once resolved into the real, registered MsgEventRecord
// type, its field 6 (data) is declared bytes in the real descriptor -- the guard's own real
// traversal (mirroring the decoder's) never even inspects that field's content for Any-shape,
// so a legitimate deposit whose Data bytes happen to look like a deep Any chain is never
// descended into or rejected. Unlike the old byte-heuristic guard, no special-casing is needed
// for this: the field is genuinely, provably terminal per its own real descriptor.
func TestCheckTxNesting_ScalarFieldOfResolvedMsgNotDescended(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	fakeNestedPayload := nestedAnyBody(maxTxNestingRecursion + 50)
	tx := applicationMsgWithScalarField("/heimdallv2.clerk.MsgEventRecord", 6, fakeNestedPayload)
	require.NoError(t, checkTxNesting(registry, tx))
}

// TestCheckTxNesting_ScalarFieldShapedAsAllowlistTypeNotDescended closes the sharper variant
// Codex's adversarial review specifically flagged against the old allowlist design: Data
// crafted as a direct google.protobuf.Any envelope naming a genuinely registered,
// self-chaining type (gov's MsgSubmitProposal) must be just as inert as an arbitrary,
// unregistered type URL in the same field -- because the field itself, not the type URL it
// happens to contain, is what the real descriptor says is terminal.
func TestCheckTxNesting_ScalarFieldShapedAsAllowlistTypeNotDescended(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	fakeChainUsingRealTypeURL := nestedAnyBody(maxTxNestingRecursion + 50)
	tx := applicationMsgWithScalarField("/heimdallv2.clerk.MsgEventRecord", 6, anyEnvelope(govProposalTypeURL, fakeChainUsingRealTypeURL))
	require.NoError(t, checkTxNesting(registry, tx))
}

// TestCheckTxNesting_ProposalOwnFieldsNotScanned confirms the real descriptor, not a
// hand-maintained allowlist, is what stops a resolved gov proposal's OTHER fields
// (Title/Summary/Metadata/Proposer, all plain strings the submitter fully controls) from being
// probed for further Any content: proposer (field 3) crafted as a direct Any chain using the
// proposal's own type URL must not trip the depth cap, because proposer's real descriptor type
// is string, not Any.
func TestCheckTxNesting_ProposalOwnFieldsNotScanned(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	fakeChainInProposerField := anyEnvelope(govProposalTypeURL, nestedAnyBody(maxTxNestingRecursion+50))
	proposal := lenField(3, fakeChainInProposerField) // MsgSubmitProposal.proposer, field 3
	tx := txRawWithBody(lenField(1, anyEnvelope(govProposalTypeURL, proposal)))
	require.NoError(t, checkTxNesting(registry, tx))
}

// TestCheckTxNesting_GovContentTypesStillCountTowardDepth confirms gov's two other real
// Any-bearing registered types (v1 MsgExecLegacyContent.content and v1beta1
// MsgSubmitProposal.content, both registered via std.RegisterInterfaces and reachable through
// this app's v1beta1 legacy gov router) are followed by the real descriptor-driven traversal
// exactly like the v1 messages field is -- no separate allowlist entry needed, and no
// under-counting risk from one being missing by hand-maintenance oversight.
func TestCheckTxNesting_GovContentTypesStillCountTowardDepth(t *testing.T) {
	registry := newTestInterfaceRegistry(t)
	for _, typeURL := range []string{"/cosmos.gov.v1.MsgExecLegacyContent", "/cosmos.gov.v1beta1.MsgSubmitProposal"} {
		require.NoError(t, checkTxNesting(registry, txRawWithBody(nestedAnyBodyField(maxTxNestingRecursion-1, typeURL, 1))), typeURL)
		require.Error(t, checkTxNesting(registry, txRawWithBody(nestedAnyBodyField(maxTxNestingRecursion, typeURL, 1))), typeURL)
	}
}
