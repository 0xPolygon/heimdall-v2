// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/types/rest/rest.proto

package rest

import (
	fmt "fmt"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type ResponseWithHeight struct {
	Height int64  `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	Result []byte `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
}

func (m *ResponseWithHeight) Reset()         { *m = ResponseWithHeight{} }
func (m *ResponseWithHeight) String() string { return proto.CompactTextString(m) }
func (*ResponseWithHeight) ProtoMessage()    {}
func (*ResponseWithHeight) Descriptor() ([]byte, []int) {
	return fileDescriptor_b3c40bf456573fa6, []int{0}
}
func (m *ResponseWithHeight) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ResponseWithHeight) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ResponseWithHeight.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ResponseWithHeight) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ResponseWithHeight.Merge(m, src)
}
func (m *ResponseWithHeight) XXX_Size() int {
	return m.Size()
}
func (m *ResponseWithHeight) XXX_DiscardUnknown() {
	xxx_messageInfo_ResponseWithHeight.DiscardUnknown(m)
}

var xxx_messageInfo_ResponseWithHeight proto.InternalMessageInfo

func (m *ResponseWithHeight) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *ResponseWithHeight) GetResult() []byte {
	if m != nil {
		return m.Result
	}
	return nil
}

type GasEstimateResponse struct {
	GasEstimate uint64 `protobuf:"varint,1,opt,name=gas_estimate,json=gasEstimate,proto3" json:"gas_estimate,omitempty"`
}

func (m *GasEstimateResponse) Reset()         { *m = GasEstimateResponse{} }
func (m *GasEstimateResponse) String() string { return proto.CompactTextString(m) }
func (*GasEstimateResponse) ProtoMessage()    {}
func (*GasEstimateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_b3c40bf456573fa6, []int{1}
}
func (m *GasEstimateResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GasEstimateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GasEstimateResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GasEstimateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GasEstimateResponse.Merge(m, src)
}
func (m *GasEstimateResponse) XXX_Size() int {
	return m.Size()
}
func (m *GasEstimateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GasEstimateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GasEstimateResponse proto.InternalMessageInfo

func (m *GasEstimateResponse) GetGasEstimate() uint64 {
	if m != nil {
		return m.GasEstimate
	}
	return 0
}

func init() {
	proto.RegisterType((*ResponseWithHeight)(nil), "heimdallv2.types.rest.ResponseWithHeight")
	proto.RegisterType((*GasEstimateResponse)(nil), "heimdallv2.types.rest.GasEstimateResponse")
}

func init() { proto.RegisterFile("heimdallv2/types/rest/rest.proto", fileDescriptor_b3c40bf456573fa6) }

var fileDescriptor_b3c40bf456573fa6 = []byte{
	// 216 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xc8, 0x48, 0xcd, 0xcc,
	0x4d, 0x49, 0xcc, 0xc9, 0x29, 0x33, 0xd2, 0x2f, 0xa9, 0x2c, 0x48, 0x2d, 0xd6, 0x2f, 0x4a, 0x2d,
	0x2e, 0x01, 0x13, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0xa2, 0x08, 0x15, 0x7a, 0x60, 0x15,
	0x7a, 0x20, 0x49, 0x25, 0x17, 0x2e, 0xa1, 0xa0, 0xd4, 0xe2, 0x82, 0xfc, 0xbc, 0xe2, 0xd4, 0xf0,
	0xcc, 0x92, 0x0c, 0x8f, 0xd4, 0xcc, 0xf4, 0x8c, 0x12, 0x21, 0x31, 0x2e, 0xb6, 0x0c, 0x30, 0x4b,
	0x82, 0x51, 0x81, 0x51, 0x83, 0x39, 0x08, 0xca, 0x03, 0x89, 0x17, 0xa5, 0x16, 0x97, 0xe6, 0x94,
	0x48, 0x30, 0x29, 0x30, 0x6a, 0xf0, 0x04, 0x41, 0x79, 0x4a, 0x16, 0x5c, 0xc2, 0xee, 0x89, 0xc5,
	0xae, 0xc5, 0x25, 0x99, 0xb9, 0x89, 0x25, 0xa9, 0x30, 0x03, 0x85, 0x14, 0xb9, 0x78, 0xd2, 0x13,
	0x8b, 0xe3, 0x53, 0xa1, 0xe2, 0x60, 0xc3, 0x58, 0x82, 0xb8, 0xd3, 0x11, 0x4a, 0x9d, 0x5c, 0x4f,
	0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18,
	0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21, 0x4a, 0x3b, 0x3d, 0xb3, 0x24, 0xa3, 0x34,
	0x49, 0x2f, 0x39, 0x3f, 0x57, 0xdf, 0xa0, 0x22, 0x20, 0x3f, 0xa7, 0x32, 0x3d, 0x3f, 0x4f, 0x1f,
	0xe6, 0x0b, 0x5d, 0x14, 0x8f, 0x26, 0xb1, 0x81, 0x3d, 0x69, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff,
	0x76, 0x0d, 0xbb, 0x44, 0x08, 0x01, 0x00, 0x00,
}

func (m *ResponseWithHeight) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ResponseWithHeight) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ResponseWithHeight) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Result) > 0 {
		i -= len(m.Result)
		copy(dAtA[i:], m.Result)
		i = encodeVarintRest(dAtA, i, uint64(len(m.Result)))
		i--
		dAtA[i] = 0x12
	}
	if m.Height != 0 {
		i = encodeVarintRest(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *GasEstimateResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GasEstimateResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GasEstimateResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.GasEstimate != 0 {
		i = encodeVarintRest(dAtA, i, uint64(m.GasEstimate))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintRest(dAtA []byte, offset int, v uint64) int {
	offset -= sovRest(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ResponseWithHeight) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Height != 0 {
		n += 1 + sovRest(uint64(m.Height))
	}
	l = len(m.Result)
	if l > 0 {
		n += 1 + l + sovRest(uint64(l))
	}
	return n
}

func (m *GasEstimateResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.GasEstimate != 0 {
		n += 1 + sovRest(uint64(m.GasEstimate))
	}
	return n
}

func sovRest(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozRest(x uint64) (n int) {
	return sovRest(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ResponseWithHeight) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRest
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
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
			return fmt.Errorf("proto: ResponseWithHeight: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ResponseWithHeight: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Result", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRest
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthRest
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Result = append(m.Result[:0], dAtA[iNdEx:postIndex]...)
			if m.Result == nil {
				m.Result = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRest(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRest
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *GasEstimateResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRest
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
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
			return fmt.Errorf("proto: GasEstimateResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GasEstimateResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasEstimate", wireType)
			}
			m.GasEstimate = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRest
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.GasEstimate |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipRest(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRest
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipRest(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRest
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRest
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRest
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthRest
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupRest
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthRest
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthRest        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRest          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupRest = fmt.Errorf("proto: unexpected end of group")
)
