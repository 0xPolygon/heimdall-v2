package grpc

import (
	"math/big"
	"testing"

	proto "github.com/0xPolygon/polyproto/bor"
	protoutil "github.com/0xPolygon/polyproto/utils"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	googleproto "google.golang.org/protobuf/proto"
)

// fullHeaderToProto mirrors bor's headerToProtoBorHeader
// A bor that predates that change populates only number/parentHash/time,
// which truncatedHeaderToProto reproduces below.
func fullHeaderToProto(h *ethTypes.Header) *proto.Header {
	out := &proto.Header{
		Number:      h.Number.Uint64(),
		ParentHash:  protoutil.ConvertHashToH256(h.ParentHash),
		Time:        h.Time,
		UncleHash:   protoutil.ConvertHashToH256(h.UncleHash),
		Coinbase:    protoutil.ConvertAddressToH160(h.Coinbase),
		StateRoot:   protoutil.ConvertHashToH256(h.Root),
		TxRoot:      protoutil.ConvertHashToH256(h.TxHash),
		ReceiptRoot: protoutil.ConvertHashToH256(h.ReceiptHash),
		Bloom:       append([]byte(nil), h.Bloom.Bytes()...),
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		ExtraData:   append([]byte(nil), h.Extra...),
		MixDigest:   protoutil.ConvertHashToH256(h.MixDigest),
		Nonce:       append([]byte(nil), h.Nonce[:]...),
	}
	if h.Difficulty != nil {
		out.Difficulty = h.Difficulty.Bytes()
	}
	if h.BaseFee != nil {
		out.BaseFee = h.BaseFee.Bytes()
	}
	if h.WithdrawalsHash != nil {
		out.WithdrawalsHash = protoutil.ConvertHashToH256(*h.WithdrawalsHash)
	}
	if h.BlobGasUsed != nil {
		v := *h.BlobGasUsed
		out.BlobGasUsed = &v
	}
	if h.ExcessBlobGas != nil {
		v := *h.ExcessBlobGas
		out.ExcessBlobGas = &v
	}
	if h.ParentBeaconRoot != nil {
		out.ParentBeaconBlockRoot = protoutil.ConvertHashToH256(*h.ParentBeaconRoot)
	}
	if h.RequestsHash != nil {
		out.RequestsHash = protoutil.ConvertHashToH256(*h.RequestsHash)
	}
	return out
}

// truncatedHeaderToProto reproduces what a bor built against polyproto < v0.0.8
// (pre-#2194) sends: only the three fields that existed in the v0.0.7 Header.
func truncatedHeaderToProto(h *ethTypes.Header) *proto.Header {
	return &proto.Header{
		Number:     h.Number.Uint64(),
		ParentHash: protoutil.ConvertHashToH256(h.ParentHash),
		Time:       h.Time,
	}
}

// mainnetBlock89174945 builds the canonical header for mainnet bor block
// 89174945 (one of the partner's reported parity-mismatch blocks). All
// post-Cancun/Prague optional fields are nil on this block.
func mainnetBlock89174945(t *testing.T) *ethTypes.Header {
	t.Helper()
	h := &ethTypes.Header{
		ParentHash:  common.HexToHash("0xf78b60d26de9e14d36e904fd5b84ef2d7226b1313989607e8d0ca02879ec88f3"),
		UncleHash:   common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Root:        common.HexToHash("0xbf2b92de1f6335481eed74b35075d0fe5f32b05876116fbfed5dac59f5341ba6"),
		TxHash:      common.HexToHash("0x5165a372ac711674fcde5277ca71fbb5e4d9ebcd50a51bf9811bc38fb01944ef"),
		ReceiptHash: common.HexToHash("0xecba6ef9201274b73f2e7059a2a58b919f08155b7059287f7dd8c3a05327580d"),
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(89174945),
		GasLimit:    0x9896800,
		GasUsed:     0x21341af,
		Time:        0x6a3e4b55,
		Extra:       common.FromHex(extra89174945),
		MixDigest:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		BaseFee:     big.NewInt(0x3ad2a03045),
	}
	h.Bloom.SetBytes(common.FromHex(bloom89174945))
	return h
}

