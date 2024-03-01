// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package checkpointv1

import (
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/api/heimdallv2/types"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	reflect "reflect"
	sync "sync"
)

var _ protoreflect.List = (*_GenesisState_5_list)(nil)

type _GenesisState_5_list struct {
	list *[]*types.Checkpoint
}

func (x *_GenesisState_5_list) Len() int {
	if x.list == nil {
		return 0
	}
	return len(*x.list)
}

func (x *_GenesisState_5_list) Get(i int) protoreflect.Value {
	return protoreflect.ValueOfMessage((*x.list)[i].ProtoReflect())
}

func (x *_GenesisState_5_list) Set(i int, value protoreflect.Value) {
	valueUnwrapped := value.Message()
	concreteValue := valueUnwrapped.Interface().(*types.Checkpoint)
	(*x.list)[i] = concreteValue
}

func (x *_GenesisState_5_list) Append(value protoreflect.Value) {
	valueUnwrapped := value.Message()
	concreteValue := valueUnwrapped.Interface().(*types.Checkpoint)
	*x.list = append(*x.list, concreteValue)
}

func (x *_GenesisState_5_list) AppendMutable() protoreflect.Value {
	v := new(types.Checkpoint)
	*x.list = append(*x.list, v)
	return protoreflect.ValueOfMessage(v.ProtoReflect())
}

func (x *_GenesisState_5_list) Truncate(n int) {
	for i := n; i < len(*x.list); i++ {
		(*x.list)[i] = nil
	}
	*x.list = (*x.list)[:n]
}

func (x *_GenesisState_5_list) NewElement() protoreflect.Value {
	v := new(types.Checkpoint)
	return protoreflect.ValueOfMessage(v.ProtoReflect())
}

func (x *_GenesisState_5_list) IsValid() bool {
	return x.list != nil
}

var (
	md_GenesisState                     protoreflect.MessageDescriptor
	fd_GenesisState_params              protoreflect.FieldDescriptor
	fd_GenesisState_buffered_checkpoint protoreflect.FieldDescriptor
	fd_GenesisState_last_no_a_c_k       protoreflect.FieldDescriptor
	fd_GenesisState_ack_count           protoreflect.FieldDescriptor
	fd_GenesisState_checkpoints         protoreflect.FieldDescriptor
)

func init() {
	file_heimdallv2_checkpoint_v1_genesis_proto_init()
	md_GenesisState = File_heimdallv2_checkpoint_v1_genesis_proto.Messages().ByName("GenesisState")
	fd_GenesisState_params = md_GenesisState.Fields().ByName("params")
	fd_GenesisState_buffered_checkpoint = md_GenesisState.Fields().ByName("buffered_checkpoint")
	fd_GenesisState_last_no_a_c_k = md_GenesisState.Fields().ByName("last_no_a_c_k")
	fd_GenesisState_ack_count = md_GenesisState.Fields().ByName("ack_count")
	fd_GenesisState_checkpoints = md_GenesisState.Fields().ByName("checkpoints")
}

var _ protoreflect.Message = (*fastReflection_GenesisState)(nil)

type fastReflection_GenesisState GenesisState

func (x *GenesisState) ProtoReflect() protoreflect.Message {
	return (*fastReflection_GenesisState)(x)
}

func (x *GenesisState) slowProtoReflect() protoreflect.Message {
	mi := &file_heimdallv2_checkpoint_v1_genesis_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_GenesisState_messageType fastReflection_GenesisState_messageType
var _ protoreflect.MessageType = fastReflection_GenesisState_messageType{}

type fastReflection_GenesisState_messageType struct{}

func (x fastReflection_GenesisState_messageType) Zero() protoreflect.Message {
	return (*fastReflection_GenesisState)(nil)
}
func (x fastReflection_GenesisState_messageType) New() protoreflect.Message {
	return new(fastReflection_GenesisState)
}
func (x fastReflection_GenesisState_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_GenesisState
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_GenesisState) Descriptor() protoreflect.MessageDescriptor {
	return md_GenesisState
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_GenesisState) Type() protoreflect.MessageType {
	return _fastReflection_GenesisState_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_GenesisState) New() protoreflect.Message {
	return new(fastReflection_GenesisState)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_GenesisState) Interface() protoreflect.ProtoMessage {
	return (*GenesisState)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_GenesisState) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.Params != nil {
		value := protoreflect.ValueOfMessage(x.Params.ProtoReflect())
		if !f(fd_GenesisState_params, value) {
			return
		}
	}
	if x.BufferedCheckpoint != nil {
		value := protoreflect.ValueOfMessage(x.BufferedCheckpoint.ProtoReflect())
		if !f(fd_GenesisState_buffered_checkpoint, value) {
			return
		}
	}
	if x.LastNoACK != uint64(0) {
		value := protoreflect.ValueOfUint64(x.LastNoACK)
		if !f(fd_GenesisState_last_no_a_c_k, value) {
			return
		}
	}
	if x.AckCount != uint64(0) {
		value := protoreflect.ValueOfUint64(x.AckCount)
		if !f(fd_GenesisState_ack_count, value) {
			return
		}
	}
	if len(x.Checkpoints) != 0 {
		value := protoreflect.ValueOfList(&_GenesisState_5_list{list: &x.Checkpoints})
		if !f(fd_GenesisState_checkpoints, value) {
			return
		}
	}
}

