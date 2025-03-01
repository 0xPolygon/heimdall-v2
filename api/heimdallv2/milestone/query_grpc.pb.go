// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: heimdallv2/milestone/query.proto

package milestone

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
	Query_GetMilestoneParams_FullMethodName          = "/heimdallv2.milestone.Query/GetMilestoneParams"
	Query_GetMilestoneCount_FullMethodName           = "/heimdallv2.milestone.Query/GetMilestoneCount"
	Query_GetLatestMilestone_FullMethodName          = "/heimdallv2.milestone.Query/GetLatestMilestone"
	Query_GetLatestNoAckMilestone_FullMethodName     = "/heimdallv2.milestone.Query/GetLatestNoAckMilestone"
	Query_GetMilestoneByNumber_FullMethodName        = "/heimdallv2.milestone.Query/GetMilestoneByNumber"
	Query_GetNoAckMilestoneById_FullMethodName       = "/heimdallv2.milestone.Query/GetNoAckMilestoneById"
	Query_GetMilestoneProposerByTimes_FullMethodName = "/heimdallv2.milestone.Query/GetMilestoneProposerByTimes"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// GetMilestoneParams queries for the x/milestone parameters
	GetMilestoneParams(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// GetMilestoneCount queries for the milestone count
	GetMilestoneCount(ctx context.Context, in *QueryCountRequest, opts ...grpc.CallOption) (*QueryCountResponse, error)
	// GetLatestMilestone queries for the latest milestone
	GetLatestMilestone(ctx context.Context, in *QueryLatestMilestoneRequest, opts ...grpc.CallOption) (*QueryLatestMilestoneResponse, error)
	// GetLatestNoAckMilestone query for the LatestNoAck
	GetLatestNoAckMilestone(ctx context.Context, in *QueryLatestNoAckMilestoneRequest, opts ...grpc.CallOption) (*QueryLatestNoAckMilestoneResponse, error)
	// GetMilestoneByNumber queries for the milestone based on the number
	GetMilestoneByNumber(ctx context.Context, in *QueryMilestoneRequest, opts ...grpc.CallOption) (*QueryMilestoneResponse, error)
	// GetNoAckMilestoneById query for the no-ack by id
	GetNoAckMilestoneById(ctx context.Context, in *QueryNoAckMilestoneByIDRequest, opts ...grpc.CallOption) (*QueryNoAckMilestoneByIDResponse, error)
	// GetMilestoneProposerByTimes queries for the milestone proposer
	GetMilestoneProposerByTimes(ctx context.Context, in *QueryMilestoneProposerRequest, opts ...grpc.CallOption) (*QueryMilestoneProposerResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) GetMilestoneParams(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, Query_GetMilestoneParams_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetMilestoneCount(ctx context.Context, in *QueryCountRequest, opts ...grpc.CallOption) (*QueryCountResponse, error) {
	out := new(QueryCountResponse)
	err := c.cc.Invoke(ctx, Query_GetMilestoneCount_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetLatestMilestone(ctx context.Context, in *QueryLatestMilestoneRequest, opts ...grpc.CallOption) (*QueryLatestMilestoneResponse, error) {
	out := new(QueryLatestMilestoneResponse)
	err := c.cc.Invoke(ctx, Query_GetLatestMilestone_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetLatestNoAckMilestone(ctx context.Context, in *QueryLatestNoAckMilestoneRequest, opts ...grpc.CallOption) (*QueryLatestNoAckMilestoneResponse, error) {
	out := new(QueryLatestNoAckMilestoneResponse)
	err := c.cc.Invoke(ctx, Query_GetLatestNoAckMilestone_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetMilestoneByNumber(ctx context.Context, in *QueryMilestoneRequest, opts ...grpc.CallOption) (*QueryMilestoneResponse, error) {
	out := new(QueryMilestoneResponse)
	err := c.cc.Invoke(ctx, Query_GetMilestoneByNumber_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetNoAckMilestoneById(ctx context.Context, in *QueryNoAckMilestoneByIDRequest, opts ...grpc.CallOption) (*QueryNoAckMilestoneByIDResponse, error) {
	out := new(QueryNoAckMilestoneByIDResponse)
	err := c.cc.Invoke(ctx, Query_GetNoAckMilestoneById_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetMilestoneProposerByTimes(ctx context.Context, in *QueryMilestoneProposerRequest, opts ...grpc.CallOption) (*QueryMilestoneProposerResponse, error) {
	out := new(QueryMilestoneProposerResponse)
	err := c.cc.Invoke(ctx, Query_GetMilestoneProposerByTimes_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// GetMilestoneParams queries for the x/milestone parameters
	GetMilestoneParams(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	// GetMilestoneCount queries for the milestone count
	GetMilestoneCount(context.Context, *QueryCountRequest) (*QueryCountResponse, error)
	// GetLatestMilestone queries for the latest milestone
	GetLatestMilestone(context.Context, *QueryLatestMilestoneRequest) (*QueryLatestMilestoneResponse, error)
	// GetLatestNoAckMilestone query for the LatestNoAck
	GetLatestNoAckMilestone(context.Context, *QueryLatestNoAckMilestoneRequest) (*QueryLatestNoAckMilestoneResponse, error)
	// GetMilestoneByNumber queries for the milestone based on the number
	GetMilestoneByNumber(context.Context, *QueryMilestoneRequest) (*QueryMilestoneResponse, error)
	// GetNoAckMilestoneById query for the no-ack by id
	GetNoAckMilestoneById(context.Context, *QueryNoAckMilestoneByIDRequest) (*QueryNoAckMilestoneByIDResponse, error)
	// GetMilestoneProposerByTimes queries for the milestone proposer
	GetMilestoneProposerByTimes(context.Context, *QueryMilestoneProposerRequest) (*QueryMilestoneProposerResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) GetMilestoneParams(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMilestoneParams not implemented")
}
func (UnimplementedQueryServer) GetMilestoneCount(context.Context, *QueryCountRequest) (*QueryCountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMilestoneCount not implemented")
}
func (UnimplementedQueryServer) GetLatestMilestone(context.Context, *QueryLatestMilestoneRequest) (*QueryLatestMilestoneResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestMilestone not implemented")
}
func (UnimplementedQueryServer) GetLatestNoAckMilestone(context.Context, *QueryLatestNoAckMilestoneRequest) (*QueryLatestNoAckMilestoneResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestNoAckMilestone not implemented")
}
func (UnimplementedQueryServer) GetMilestoneByNumber(context.Context, *QueryMilestoneRequest) (*QueryMilestoneResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMilestoneByNumber not implemented")
}
func (UnimplementedQueryServer) GetNoAckMilestoneById(context.Context, *QueryNoAckMilestoneByIDRequest) (*QueryNoAckMilestoneByIDResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNoAckMilestoneById not implemented")
}
func (UnimplementedQueryServer) GetMilestoneProposerByTimes(context.Context, *QueryMilestoneProposerRequest) (*QueryMilestoneProposerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMilestoneProposerByTimes not implemented")
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

func _Query_GetMilestoneParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetMilestoneParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetMilestoneParams_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetMilestoneParams(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetMilestoneCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetMilestoneCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetMilestoneCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetMilestoneCount(ctx, req.(*QueryCountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetLatestMilestone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLatestMilestoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetLatestMilestone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetLatestMilestone_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetLatestMilestone(ctx, req.(*QueryLatestMilestoneRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetLatestNoAckMilestone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLatestNoAckMilestoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetLatestNoAckMilestone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetLatestNoAckMilestone_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetLatestNoAckMilestone(ctx, req.(*QueryLatestNoAckMilestoneRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetMilestoneByNumber_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryMilestoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetMilestoneByNumber(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetMilestoneByNumber_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetMilestoneByNumber(ctx, req.(*QueryMilestoneRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetNoAckMilestoneById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryNoAckMilestoneByIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetNoAckMilestoneById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetNoAckMilestoneById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetNoAckMilestoneById(ctx, req.(*QueryNoAckMilestoneByIDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetMilestoneProposerByTimes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryMilestoneProposerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetMilestoneProposerByTimes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_GetMilestoneProposerByTimes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetMilestoneProposerByTimes(ctx, req.(*QueryMilestoneProposerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "heimdallv2.milestone.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetMilestoneParams",
			Handler:    _Query_GetMilestoneParams_Handler,
		},
		{
			MethodName: "GetMilestoneCount",
			Handler:    _Query_GetMilestoneCount_Handler,
		},
		{
			MethodName: "GetLatestMilestone",
			Handler:    _Query_GetLatestMilestone_Handler,
		},
		{
			MethodName: "GetLatestNoAckMilestone",
			Handler:    _Query_GetLatestNoAckMilestone_Handler,
		},
		{
			MethodName: "GetMilestoneByNumber",
			Handler:    _Query_GetMilestoneByNumber_Handler,
		},
		{
			MethodName: "GetNoAckMilestoneById",
			Handler:    _Query_GetNoAckMilestoneById_Handler,
		},
		{
			MethodName: "GetMilestoneProposerByTimes",
			Handler:    _Query_GetMilestoneProposerByTimes_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "heimdallv2/milestone/query.proto",
}
