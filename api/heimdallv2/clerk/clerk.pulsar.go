// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package clerk

import (
	_ "cosmossdk.io/api/amino"
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/api/heimdallv2/types"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_EventRecord               protoreflect.MessageDescriptor
	fd_EventRecord_i_d           protoreflect.FieldDescriptor
	fd_EventRecord_contract      protoreflect.FieldDescriptor
	fd_EventRecord_data          protoreflect.FieldDescriptor
	fd_EventRecord_tx_hash       protoreflect.FieldDescriptor
	fd_EventRecord_log_index     protoreflect.FieldDescriptor
	fd_EventRecord_bor_chain_i_d protoreflect.FieldDescriptor
	fd_EventRecord_record_time   protoreflect.FieldDescriptor
)

func init() {
	file_heimdallv2_clerk_clerk_proto_init()
	md_EventRecord = File_heimdallv2_clerk_clerk_proto.Messages().ByName("EventRecord")
	fd_EventRecord_i_d = md_EventRecord.Fields().ByName("i_d")
	fd_EventRecord_contract = md_EventRecord.Fields().ByName("contract")
	fd_EventRecord_data = md_EventRecord.Fields().ByName("data")
	fd_EventRecord_tx_hash = md_EventRecord.Fields().ByName("tx_hash")
	fd_EventRecord_log_index = md_EventRecord.Fields().ByName("log_index")
	fd_EventRecord_bor_chain_i_d = md_EventRecord.Fields().ByName("bor_chain_i_d")
	fd_EventRecord_record_time = md_EventRecord.Fields().ByName("record_time")
}

var _ protoreflect.Message = (*fastReflection_EventRecord)(nil)

type fastReflection_EventRecord EventRecord

func (x *EventRecord) ProtoReflect() protoreflect.Message {
	return (*fastReflection_EventRecord)(x)
}