// Has reports whether a field is populated.
//
// Some fields have the property of nullability where it is possible to
// distinguish between the default value of a field and whether the field
// was explicitly populated with the default value. Singular message fields,
// member fields of a oneof, and proto2 scalar fields are nullable. Such
// fields are populated only if explicitly set.
//
// In other cases (aside from the nullable cases above),
// a proto3 scalar field is populated if it contains a non-zero value, and
// a repeated field is populated if it is non-empty.
func (x *fastReflection_GenesisState) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		return x.Params != nil
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		return x.BufferedCheckpoint != nil
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		return x.LastNoACK != uint64(0)
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		return x.AckCount != uint64(0)
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		return len(x.Checkpoints) != 0
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_GenesisState) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		x.Params = nil
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		x.BufferedCheckpoint = nil
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		x.LastNoACK = uint64(0)
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		x.AckCount = uint64(0)
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		x.Checkpoints = nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_GenesisState) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		value := x.Params
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		value := x.BufferedCheckpoint
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		value := x.LastNoACK
		return protoreflect.ValueOfUint64(value)
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		value := x.AckCount
		return protoreflect.ValueOfUint64(value)
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		if len(x.Checkpoints) == 0 {
			return protoreflect.ValueOfList(&_GenesisState_5_list{})
		}
		listValue := &_GenesisState_5_list{list: &x.Checkpoints}
		return protoreflect.ValueOfList(listValue)
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", descriptor.FullName()))
	}
}

// Set stores the value for a field.
//
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType.
// When setting a composite type, it is unspecified whether the stored value
// aliases the source's memory in any way. If the composite value is an
// empty, read-only value, then it panics.
//
// Set is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_GenesisState) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		x.Params = value.Message().Interface().(*types.Params)
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		x.BufferedCheckpoint = value.Message().Interface().(*types.Checkpoint)
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		x.LastNoACK = value.Uint()
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		x.AckCount = value.Uint()
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		lv := value.List()
		clv := lv.(*_GenesisState_5_list)
		x.Checkpoints = *clv.list
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", fd.FullName()))
	}
}

