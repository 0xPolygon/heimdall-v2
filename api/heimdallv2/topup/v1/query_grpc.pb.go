// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: heimdallv2/topup/v1/query.proto

package topupv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Query_IsTopupTxOld_FullMethodName                = "/heimdallv2.topup.v1.Query/IsTopupTxOld"
	Query_GetTopupTxSequence_FullMethodName          = "/heimdallv2.topup.v1.Query/GetTopupTxSequence"
	Query_GetDividendAccountByAddress_FullMethodName = "/heimdallv2.topup.v1.Query/GetDividendAccountByAddress"
	Query_GetDividendAccountRootHash_FullMethodName  = "/heimdallv2.topup.v1.Query/GetDividendAccountRootHash"
	Query_VerifyAccountProof_FullMethodName          = "/heimdallv2.topup.v1.Query/VerifyAccountProof"
	Query_GetAccountProof_FullMethodName             = "/heimdallv2.topup.v1.Query/GetAccountProof"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// IsTopupTxOld queries for a specific topup tx to check its status (old
	// means already submitted)
	IsTopupTxOld(ctx context.Context, in *QueryTopupSequenceRequest, opts ...grpc.CallOption) (*QueryIsTopupTxOldResponse, error)
	// GetTopupTxSequence queries for a specific topup tx and returns its sequence
	GetTopupTxSequence(ctx context.Context, in *QueryTopupSequenceRequest, opts ...grpc.CallOption) (*QueryTopupSequenceResponse, error)
	// GetDividendAccountByAddress queries for a specific DividendAccount by its
	// address
	GetDividendAccountByAddress(ctx context.Context, in *QueryDividendAccountRequest, opts ...grpc.CallOption) (*QueryDividendAccountResponse, error)
	// GetDividendAccountRootHash queries for the dividend account of the genesis
	// root hash
	GetDividendAccountRootHash(ctx context.Context, in *QueryDividendAccountRootHashRequest, opts ...grpc.CallOption) (*QueryDividendAccountRootHashResponse, error)
	// VerifyAccountProof queries for the proof of an account given its address
	VerifyAccountProof(ctx context.Context, in *QueryVerifyAccountProofRequest, opts ...grpc.CallOption) (*QueryVerifyAccountProofResponse, error)
	// GetAccountProof queries for the account proof of a given address
	GetAccountProof(ctx context.Context, in *QueryAccountProofRequest, opts ...grpc.CallOption) (*QueryAccountProofResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) IsTopupTxOld(ctx context.Context, in *QueryTopupSequenceRequest, opts ...grpc.CallOption) (*QueryIsTopupTxOldResponse, error) {
	out := new(QueryIsTopupTxOldResponse)
	err := c.cc.Invoke(ctx, Query_IsTopupTxOld_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetTopupTxSequence(ctx context.Context, in *QueryTopupSequenceRequest, opts ...grpc.CallOption) (*QueryTopupSequenceResponse, error) {
	out := new(QueryTopupSequenceResponse)
	err := c.cc.Invoke(ctx, Query_GetTopupTxSequence_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetDividendAccountByAddress(ctx context.Context, in *QueryDividendAccountRequest, opts ...grpc.CallOption) (*QueryDividendAccountResponse, error) {
	out := new(QueryDividendAccountResponse)
	err := c.cc.Invoke(ctx, Query_GetDividendAccountByAddress_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetDividendAccountRootHash(ctx context.Context, in *QueryDividendAccountRootHashRequest, opts ...grpc.CallOption) (*QueryDividendAccountRootHashResponse, error) {
	out := new(QueryDividendAccountRootHashResponse)
	err := c.cc.Invoke(ctx, Query_GetDividendAccountRootHash_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) VerifyAccountProof(ctx context.Context, in *QueryVerifyAccountProofRequest, opts ...grpc.CallOption) (*QueryVerifyAccountProofResponse, error) {
	out := new(QueryVerifyAccountProofResponse)
	err := c.cc.Invoke(ctx, Query_VerifyAccountProof_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetAccountProof(ctx context.Context, in *QueryAccountProofRequest, opts ...grpc.CallOption) (*QueryAccountProofResponse, error) {
	out := new(QueryAccountProofResponse)
	err := c.cc.Invoke(ctx, Query_GetAccountProof_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// IsTopupTxOld queries for a specific topup tx to check its status (old
	// means already submitted)
	IsTopupTxOld(context.Context, *QueryTopupSequenceRequest) (*QueryIsTopupTxOldResponse, error)
	// GetTopupTxSequence queries for a specific topup tx and returns its sequence
	GetTopupTxSequence(context.Context, *QueryTopupSequenceRequest) (*QueryTopupSequenceResponse, error)
	// GetDividendAccountByAddress queries for a specific DividendAccount by its
	// address
	GetDividendAccountByAddress(context.Context, *QueryDividendAccountRequest) (*QueryDividendAccountResponse, error)
	// GetDividendAccountRootHash queries for the dividend account of the genesis
	// root hash
	GetDividendAccountRootHash(context.Context, *QueryDividendAccountRootHashRequest) (*QueryDividendAccountRootHashResponse, error)
	// VerifyAccountProof queries for the proof of an account given its address
	VerifyAccountProof(context.Context, *QueryVerifyAccountProofRequest) (*QueryVerifyAccountProofResponse, error)
	// GetAccountProof queries for the account proof of a given address
	GetAccountProof(context.Context, *QueryAccountProofRequest) (*QueryAccountProofResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) IsTopupTxOld(context.Context, *QueryTopupSequenceRequest) (*QueryIsTopupTxOldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IsTopupTxOld not implemented")
}
func (UnimplementedQueryServer) GetTopupTxSequence(context.Context, *QueryTopupSequenceRequest) (*QueryTopupSequenceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTopupTxSequence not implemented")
}
func (UnimplementedQueryServer) GetDividendAccountByAddress(context.Context, *QueryDividendAccountRequest) (*QueryDividendAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDividendAccountByAddress not implemented")
}
func (UnimplementedQueryServer) GetDividendAccountRootHash(context.Context, *QueryDividendAccountRootHashRequest) (*QueryDividendAccountRootHashResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDividendAccountRootHash not implemented")
}
func (UnimplementedQueryServer) VerifyAccountProof(context.Context, *QueryVerifyAccountProofRequest) (*QueryVerifyAccountProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyAccountProof not implemented")
}
func (UnimplementedQueryServer) GetAccountProof(context.Context, *QueryAccountProofRequest) (*QueryAccountProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAccountProof not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_IsTopupTxOld_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTopupSequenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).IsTopupTxOld(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_IsTopupTxOld_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).IsTopupTxOld(ctx, req.(*QueryTopupSequenceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetTopupTxSequence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTopupSequenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetTopupTxSequence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetTopupTxSequence_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetTopupTxSequence(ctx, req.(*QueryTopupSequenceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetDividendAccountByAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDividendAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetDividendAccountByAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetDividendAccountByAddress_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetDividendAccountByAddress(ctx, req.(*QueryDividendAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetDividendAccountRootHash_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDividendAccountRootHashRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetDividendAccountRootHash(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetDividendAccountRootHash_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetDividendAccountRootHash(ctx, req.(*QueryDividendAccountRootHashRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_VerifyAccountProof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryVerifyAccountProofRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).VerifyAccountProof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_VerifyAccountProof_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).VerifyAccountProof(ctx, req.(*QueryVerifyAccountProofRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetAccountProof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAccountProofRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetAccountProof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetAccountProof_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetAccountProof(ctx, req.(*QueryAccountProofRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "heimdallv2.topup.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "IsTopupTxOld",
			Handler:    _Query_IsTopupTxOld_Handler,
		},
		{
			MethodName: "GetTopupTxSequence",
			Handler:    _Query_GetTopupTxSequence_Handler,
		},
		{
			MethodName: "GetDividendAccountByAddress",
			Handler:    _Query_GetDividendAccountByAddress_Handler,
		},
		{
			MethodName: "GetDividendAccountRootHash",
			Handler:    _Query_GetDividendAccountRootHash_Handler,
		},
		{
			MethodName: "VerifyAccountProof",
			Handler:    _Query_VerifyAccountProof_Handler,
		},
		{
			MethodName: "GetAccountProof",
			Handler:    _Query_GetAccountProof_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "heimdallv2/topup/v1/query.proto",
}