func (x *EventRecord) slowProtoReflect() protoreflect.Message {
	mi := &file_heimdallv2_clerk_clerk_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_EventRecord_messageType fastReflection_EventRecord_messageType
var _ protoreflect.MessageType = fastReflection_EventRecord_messageType{}

type fastReflection_EventRecord_messageType struct{}

func (x fastReflection_EventRecord_messageType) Zero() protoreflect.Message {
	return (*fastReflection_EventRecord)(nil)
}
func (x fastReflection_EventRecord_messageType) New() protoreflect.Message {
	return new(fastReflection_EventRecord)
}
func (x fastReflection_EventRecord_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_EventRecord
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_EventRecord) Descriptor() protoreflect.MessageDescriptor {
	return md_EventRecord
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_EventRecord) Type() protoreflect.MessageType {
	return _fastReflection_EventRecord_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_EventRecord) New() protoreflect.Message {
	return new(fastReflection_EventRecord)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_EventRecord) Interface() protoreflect.ProtoMessage {
	return (*EventRecord)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_EventRecord) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.ID != uint64(0) {
		value := protoreflect.ValueOfUint64(x.ID)
		if !f(fd_EventRecord_i_d, value) {
			return
		}
	}
	if x.Contract != "" {
		value := protoreflect.ValueOfString(x.Contract)
		if !f(fd_EventRecord_contract, value) {
			return
		}
	}
	if x.Data != nil {
		value := protoreflect.ValueOfMessage(x.Data.ProtoReflect())
		if !f(fd_EventRecord_data, value) {
			return
		}
	}
	if x.TxHash != "" {
		value := protoreflect.ValueOfString(x.TxHash)
		if !f(fd_EventRecord_tx_hash, value) {
			return
		}
	}
	if x.LogIndex != uint64(0) {
		value := protoreflect.ValueOfUint64(x.LogIndex)
		if !f(fd_EventRecord_log_index, value) {
			return
		}
	}
	if x.BorChainID != "" {
		value := protoreflect.ValueOfString(x.BorChainID)
		if !f(fd_EventRecord_bor_chain_i_d, value) {
			return
		}
	}
	if x.RecordTime != nil {
		value := protoreflect.ValueOfMessage(x.RecordTime.ProtoReflect())
		if !f(fd_EventRecord_record_time, value) {
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
func (x *fastReflection_EventRecord) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "heimdallv2.clerk.EventRecord.i_d":
		return x.ID != uint64(0)
	case "heimdallv2.clerk.EventRecord.contract":
		return x.Contract != ""
	case "heimdallv2.clerk.EventRecord.data":
		return x.Data != nil
	case "heimdallv2.clerk.EventRecord.tx_hash":
		return x.TxHash != ""
	case "heimdallv2.clerk.EventRecord.log_index":
		return x.LogIndex != uint64(0)
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		return x.BorChainID != ""
	case "heimdallv2.clerk.EventRecord.record_time":
		return x.RecordTime != nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_EventRecord) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "heimdallv2.clerk.EventRecord.i_d":
		x.ID = uint64(0)
	case "heimdallv2.clerk.EventRecord.contract":
		x.Contract = ""
	case "heimdallv2.clerk.EventRecord.data":
		x.Data = nil
	case "heimdallv2.clerk.EventRecord.tx_hash":
		x.TxHash = ""
	case "heimdallv2.clerk.EventRecord.log_index":
		x.LogIndex = uint64(0)
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		x.BorChainID = ""
	case "heimdallv2.clerk.EventRecord.record_time":
		x.RecordTime = nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_EventRecord) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "heimdallv2.clerk.EventRecord.i_d":
		value := x.ID
		return protoreflect.ValueOfUint64(value)
	case "heimdallv2.clerk.EventRecord.contract":
		value := x.Contract
		return protoreflect.ValueOfString(value)
	case "heimdallv2.clerk.EventRecord.data":
		value := x.Data
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "heimdallv2.clerk.EventRecord.tx_hash":
		value := x.TxHash
		return protoreflect.ValueOfString(value)
	case "heimdallv2.clerk.EventRecord.log_index":
		value := x.LogIndex
		return protoreflect.ValueOfUint64(value)
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		value := x.BorChainID
		return protoreflect.ValueOfString(value)
	case "heimdallv2.clerk.EventRecord.record_time":
		value := x.RecordTime
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", descriptor.FullName()))
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
func (x *fastReflection_EventRecord) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "heimdallv2.clerk.EventRecord.i_d":
		x.ID = value.Uint()
	case "heimdallv2.clerk.EventRecord.contract":
		x.Contract = value.Interface().(string)
	case "heimdallv2.clerk.EventRecord.data":
		x.Data = value.Message().Interface().(*types.HexBytes)
	case "heimdallv2.clerk.EventRecord.tx_hash":
		x.TxHash = value.Interface().(string)
	case "heimdallv2.clerk.EventRecord.log_index":
		x.LogIndex = value.Uint()
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		x.BorChainID = value.Interface().(string)
	case "heimdallv2.clerk.EventRecord.record_time":
		x.RecordTime = value.Message().Interface().(*timestamppb.Timestamp)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", fd.FullName()))
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
func (x *fastReflection_EventRecord) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.clerk.EventRecord.data":
		if x.Data == nil {
			x.Data = new(types.HexBytes)
		}
		return protoreflect.ValueOfMessage(x.Data.ProtoReflect())
	case "heimdallv2.clerk.EventRecord.record_time":
		if x.RecordTime == nil {
			x.RecordTime = new(timestamppb.Timestamp)
		}
		return protoreflect.ValueOfMessage(x.RecordTime.ProtoReflect())
	case "heimdallv2.clerk.EventRecord.i_d":
		panic(fmt.Errorf("field i_d of message heimdallv2.clerk.EventRecord is not mutable"))
	case "heimdallv2.clerk.EventRecord.contract":
		panic(fmt.Errorf("field contract of message heimdallv2.clerk.EventRecord is not mutable"))
	case "heimdallv2.clerk.EventRecord.tx_hash":
		panic(fmt.Errorf("field tx_hash of message heimdallv2.clerk.EventRecord is not mutable"))
	case "heimdallv2.clerk.EventRecord.log_index":
		panic(fmt.Errorf("field log_index of message heimdallv2.clerk.EventRecord is not mutable"))
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		panic(fmt.Errorf("field bor_chain_i_d of message heimdallv2.clerk.EventRecord is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_EventRecord) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.clerk.EventRecord.i_d":
		return protoreflect.ValueOfUint64(uint64(0))
	case "heimdallv2.clerk.EventRecord.contract":
		return protoreflect.ValueOfString("")
	case "heimdallv2.clerk.EventRecord.data":
		m := new(types.HexBytes)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "heimdallv2.clerk.EventRecord.tx_hash":
		return protoreflect.ValueOfString("")
	case "heimdallv2.clerk.EventRecord.log_index":
		return protoreflect.ValueOfUint64(uint64(0))
	case "heimdallv2.clerk.EventRecord.bor_chain_i_d":
		return protoreflect.ValueOfString("")
	case "heimdallv2.clerk.EventRecord.record_time":
		m := new(timestamppb.Timestamp)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.clerk.EventRecord"))
		}
		panic(fmt.Errorf("message heimdallv2.clerk.EventRecord does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_EventRecord) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in heimdallv2.clerk.EventRecord", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_EventRecord) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_EventRecord) SetUnknown(fields protoreflect.RawFields) {
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
func (x *fastReflection_EventRecord) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_EventRecord) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*EventRecord)
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
		if x.ID != 0 {
			n += 1 + runtime.Sov(uint64(x.ID))
		}
		l = len(x.Contract)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.Data != nil {
			l = options.Size(x.Data)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.TxHash)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.LogIndex != 0 {
			n += 1 + runtime.Sov(uint64(x.LogIndex))
		}
		l = len(x.BorChainID)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.RecordTime != nil {
			l = options.Size(x.RecordTime)
			n += 1 + l + runtime.Sov(uint64(l))
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
		x := input.Message.Interface().(*EventRecord)
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
		if x.RecordTime != nil {
			encoded, err := options.Marshal(x.RecordTime)
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
			dAtA[i] = 0x3a
		}
		if len(x.BorChainID) > 0 {
			i -= len(x.BorChainID)
			copy(dAtA[i:], x.BorChainID)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.BorChainID)))
			i--
			dAtA[i] = 0x32
		}
		if x.LogIndex != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.LogIndex))
			i--
			dAtA[i] = 0x28
		}
		if len(x.TxHash) > 0 {
			i -= len(x.TxHash)
			copy(dAtA[i:], x.TxHash)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.TxHash)))
			i--
			dAtA[i] = 0x22
		}
		if x.Data != nil {
			encoded, err := options.Marshal(x.Data)
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
			dAtA[i] = 0x1a
		}
		if len(x.Contract) > 0 {
			i -= len(x.Contract)
			copy(dAtA[i:], x.Contract)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.Contract)))
			i--
			dAtA[i] = 0x12
		}
		if x.ID != 0 {
			i = runtime.EncodeVarint(dAtA, i, uint64(x.ID))
			i--
			dAtA[i] = 0x8
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
		x := input.Message.Interface().(*EventRecord)
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
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: EventRecord: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: EventRecord: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
				}
				x.ID = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.ID |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Contract", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Contract = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 3:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
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
				if x.Data == nil {
					x.Data = &types.HexBytes{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.Data); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 4:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.TxHash = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 5:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field LogIndex", wireType)
				}
				x.LogIndex = 0
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					x.LogIndex |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
			case 6:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field BorChainID", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.BorChainID = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 7:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field RecordTime", wireType)
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
				if x.RecordTime == nil {
					x.RecordTime = &timestamppb.Timestamp{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.RecordTime); err != nil {
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
// source: heimdallv2/clerk/clerk.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EventRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID         uint64                 `protobuf:"varint,1,opt,name=i_d,json=iD,proto3" json:"i_d,omitempty"`
	Contract   string                 `protobuf:"bytes,2,opt,name=contract,proto3" json:"contract,omitempty"`
	Data       *types.HexBytes        `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	TxHash     string                 `protobuf:"bytes,4,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex   uint64                 `protobuf:"varint,5,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BorChainID string                 `protobuf:"bytes,6,opt,name=bor_chain_i_d,json=borChainID,proto3" json:"bor_chain_i_d,omitempty"`
	RecordTime *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=record_time,json=recordTime,proto3" json:"record_time,omitempty"`
}

func (x *EventRecord) Reset() {
	*x = EventRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_heimdallv2_clerk_clerk_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventRecord) ProtoMessage() {}

// Deprecated: Use EventRecord.ProtoReflect.Descriptor instead.
func (*EventRecord) Descriptor() ([]byte, []int) {
	return file_heimdallv2_clerk_clerk_proto_rawDescGZIP(), []int{0}
}

func (x *EventRecord) GetID() uint64 {
	if x != nil {
		return x.ID
	}
	return 0
}

func (x *EventRecord) GetContract() string {
	if x != nil {
		return x.Contract
	}
	return ""
}

func (x *EventRecord) GetData() *types.HexBytes {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *EventRecord) GetTxHash() string {
	if x != nil {
		return x.TxHash
	}
	return ""
}

func (x *EventRecord) GetLogIndex() uint64 {
	if x != nil {
		return x.LogIndex
	}
	return 0
}

func (x *EventRecord) GetBorChainID() string {
	if x != nil {
		return x.BorChainID
	}
	return ""
}

func (x *EventRecord) GetRecordTime() *timestamppb.Timestamp {
	if x != nil {
		return x.RecordTime
	}
	return nil
}

var File_heimdallv2_clerk_clerk_proto protoreflect.FileDescriptor

var file_heimdallv2_clerk_clerk_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2f, 0x63, 0x6c, 0x65,
	0x72, 0x6b, 0x2f, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10,
	0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x63, 0x6c, 0x65, 0x72, 0x6b,
	0x1a, 0x11, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2f, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14,
	0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76,
	0x32, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x68, 0x61, 0x73, 0x68, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0xb9, 0x02, 0x0a, 0x0b, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x63, 0x6f,
	0x72, 0x64, 0x12, 0x0f, 0x0a, 0x03, 0x69, 0x5f, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x02, 0x69, 0x44, 0x12, 0x39, 0x0a, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1d, 0xd2, 0xb4, 0x2d, 0x14, 0x63, 0x6f, 0x73, 0x6d, 0x6f,
	0x73, 0x2e, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0xa8,
	0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x12, 0x34,
	0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x68,
	0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e,
	0x48, 0x65, 0x78, 0x42, 0x79, 0x74, 0x65, 0x73, 0x42, 0x04, 0xc8, 0xde, 0x1f, 0x00, 0x52, 0x04,
	0x64, 0x61, 0x74, 0x61, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x78, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74, 0x78, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1b, 0x0a,
	0x09, 0x6c, 0x6f, 0x67, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x08, 0x6c, 0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x21, 0x0a, 0x0d, 0x62, 0x6f,
	0x72, 0x5f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5f, 0x69, 0x5f, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x62, 0x6f, 0x72, 0x43, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x12, 0x45, 0x0a,
	0x0b, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x08,
	0xc8, 0xde, 0x1f, 0x00, 0x90, 0xdf, 0x1f, 0x01, 0x52, 0x0a, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64,
	0x54, 0x69, 0x6d, 0x65, 0x3a, 0x08, 0x88, 0xa0, 0x1f, 0x00, 0xe8, 0xa0, 0x1f, 0x00, 0x42, 0xba,
	0x01, 0x0a, 0x14, 0x63, 0x6f, 0x6d, 0x2e, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76,
	0x32, 0x2e, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0x42, 0x0a, 0x43, 0x6c, 0x65, 0x72, 0x6b, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x30, 0x78, 0x50, 0x6f, 0x6c, 0x79, 0x67, 0x6f, 0x6e, 0x2f, 0x68, 0x65, 0x69, 0x6d,
	0x64, 0x61, 0x6c, 0x6c, 0x2d, 0x76, 0x32, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x68, 0x65, 0x69, 0x6d,
	0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2f, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0xa2, 0x02, 0x03, 0x48,
	0x43, 0x58, 0xaa, 0x02, 0x10, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e,
	0x43, 0x6c, 0x65, 0x72, 0x6b, 0xca, 0x02, 0x10, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c,
	0x76, 0x32, 0x5c, 0x43, 0x6c, 0x65, 0x72, 0x6b, 0xe2, 0x02, 0x1c, 0x48, 0x65, 0x69, 0x6d, 0x64,
	0x61, 0x6c, 0x6c, 0x76, 0x32, 0x5c, 0x43, 0x6c, 0x65, 0x72, 0x6b, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x11, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61,
	0x6c, 0x6c, 0x76, 0x32, 0x3a, 0x3a, 0x43, 0x6c, 0x65, 0x72, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_heimdallv2_clerk_clerk_proto_rawDescOnce sync.Once
	file_heimdallv2_clerk_clerk_proto_rawDescData = file_heimdallv2_clerk_clerk_proto_rawDesc
)

