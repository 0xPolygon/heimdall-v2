// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package types

import (
	_ "cosmossdk.io/api/amino"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_DividendAccount            protoreflect.MessageDescriptor
	fd_DividendAccount_user       protoreflect.FieldDescriptor
	fd_DividendAccount_fee_amount protoreflect.FieldDescriptor
)

func init() {
	file_heimdallv2_types_dividend_account_proto_init()
	md_DividendAccount = File_heimdallv2_types_dividend_account_proto.Messages().ByName("DividendAccount")
	fd_DividendAccount_user = md_DividendAccount.Fields().ByName("user")
	fd_DividendAccount_fee_amount = md_DividendAccount.Fields().ByName("fee_amount")
}

var _ protoreflect.Message = (*fastReflection_DividendAccount)(nil)

type fastReflection_DividendAccount DividendAccount

func (x *DividendAccount) ProtoReflect() protoreflect.Message {
	return (*fastReflection_DividendAccount)(x)
}

func (x *DividendAccount) slowProtoReflect() protoreflect.Message {
	mi := &file_heimdallv2_types_dividend_account_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_DividendAccount_messageType fastReflection_DividendAccount_messageType
var _ protoreflect.MessageType = fastReflection_DividendAccount_messageType{}

type fastReflection_DividendAccount_messageType struct{}

func (x fastReflection_DividendAccount_messageType) Zero() protoreflect.Message {
	return (*fastReflection_DividendAccount)(nil)
}
func (x fastReflection_DividendAccount_messageType) New() protoreflect.Message {
	return new(fastReflection_DividendAccount)
}
func (x fastReflection_DividendAccount_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_DividendAccount
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_DividendAccount) Descriptor() protoreflect.MessageDescriptor {
	return md_DividendAccount
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_DividendAccount) Type() protoreflect.MessageType {
	return _fastReflection_DividendAccount_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_DividendAccount) New() protoreflect.Message {
	return new(fastReflection_DividendAccount)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_DividendAccount) Interface() protoreflect.ProtoMessage {
	return (*DividendAccount)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_DividendAccount) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.User != "" {
		value := protoreflect.ValueOfString(x.User)
		if !f(fd_DividendAccount_user, value) {
			return
		}
	}
	if x.FeeAmount != "" {
		value := protoreflect.ValueOfString(x.FeeAmount)
		if !f(fd_DividendAccount_fee_amount, value) {
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
func (x *fastReflection_DividendAccount) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		return x.User != ""
	case "heimdallv2.types.DividendAccount.fee_amount":
		return x.FeeAmount != ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_DividendAccount) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		x.User = ""
	case "heimdallv2.types.DividendAccount.fee_amount":
		x.FeeAmount = ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_DividendAccount) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		value := x.User
		return protoreflect.ValueOfString(value)
	case "heimdallv2.types.DividendAccount.fee_amount":
		value := x.FeeAmount
		return protoreflect.ValueOfString(value)
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", descriptor.FullName()))
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
func (x *fastReflection_DividendAccount) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		x.User = value.Interface().(string)
	case "heimdallv2.types.DividendAccount.fee_amount":
		x.FeeAmount = value.Interface().(string)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", fd.FullName()))
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
func (x *fastReflection_DividendAccount) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		panic(fmt.Errorf("field user of message heimdallv2.types.DividendAccount is not mutable"))
	case "heimdallv2.types.DividendAccount.fee_amount":
		panic(fmt.Errorf("field fee_amount of message heimdallv2.types.DividendAccount is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_DividendAccount) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "heimdallv2.types.DividendAccount.user":
		return protoreflect.ValueOfString("")
	case "heimdallv2.types.DividendAccount.fee_amount":
		return protoreflect.ValueOfString("")
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: heimdallv2.types.DividendAccount"))
		}
		panic(fmt.Errorf("message heimdallv2.types.DividendAccount does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_DividendAccount) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in heimdallv2.types.DividendAccount", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_DividendAccount) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_DividendAccount) SetUnknown(fields protoreflect.RawFields) {
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
func (x *fastReflection_DividendAccount) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_DividendAccount) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*DividendAccount)
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
		l = len(x.User)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.FeeAmount)
		if l > 0 {
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
		x := input.Message.Interface().(*DividendAccount)
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
		if len(x.FeeAmount) > 0 {
			i -= len(x.FeeAmount)
			copy(dAtA[i:], x.FeeAmount)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.FeeAmount)))
			i--
			dAtA[i] = 0x12
		}
		if len(x.User) > 0 {
			i -= len(x.User)
			copy(dAtA[i:], x.User)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.User)))
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
		x := input.Message.Interface().(*DividendAccount)
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
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: DividendAccount: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: DividendAccount: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field User", wireType)
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
				x.User = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field FeeAmount", wireType)
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
				x.FeeAmount = string(dAtA[iNdEx:postIndex])
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
// source: heimdallv2/types/dividend_account.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// DividendAccount contains the burned fees
type DividendAccount struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User      string `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	FeeAmount string `protobuf:"bytes,2,opt,name=fee_amount,json=feeAmount,proto3" json:"fee_amount,omitempty"`
}

func (x *DividendAccount) Reset() {
	*x = DividendAccount{}
	if protoimpl.UnsafeEnabled {
		mi := &file_heimdallv2_types_dividend_account_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DividendAccount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DividendAccount) ProtoMessage() {}

// Deprecated: Use DividendAccount.ProtoReflect.Descriptor instead.
func (*DividendAccount) Descriptor() ([]byte, []int) {
	return file_heimdallv2_types_dividend_account_proto_rawDescGZIP(), []int{0}
}

func (x *DividendAccount) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *DividendAccount) GetFeeAmount() string {
	if x != nil {
		return x.FeeAmount
	}
	return ""
}

var File_heimdallv2_types_dividend_account_proto protoreflect.FileDescriptor

var file_heimdallv2_types_dividend_account_proto_rawDesc = []byte{
	0x0a, 0x27, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2f, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2f, 0x64, 0x69, 0x76, 0x69, 0x64, 0x65, 0x6e, 0x64, 0x5f, 0x61, 0x63, 0x63, 0x6f,
	0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10, 0x68, 0x65, 0x69, 0x6d, 0x64,
	0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x19, 0x63, 0x6f, 0x73,
	0x6d, 0x6f, 0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x11, 0x61, 0x6d,
	0x69, 0x6e, 0x6f, 0x2f, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x70, 0x0a, 0x0f, 0x44, 0x69, 0x76, 0x69, 0x64, 0x65, 0x6e, 0x64, 0x41, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x12, 0x31, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x42, 0x1d, 0xd2, 0xb4, 0x2d, 0x14, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x41, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52,
	0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x24, 0x0a, 0x0a, 0x66, 0x65, 0x65, 0x5f, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x05, 0xa8, 0xe7, 0xb0, 0x2a, 0x01,
	0x52, 0x09, 0x66, 0x65, 0x65, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x3a, 0x04, 0xe8, 0xa0, 0x1f,
	0x01, 0x42, 0xc4, 0x01, 0x0a, 0x14, 0x63, 0x6f, 0x6d, 0x2e, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61,
	0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x42, 0x14, 0x44, 0x69, 0x76, 0x69,
	0x64, 0x65, 0x6e, 0x64, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x30,
	0x78, 0x50, 0x6f, 0x6c, 0x79, 0x67, 0x6f, 0x6e, 0x2f, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c,
	0x6c, 0x2d, 0x76, 0x32, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x68, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c,
	0x6c, 0x76, 0x32, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0xa2, 0x02, 0x03, 0x48, 0x54, 0x58, 0xaa,
	0x02, 0x10, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x2e, 0x54, 0x79, 0x70,
	0x65, 0x73, 0xca, 0x02, 0x10, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76, 0x32, 0x5c,
	0x54, 0x79, 0x70, 0x65, 0x73, 0xe2, 0x02, 0x1c, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c,
	0x76, 0x32, 0x5c, 0x54, 0x79, 0x70, 0x65, 0x73, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x11, 0x48, 0x65, 0x69, 0x6d, 0x64, 0x61, 0x6c, 0x6c, 0x76,
	0x32, 0x3a, 0x3a, 0x54, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_heimdallv2_types_dividend_account_proto_rawDescOnce sync.Once
	file_heimdallv2_types_dividend_account_proto_rawDescData = file_heimdallv2_types_dividend_account_proto_rawDesc
)

func file_heimdallv2_types_dividend_account_proto_rawDescGZIP() []byte {
	file_heimdallv2_types_dividend_account_proto_rawDescOnce.Do(func() {
		file_heimdallv2_types_dividend_account_proto_rawDescData = protoimpl.X.CompressGZIP(file_heimdallv2_types_dividend_account_proto_rawDescData)
	})
	return file_heimdallv2_types_dividend_account_proto_rawDescData
}

var file_heimdallv2_types_dividend_account_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_heimdallv2_types_dividend_account_proto_goTypes = []interface{}{
	(*DividendAccount)(nil), // 0: heimdallv2.types.DividendAccount
}
var file_heimdallv2_types_dividend_account_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_heimdallv2_types_dividend_account_proto_init() }
func file_heimdallv2_types_dividend_account_proto_init() {
	if File_heimdallv2_types_dividend_account_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_heimdallv2_types_dividend_account_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DividendAccount); i {
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
			RawDescriptor: file_heimdallv2_types_dividend_account_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_heimdallv2_types_dividend_account_proto_goTypes,
		DependencyIndexes: file_heimdallv2_types_dividend_account_proto_depIdxs,
		MessageInfos:      file_heimdallv2_types_dividend_account_proto_msgTypes,
	}.Build()
	File_heimdallv2_types_dividend_account_proto = out.File
	file_heimdallv2_types_dividend_account_proto_rawDesc = nil
	file_heimdallv2_types_dividend_account_proto_goTypes = nil
	file_heimdallv2_types_dividend_account_proto_depIdxs = nil
}