// Mutable returns a mutable reference to a composite type.
//
// If the field is unpopulated, it may allocate a composite value.
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType
// if not already stored.
// It panics if the field does not contain a composite type.
//
// Mutable is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_GenesisState) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		if x.Params == nil {
			x.Params = new(types.Params)
		}
		return protoreflect.ValueOfMessage(x.Params.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		if x.BufferedCheckpoint == nil {
			x.BufferedCheckpoint = new(types.Checkpoint)
		}
		return protoreflect.ValueOfMessage(x.BufferedCheckpoint.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		if x.Checkpoints == nil {
			x.Checkpoints = []*types.Checkpoint{}
		}
		value := &_GenesisState_5_list{list: &x.Checkpoints}
		return protoreflect.ValueOfList(value)
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		panic(fmt.Errorf("field last_no_a_c_k of message heimdallv2.checkpoint.v1.GenesisState is not mutable"))
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		panic(fmt.Errorf("field ack_count of message heimdallv2.checkpoint.v1.GenesisState is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_GenesisState) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.checkpoint.v1.GenesisState.params":
		m := new(types.Params)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint":
		m := new(types.Checkpoint)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "heimdallv2.checkpoint.v1.GenesisState.last_no_a_c_k":
		return protoreflect.ValueOfUint64(uint64(0))
	case "heimdallv2.checkpoint.v1.GenesisState.ack_count":
		return protoreflect.ValueOfUint64(uint64(0))
	case "heimdallv2.checkpoint.v1.GenesisState.checkpoints":
		list := []*types.Checkpoint{}
		return protoreflect.ValueOfList(&_GenesisState_5_list{list: &list})
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.checkpoint.v1.GenesisState"))
		}
		panic(fmt.Errorf("message heimdallv2.checkpoint.v1.GenesisState does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_GenesisState) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in heimdallv2.checkpoint.v1.GenesisState", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_GenesisState) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_GenesisState) SetUnknown(fields protoreflect.RawFields) {
	x.unknownFields = fields
}

// IsValid reports whether the message is valid.
//
// An invalid message is an empty, read-only value.
//
// An invalid message often corresponds to a nil pointer of the concrete
// message type, but the details are implementation dependent.
// Validity is not part of the protobuf data model, and may not
// be preserved in marshaling or other operations.
func (x *fastReflection_GenesisState) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_GenesisState) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*GenesisState)
		if x == nil {
			return protoiface.SizeOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Size:              0,
			}
		}
		options := runtime.SizeInputToOptions(input)
		_ = options
		var n int
		var l int
		_ = l
		if x.Params != nil {
			l = options.Size(x.Params)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.BufferedCheckpoint != nil {
			l = options.Size(x.BufferedCheckpoint)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.LastNoACK != 0 {
			n += 1 + runtime.Sov(uint64(x.LastNoACK))
		}
		if x.AckCount != 0 {
			n += 1 + runtime.Sov(uint64(x.AckCount))
		}
		if len(x.Checkpoints) > 0 {
			for _, e := range x.Checkpoints {
				l = options.Size(e)
				n += 1 + l + runtime.Sov(uint64(l))
			}
		}
		if x.unknownFields != nil {
			n += len(x.unknownFields)
		}
		return protoiface.SizeOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Size:              n,
		}
	}

	marshal := func(input protoiface.MarshalInput) (protoiface.MarshalOutput, error) {
		x := input.Message.Interface().(*GenesisState)
		if x == nil {
			return protoiface.MarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Buf:               input.Buf,
			}, nil
		}
		options := runtime.MarshalInputToOptions(input)
		_ = options
		size := options.Size(x)
		dAtA := make([]byte, size)
		i := len(dAtA)
		_ = i
		var l int
		_ = l
		if x.unknownFields != nil {
			i -= len(x.unknownFields)
			copy(dAtA[i:], x.unknownFields)
		}
		if len(x.Checkpoints) > 0 {
			for iNdEx := len(x.Checkpoints) - 1; iNdEx >= 0; iNdEx-- {
				encoded, err := options.Marshal(x.Checkpoints[iNdEx])
				if err != nil {
					return protoiface.MarshalOutput{
						NoUnkeyedLiterals: input.NoUnkeyedLiterals,
						Buf:               input.Buf,
					}, err
				}
				i -= len(encoded)
				copy(dAtA[i:], encoded)
				i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
				i--
				dAtA[i] = 0x2a
			}
		}
		if x.AckCount != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.AckCount))
			i--
			dAtA[i] = 0x20
		}
		if x.LastNoACK != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.LastNoACK))
			i--
			dAtA[i] = 0x18
		}
		if x.BufferedCheckpoint != nil {
			encoded, err := options.Marshal(x.BufferedCheckpoint)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0x12
		}
		if x.Params != nil {
			encoded, err := options.Marshal(x.Params)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0xa
		}
		if input.Buf != nil {
			input.Buf = append(input.Buf, dAtA...)
		} else {
			input.Buf = dAtA
		}
		return protoiface.MarshalOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Buf:               input.Buf,
		}, nil
	}
	unmarshal := func(input protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		x := input.Message.Interface().(*GenesisState)
		if x == nil {
			return protoiface.UnmarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Flags:             input.Flags,
			}, nil
		}
		options := runtime.UnmarshalInputToOptions(input)
		_ = options
		dAtA := input.Buf
		l := len(dAtA)
		iNdEx := 0
		for iNdEx < l {
			preIndex := iNdEx
			var wire uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
				}
				if iNdEx >= l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				wire |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			fieldNum := int32(wire >> 3)
			wireType := int(wire & 0x7)
			if wireType == 4 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.Params == nil {
					x.Params = &types.Params{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.Params); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field BufferedCheckpoint", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.BufferedCheckpoint == nil {
					x.BufferedCheckpoint = &types.Checkpoint{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.BufferedCheckpoint); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 3:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field LastNoACK", wireType)
				}
				x.LastNoACK = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.LastNoACK |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 4:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field AckCount", wireType)
				}
				x.AckCount = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.AckCount |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 5:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Checkpoints", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Checkpoints = append(x.Checkpoints, &types.Checkpoint{})
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.Checkpoints[len(x.Checkpoints)-1]); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			default:
				iNdEx = preIndex
				skippy, err := runtime.Skip(dAtA[iNdEx:])
				if err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				if (skippy < 0) || (iNdEx+skippy) < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if (iNdEx + skippy) > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if !options.DiscardUnknown {
					x.unknownFields = append(x.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
				}
				iNdEx += skippy
			}
		}

		if iNdEx > l {
			return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
		}
		return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, nil
	}
	return &protoiface.Methods{
		NoUnkeyedLiterals: struct{}{},
		Flags:             protoiface.SupportMarshalDeterministic | protoiface.SupportUnmarshalDiscardUnknown,
		Size:              size,
		Marshal:           marshal,
		Unmarshal:         unmarshal,
		Merge:             nil,
		CheckInitialized:  nil,
	}
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: heimdallv2/checkpoint/v1/genesis.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// GenesisState defines the staking module's genesis state.
type GenesisState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// It defines the params
	Params *types.Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params,omitempty"`
	// BufferedCheckpoint defines the buffered checkpoint
	BufferedCheckpoint *types.Checkpoint   `protobuf:"bytes,2,opt,name=buffered_checkpoint,json=bufferedCheckpoint,proto3" json:"buffered_checkpoint,omitempty"`
	LastNoACK          uint64              `protobuf:"varint,3,opt,name=last_no_a_c_k,json=lastNoACK,proto3" json:"last_no_a_c_k,omitempty"`
	AckCount           uint64              `protobuf:"varint,4,opt,name=ack_count,json=ackCount,proto3" json:"ack_count,omitempty"`
	Checkpoints        []*types.Checkpoint `protobuf:"bytes,5,rep,name=checkpoints,proto3" json:"checkpoints,omitempty"`
}

