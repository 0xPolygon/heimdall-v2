package milestone

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMilestoneFastReflection(t *testing.T) {
	msg := &Milestone{
		Proposer:        "0xabc",
		StartBlock:      11,
		EndBlock:        22,
		Hash:            []byte{0x01, 0x02, 0x03},
		BorChainId:      "137",
		MilestoneId:     "mid-1",
		Timestamp:       33,
		TotalDifficulty: 44,
	}

	r := msg.ProtoReflect()
	desc := r.Descriptor()
	require.Equal(t, protoreflect.Name("Milestone"), desc.Name())
	require.Equal(t, protoreflect.FullName("heimdallv2.milestone.Milestone"), desc.FullName())
	_ = r.Type().Zero()
	_ = r.Type().New()
	_ = r.Type().Descriptor()
	_ = msg.slowProtoReflect()
	require.False(t, (*Milestone)(nil).ProtoReflect().IsValid())

	seen := make([]string, 0, desc.Fields().Len())
	r.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		seen = append(seen, string(fd.Name()))
		require.True(t, r.Has(fd))
		require.Equal(t, value.Interface(), r.Get(fd).Interface())
		return true
	})
	require.Len(t, seen, desc.Fields().Len())

	stopped := 0
	r.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		stopped++
		return false
	})
	require.Equal(t, 1, stopped)

	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)

		switch fd.Kind() {
		case protoreflect.StringKind:
			require.Equal(t, "", r.NewField(fd).String())
			r.Set(fd, protoreflect.ValueOfString("changed"))
			require.Equal(t, "changed", r.Get(fd).String())
		case protoreflect.Uint64Kind:
			require.Zero(t, r.NewField(fd).Uint())
			r.Set(fd, protoreflect.ValueOfUint64(999))
			require.Equal(t, uint64(999), r.Get(fd).Uint())
		case protoreflect.BytesKind:
			require.Nil(t, r.NewField(fd).Bytes())
			r.Set(fd, protoreflect.ValueOfBytes([]byte{0xAA, 0xBB}))
			require.Equal(t, []byte{0xAA, 0xBB}, r.Get(fd).Bytes())
		default:
			t.Fatalf("unexpected kind %s", fd.Kind())
		}

		require.True(t, r.Has(fd))
		require.Panics(t, func() { r.Mutable(fd) })
		r.Clear(fd)
		require.False(t, r.Has(fd))
	}

	// Exercise the default panic paths with a descriptor from another message.
	foreignFD := (&descriptorpb.FileOptions{}).ProtoReflect().Descriptor().Fields().Get(0)
	require.Panics(t, func() { r.Has(foreignFD) })
	require.Panics(t, func() { r.Clear(foreignFD) })
	require.Panics(t, func() { r.Get(foreignFD) })
	require.Panics(t, func() { r.Set(foreignFD, protoreflect.ValueOfString("x")) })
	require.Panics(t, func() { r.NewField(foreignFD) })
	require.Panics(t, func() { r.Mutable(foreignFD) })

	unknown := protoreflect.RawFields(protowire.AppendVarint(protowire.AppendTag(nil, 100, protowire.VarintType), 1))
	r.SetUnknown(unknown)
	require.Equal(t, unknown, r.GetUnknown())

	wireMsg := &Milestone{
		Proposer:        "0xabc",
		StartBlock:      11,
		EndBlock:        22,
		Hash:            []byte{0x01, 0x02, 0x03},
		BorChainId:      "137",
		MilestoneId:     "mid-1",
		Timestamp:       33,
		TotalDifficulty: 44,
	}
	wireMsg.ProtoReflect().SetUnknown(unknown)

	methods := wireMsg.ProtoReflect().ProtoMethods()
	require.NotNil(t, methods)
	require.NotNil(t, methods.Size)
	require.NotNil(t, methods.Marshal)
	require.NotNil(t, methods.Unmarshal)

	sizeOut := methods.Size(protoiface.SizeInput{Message: wireMsg.ProtoReflect()})
	require.Greater(t, sizeOut.Size, 0)

	marshalOut, err := methods.Marshal(protoiface.MarshalInput{Message: wireMsg.ProtoReflect()})
	require.NoError(t, err)
	require.Equal(t, sizeOut.Size, len(marshalOut.Buf))

	roundTrip := &Milestone{}
	_, err = methods.Unmarshal(protoiface.UnmarshalInput{Message: roundTrip.ProtoReflect(), Buf: marshalOut.Buf})
	require.NoError(t, err)
	require.Equal(t, wireMsg.Proposer, roundTrip.Proposer)
	require.Equal(t, wireMsg.StartBlock, roundTrip.StartBlock)
	require.Equal(t, wireMsg.EndBlock, roundTrip.EndBlock)
	require.Equal(t, wireMsg.Hash, roundTrip.Hash)
	require.Equal(t, wireMsg.BorChainId, roundTrip.BorChainId)
	require.Equal(t, wireMsg.MilestoneId, roundTrip.MilestoneId)
	require.Equal(t, wireMsg.Timestamp, roundTrip.Timestamp)
	require.Equal(t, wireMsg.TotalDifficulty, roundTrip.TotalDifficulty)
	require.Equal(t, unknown, roundTrip.ProtoReflect().GetUnknown())
	require.Equal(t, wireMsg.Proposer, wireMsg.GetProposer())
	require.Equal(t, wireMsg.StartBlock, wireMsg.GetStartBlock())
	require.Equal(t, wireMsg.EndBlock, wireMsg.GetEndBlock())
	require.Equal(t, wireMsg.Hash, wireMsg.GetHash())
	require.Equal(t, wireMsg.BorChainId, wireMsg.GetBorChainId())
	require.Equal(t, wireMsg.MilestoneId, wireMsg.GetMilestoneId())
	require.Equal(t, wireMsg.Timestamp, wireMsg.GetTimestamp())
	require.Equal(t, wireMsg.TotalDifficulty, wireMsg.GetTotalDifficulty())
	_, _ = wireMsg.Descriptor()
	_ = wireMsg.String()
	wireMsg.ProtoMessage()
	wireMsg.Reset()

	countMsg := &MilestoneCount{Count: 7}
	countReflect := countMsg.ProtoReflect()
	countDesc := countReflect.Descriptor()
	require.Equal(t, protoreflect.Name("MilestoneCount"), countDesc.Name())
	require.Equal(t, 1, countDesc.Fields().Len())
	_ = countReflect.Type().Zero()
	_ = countReflect.Type().New()
	_ = countReflect.Type().Descriptor()
	_ = countMsg.slowProtoReflect()
	require.False(t, (*MilestoneCount)(nil).ProtoReflect().IsValid())

	countField := countDesc.Fields().Get(0)
	require.Equal(t, protoreflect.Name("count"), countField.Name())
	require.True(t, countReflect.Has(countField))
	require.Equal(t, uint64(7), countReflect.Get(countField).Uint())
	require.Equal(t, uint64(0), countReflect.NewField(countField).Uint())
	require.Panics(t, func() { countReflect.Mutable(countField) })
	countReflect.Clear(countField)
	require.False(t, countReflect.Has(countField))
	countReflect.Set(countField, protoreflect.ValueOfUint64(9))
	require.Equal(t, uint64(9), countReflect.Get(countField).Uint())
	countReflect.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool { return false })

	countMethods := countReflect.ProtoMethods()
	require.NotNil(t, countMethods)
	sizeOut = countMethods.Size(protoiface.SizeInput{Message: countReflect})
	require.Greater(t, sizeOut.Size, 0)
	marshalOut, err = countMethods.Marshal(protoiface.MarshalInput{Message: countReflect})
	require.NoError(t, err)
	require.Equal(t, sizeOut.Size, len(marshalOut.Buf))
	countRoundTrip := &MilestoneCount{}
	_, err = countMethods.Unmarshal(protoiface.UnmarshalInput{Message: countRoundTrip.ProtoReflect(), Buf: marshalOut.Buf})
	require.NoError(t, err)
	require.Equal(t, countMsg.Count, countRoundTrip.Count)
	require.Equal(t, countMsg.Count, countMsg.GetCount())
	_, _ = countMsg.Descriptor()
	_ = countMsg.String()
	countMsg.ProtoMessage()
	countMsg.Reset()

	propMsg := &MilestoneProposition{
		BlockHashes:       [][]byte{{0x11}},
		StartBlockNumber:  123,
		ParentHash:        []byte{0x22},
		BlockTds:          []uint64{7},
		LatestBlockNumber: 130,
		LatestBlockHash:   []byte{0x33},
	}
	propReflect := propMsg.ProtoReflect()
	propDesc := propReflect.Descriptor()
	require.Equal(t, protoreflect.Name("MilestoneProposition"), propDesc.Name())
	_ = propReflect.Type().Zero()
	_ = propReflect.Type().New()
	_ = propReflect.Type().Descriptor()
	_ = propMsg.slowProtoReflect()
	require.False(t, (*MilestoneProposition)(nil).ProtoReflect().IsValid())

	propHashesFD := propDesc.Fields().ByName("block_hashes")
	propHashes := propReflect.Mutable(propHashesFD).List()
	require.Equal(t, 1, propHashes.Len())
	require.Equal(t, []byte{0x11}, propHashes.Get(0).Bytes())
	propHashes.Set(0, protoreflect.ValueOfBytes([]byte{0x12}))
	propHashes.Append(protoreflect.ValueOfBytes([]byte{0x13}))
	require.Equal(t, 2, propHashes.Len())
	require.Panics(t, func() { propHashes.AppendMutable() })
	propHashes.Truncate(1)
	require.Equal(t, 1, propHashes.Len())
	require.Empty(t, propHashes.NewElement().Bytes())
	require.True(t, propHashes.IsValid())

	propTdsFD := propDesc.Fields().ByName("block_tds")
	propTds := propReflect.Mutable(propTdsFD).List()
	require.Equal(t, 1, propTds.Len())
	require.Equal(t, uint64(7), propTds.Get(0).Uint())
	propTds.Set(0, protoreflect.ValueOfUint64(8))
	propTds.Append(protoreflect.ValueOfUint64(9))
	require.Equal(t, 2, propTds.Len())
	require.Panics(t, func() { propTds.AppendMutable() })
	propTds.Truncate(1)
	require.Equal(t, 1, propTds.Len())
	require.Zero(t, propTds.NewElement().Uint())
	require.True(t, propTds.IsValid())

	require.Equal(t, uint64(123), propMsg.GetStartBlockNumber())
	require.Equal(t, []byte{0x22}, propMsg.GetParentHash())
	require.Equal(t, uint64(130), propMsg.GetLatestBlockNumber())
	require.Equal(t, []byte{0x33}, propMsg.GetLatestBlockHash())
	require.Equal(t, propMsg.BlockHashes, propMsg.GetBlockHashes())
	require.Equal(t, propMsg.StartBlockNumber, propMsg.GetStartBlockNumber())
	require.Equal(t, propMsg.ParentHash, propMsg.GetParentHash())
	require.Equal(t, propMsg.BlockTds, propMsg.GetBlockTds())
	require.Equal(t, propMsg.LatestBlockNumber, propMsg.GetLatestBlockNumber())
	require.Equal(t, propMsg.LatestBlockHash, propMsg.GetLatestBlockHash())
	_, _ = propMsg.Descriptor()
	_ = propMsg.String()
	propMsg.ProtoMessage()
	propMsg.Reset()
	propMsg.String()

	paramsMsg := &Params{
		MaxMilestonePropositionLength: 5,
		FfMilestoneThreshold:          10,
		FfMilestoneBlockInterval:      5,
	}
	paramsReflect := paramsMsg.ProtoReflect()
	paramsDesc := paramsReflect.Descriptor()
	require.Equal(t, protoreflect.Name("Params"), paramsDesc.Name())
	_ = paramsReflect.Type().Zero()
	_ = paramsReflect.Type().New()
	_ = paramsReflect.Type().Descriptor()
	_ = paramsMsg.slowProtoReflect()
	require.False(t, (*Params)(nil).ProtoReflect().IsValid())
	require.Equal(t, uint64(5), paramsMsg.GetMaxMilestonePropositionLength())
	require.Equal(t, uint64(10), paramsMsg.GetFfMilestoneThreshold())
	require.Equal(t, uint64(5), paramsMsg.GetFfMilestoneBlockInterval())
	_, _ = paramsMsg.Descriptor()
	require.Equal(t, paramsMsg.MaxMilestonePropositionLength, paramsMsg.GetMaxMilestonePropositionLength())
	require.Equal(t, paramsMsg.FfMilestoneThreshold, paramsMsg.GetFfMilestoneThreshold())
	require.Equal(t, paramsMsg.FfMilestoneBlockInterval, paramsMsg.GetFfMilestoneBlockInterval())
	_ = paramsMsg.String()
	paramsMsg.ProtoMessage()
	paramsMsg.Reset()
	paramsMsg.String()
}