const canonicalHash89174945 = "0xa495c1c542982a5e0ec31d9b03d25c55235fdecf73867dd4b80eb184967d8844"

const bloom89174945 = "0xbeb10aba6a0f36a1089841c3910a58c731765c40f871d89ab8d496fe2465f874b6481e17a74f5af4f7d5129db0c1b9eb72e5ff3660fcab4f7bce02ef0d3d60d57d8dbfb9fe64aabfa89482199c16dbe2bbb024120ffeb0a943eb2f19b7cde1cb6e91eb0f66349a133d4b795103fdcb24ee2799f32f1d839b86aaf29642d82cf94cf3e9d005985b16bfd1bfe57475f07d6ab98f0eaf2eb44dd1d36070b5b533fbebb2dfe8bec943e226730a948933c739ca2ee9a5495aa4e5dce44ed61e186deff1013de31bd6ee4d7f41595af4c6b33466cf3f70dc2fb81df5d6f859bf667095b7532cea5933a009e7119f3b412095ae668f2a6612b0695f579a9cbddbf1ed47"

const extra89174945 = "0xd78301100883626f7288676f312e32362e33856c696e75780000000000000000f8cd80f8c4c0c0c0c0c102c0c104c101c106c108c0c109c10bc10ac0c10cc10fc110c111c112c113c0c114c10dc117c0c119c118c107c11bc11dc11ec11fc120c121c122c123c124c125c126c127c128c129c12ac12bc12cc12dc12ec12fc130c131c132c133c134c135c136c137c138c139c116c21c3ac105c33c3d3bc13ec13ec13fc141c142c143c0c0c13dc143c24844c147c149c0c14bc14dc14ec0c140c14fc13cc11ac0c0c24053c0c158c0c152c13cc15bc157c15ec15fc160c161c0c162c0c25157c0c0c08406acfc00408835b2a6f205a491e9bbdcfc7e41aff312b247983083ee60b9f06ecaae4bf36440c6081350ad04f328f666e61ea4168157a4e3fb9827686f8576235978e997c101"

// roundTrip marshals the proto header to wire bytes and back, then reconstructs
// the eth header — faithful to what crosses the gRPC boundary.
func roundTrip(t *testing.T, p *proto.Header) *ethTypes.Header {
	t.Helper()
	raw, err := googleproto.Marshal(p)
	require.NoError(t, err)
	var got proto.Header
	require.NoError(t, googleproto.Unmarshal(raw, &got))
	return protoHeaderToEthHeader(&got)
}

// TestProtoHeaderRoundTrip_FullPopulate confirms that a bor on polyproto v0.0.8
// (full Header populate) round-trips to the identical block hash — i.e. our
// shipped bor v2.8.3 + heimdall v0.9.0 do NOT produce a parity mismatch.
func TestProtoHeaderRoundTrip_FullPopulate(t *testing.T) {
	h := mainnetBlock89174945(t)
	require.Equal(t, canonicalHash89174945, h.Hash().Hex(),
		"sanity: reconstructed canonical header must hash to the real block hash")

	got := roundTrip(t, fullHeaderToProto(h))
	require.Equal(t, h.Hash(), got.Hash(),
		"full-populate gRPC round-trip must preserve the block hash")
}

// TestProtoHeaderRoundTrip_TruncatedPopulate reproduces the partner's FATAL: a
// bor built against polyproto < v0.0.8 sends only number/parentHash/time, so the
// reconstructed header has 18 zeroed fields and a different hash on every block.
func TestProtoHeaderRoundTrip_TruncatedPopulate(t *testing.T) {
	h := mainnetBlock89174945(t)

	got := roundTrip(t, truncatedHeaderToProto(h))
	require.NotEqual(t, h.Hash(), got.Hash(),
		"truncated (pre-v0.0.8) header must diverge — this is the reported mismatch")
	require.Equal(t, h.ParentHash, got.ParentHash, "the 3 carried fields still survive")
	require.Equal(t, h.Number.Uint64(), got.Number.Uint64())
	require.Equal(t, common.Hash{}, got.Root, "state root is zeroed without the full populate")
}