func (x *GenesisState) Reset() {
	*x = GenesisState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_heimdallv2_checkpoint_v1_genesis_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GenesisState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenesisState) ProtoMessage() {}

// Deprecated: Use GenesisState.ProtoReflect.Descriptor instead.
func (*GenesisState) Descriptor() ([]byte, []int) {
	return file_heimdallv2_checkpoint_v1_genesis_proto_rawDescGZIP(), []int{0}
}

func (x *GenesisState) GetParams() *types.Params {
	if x != nil {
		return x.Params
	}
	return nil
}

func (x *GenesisState) GetBufferedCheckpoint() *types.Checkpoint {
	if x != nil {
		return x.BufferedCheckpoint
	}
	return nil
}

func (x *GenesisState) GetLastNoACK() uint64 {
	if x != nil {
		return x.LastNoACK
	}
	return 0
}

func (x *GenesisState) GetAckCount() uint64 {
	if x != nil {
		return x.AckCount
	}
	return 0
}

func (x *GenesisState) GetCheckpoints() []*types.Checkpoint {
	if x != nil {
		return x.Checkpoints
	}
	return nil
}

var File_heimdallv2_checkpoint_v1_genesis_proto protoreflect.FileDescriptor

var file_heimdallv2_checkpoint_v1_genesis_proto_rawDesc = []byte{
	0x0a, 0x26, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2f, 0x63, 0x68, 0x65,
	0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x73,
	0x69, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x18, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61,
	0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x2e,
	0x76, 0x31, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f,
	0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73,
	0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x11, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2f, 0x61, 0x6d, 0x69, 0x6e, 0x6f,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c,
	0x76, 0x32, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f,
	0x69, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbd, 0x02, 0x0a, 0x0c, 0x47, 0x65,
	0x6e, 0x65, 0x73, 0x69, 0x73, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3b, 0x0a, 0x06, 0x70, 0x61,
	0x72, 0x61, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x68, 0x65, 0x69,
	0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x73, 0x42, 0x09, 0xc8, 0xde, 0x1f, 0x00, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52,
	0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x12, 0x58, 0x0a, 0x13, 0x62, 0x75, 0x66, 0x66, 0x65,
	0x72, 0x65, 0x64, 0x5f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76,
	0x32, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69,
	0x6e, 0x74, 0x42, 0x09, 0xc8, 0xde, 0x1f, 0x01, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x12, 0x62,
	0x75, 0x66, 0x66, 0x65, 0x72, 0x65, 0x64, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x12, 0x27, 0x0a, 0x0d, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x6e, 0x6f, 0x5f, 0x61, 0x5f, 0x63,
	0x5f, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x42, 0x05, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52,
	0x09, 0x6c, 0x61, 0x73, 0x74, 0x4e, 0x6f, 0x41, 0x43, 0x4b, 0x12, 0x22, 0x0a, 0x09, 0x61, 0x63,
	0x6b, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x42, 0x05, 0xa8,
	0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x08, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x49,
	0x0a, 0x0b, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32,
	0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x42, 0x09, 0xc8, 0xde, 0x1f, 0x00, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x0b, 0x63, 0x68,
	0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x42, 0xfa, 0x01, 0x0a, 0x1c, 0x63, 0x6f,
	0x6d, 0x2e, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x63, 0x68, 0x65,
	0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x42, 0x0c, 0x47, 0x65, 0x6e, 0x65,
	0x73, 0x69, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x4a, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x30, 0x78, 0x50, 0x6f, 0x6c, 0x79, 0x67, 0x6f, 0x6e,
	0x2f, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x2d, 0x76, 0x32, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2f, 0x63, 0x68, 0x65, 0x63,
	0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70,
	0x6f, 0x69, 0x6e, 0x74, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x48, 0x43, 0x58, 0xaa, 0x02, 0x18, 0x48,
	0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70,
	0x6f, 0x69, 0x6e, 0x74, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x18, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61,
	0x6c, 0x6c, 0x76, 0x32, 0x5c, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x5c,
	0x56, 0x31, 0xe2, 0x02, 0x24, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x5c,
	0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50,
	0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1a, 0x48, 0x65, 0x69, 0x6d,
	0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x3a, 0x3a, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69,
	0x6e, 0x74, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_heimdallv2_checkpoint_v1_genesis_proto_rawDescOnce sync.Once
	file_heimdallv2_checkpoint_v1_genesis_proto_rawDescData = file_heimdallv2_checkpoint_v1_genesis_proto_rawDesc
)