func file_heimdallv2_clerk_clerk_proto_rawDescGZIP() []byte {
	file_heimdallv2_clerk_clerk_proto_rawDescOnce.Do(func() {
		file_heimdallv2_clerk_clerk_proto_rawDescData = protoimpl.X.CompressGZIP(file_heimdallv2_clerk_clerk_proto_rawDescData)
	})
	return file_heimdallv2_clerk_clerk_proto_rawDescData
}

var file_heimdallv2_clerk_clerk_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_heimdallv2_clerk_clerk_proto_goTypes = []interface{}{
	(*EventRecord)(nil),           // 0: heimdallv2.clerk.EventRecord
	(*types.HexBytes)(nil),        // 1: heimdallv2.types.HexBytes
	(*timestamppb.Timestamp)(nil), // 2: google.protobuf.Timestamp
}
var file_heimdallv2_clerk_clerk_proto_depIdxs = []int32{
	1, // 0: heimdallv2.clerk.EventRecord.data:type_name -> heimdallv2.types.HexBytes
	2, // 1: heimdallv2.clerk.EventRecord.record_time:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_heimdallv2_clerk_clerk_proto_init() }
func file_heimdallv2_clerk_clerk_proto_init() {
	if File_heimdallv2_clerk_clerk_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_heimdallv2_clerk_clerk_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventRecord); i {
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
			RawDescriptor: file_heimdallv2_clerk_clerk_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_heimdallv2_clerk_clerk_proto_goTypes,
		DependencyIndexes: file_heimdallv2_clerk_clerk_proto_depIdxs,
		MessageInfos:      file_heimdallv2_clerk_clerk_proto_msgTypes,
	}.Build()
	File_heimdallv2_clerk_clerk_proto = out.File
	file_heimdallv2_clerk_clerk_proto_rawDesc = nil
	file_heimdallv2_clerk_clerk_proto_goTypes = nil
	file_heimdallv2_clerk_clerk_proto_depIdxs = nil
}