func file_heimdallv2_checkpoint_v1_genesis_proto_rawDescGZIP() []byte {
	file_heimdallv2_checkpoint_v1_genesis_proto_rawDescOnce.Do(func() {
		file_heimdallv2_checkpoint_v1_genesis_proto_rawDescData = protoimpl.X.CompressGZIP(file_heimdallv2_checkpoint_v1_genesis_proto_rawDescData)
	})
	return file_heimdallv2_checkpoint_v1_genesis_proto_rawDescData
}

var file_heimdallv2_checkpoint_v1_genesis_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_heimdallv2_checkpoint_v1_genesis_proto_goTypes = []interface{}{
	(*GenesisState)(nil),     // 0: heimdallv2.checkpoint.v1.GenesisState
	(*types.Params)(nil),     // 1: heimdallv2.types.Params
	(*types.Checkpoint)(nil), // 2: heimdallv2.types.Checkpoint
}
var file_heimdallv2_checkpoint_v1_genesis_proto_depIdxs = []int32{
	1, // 0: heimdallv2.checkpoint.v1.GenesisState.params:type_name -> heimdallv2.types.Params
	2, // 1: heimdallv2.checkpoint.v1.GenesisState.buffered_checkpoint:type_name -> heimdallv2.types.Checkpoint
	2, // 2: heimdallv2.checkpoint.v1.GenesisState.checkpoints:type_name -> heimdallv2.types.Checkpoint
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_heimdallv2_checkpoint_v1_genesis_proto_init() }
func file_heimdallv2_checkpoint_v1_genesis_proto_init() {
	if File_heimdallv2_checkpoint_v1_genesis_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_heimdallv2_checkpoint_v1_genesis_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GenesisState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_heimdallv2_checkpoint_v1_genesis_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_heimdallv2_checkpoint_v1_genesis_proto_goTypes,
		DependencyIndexes: file_heimdallv2_checkpoint_v1_genesis_proto_depIdxs,
		MessageInfos:      file_heimdallv2_checkpoint_v1_genesis_proto_msgTypes,
	}.Build()
	File_heimdallv2_checkpoint_v1_genesis_proto = out.File
	file_heimdallv2_checkpoint_v1_genesis_proto_rawDesc = nil
	file_heimdallv2_checkpoint_v1_genesis_proto_goTypes = nil
	file_heimdallv2_checkpoint_v1_genesis_proto_depIdxs = nil
}
