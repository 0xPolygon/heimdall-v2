// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: cosmos/staking/v1beta1/tx.proto

package stakingv1beta1

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
	Msg_JoinValidator_FullMethodName = "/cosmos.staking.v1beta1.Msg/JoinValidator"
	Msg_StakeUpdate_FullMethodName   = "/cosmos.staking.v1beta1.Msg/StakeUpdate"
	Msg_SignerUpdate_FullMethodName  = "/cosmos.staking.v1beta1.Msg/SignerUpdate"
	Msg_ValidatorExit_FullMethodName = "/cosmos.staking.v1beta1.Msg/ValidatorExit"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	// JoinValidator defines a method for joining a new validator.
	JoinValidator(ctx context.Context, in *MsgValidatorJoin, opts ...grpc.CallOption) (*MsgValidatorJoinResponse, error)
	// StakeUpdate defines a method for updating an existing validator's stake.
	StakeUpdate(ctx context.Context, in *MsgStakeUpdate, opts ...grpc.CallOption) (*MsgStakeUpdateResponse, error)
	// v defines a method for updating an existing validator's signer.
	SignerUpdate(ctx context.Context, in *MsgSignerUpdate, opts ...grpc.CallOption) (*MsgSignerUpdateResponse, error)
	// ValidatorExit defines a method for exiting an existing validator
	ValidatorExit(ctx context.Context, in *MsgValidatorExit, opts ...grpc.CallOption) (*MsgValidatorExitResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) JoinValidator(ctx context.Context, in *MsgValidatorJoin, opts ...grpc.CallOption) (*MsgValidatorJoinResponse, error) {
	out := new(MsgValidatorJoinResponse)
	err := c.cc.Invoke(ctx, Msg_JoinValidator_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) StakeUpdate(ctx context.Context, in *MsgStakeUpdate, opts ...grpc.CallOption) (*MsgStakeUpdateResponse, error) {
	out := new(MsgStakeUpdateResponse)
	err := c.cc.Invoke(ctx, Msg_StakeUpdate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SignerUpdate(ctx context.Context, in *MsgSignerUpdate, opts ...grpc.CallOption) (*MsgSignerUpdateResponse, error) {
	out := new(MsgSignerUpdateResponse)
	err := c.cc.Invoke(ctx, Msg_SignerUpdate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ValidatorExit(ctx context.Context, in *MsgValidatorExit, opts ...grpc.CallOption) (*MsgValidatorExitResponse, error) {
	out := new(MsgValidatorExitResponse)
	err := c.cc.Invoke(ctx, Msg_ValidatorExit_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
type MsgServer interface {
	// JoinValidator defines a method for joining a new validator.
	JoinValidator(context.Context, *MsgValidatorJoin) (*MsgValidatorJoinResponse, error)
	// StakeUpdate defines a method for updating an existing validator's stake.
	StakeUpdate(context.Context, *MsgStakeUpdate) (*MsgStakeUpdateResponse, error)
	// v defines a method for updating an existing validator's signer.
	SignerUpdate(context.Context, *MsgSignerUpdate) (*MsgSignerUpdateResponse, error)
	// ValidatorExit defines a method for exiting an existing validator
	ValidatorExit(context.Context, *MsgValidatorExit) (*MsgValidatorExitResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) JoinValidator(context.Context, *MsgValidatorJoin) (*MsgValidatorJoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method JoinValidator not implemented")
}
func (UnimplementedMsgServer) StakeUpdate(context.Context, *MsgStakeUpdate) (*MsgStakeUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StakeUpdate not implemented")
}
func (UnimplementedMsgServer) SignerUpdate(context.Context, *MsgSignerUpdate) (*MsgSignerUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignerUpdate not implemented")
}
func (UnimplementedMsgServer) ValidatorExit(context.Context, *MsgValidatorExit) (*MsgValidatorExitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ValidatorExit not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_JoinValidator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgValidatorJoin)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).JoinValidator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_JoinValidator_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).JoinValidator(ctx, req.(*MsgValidatorJoin))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_StakeUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStakeUpdate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StakeUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_StakeUpdate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StakeUpdate(ctx, req.(*MsgStakeUpdate))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SignerUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSignerUpdate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SignerUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_SignerUpdate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SignerUpdate(ctx, req.(*MsgSignerUpdate))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ValidatorExit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgValidatorExit)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ValidatorExit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ValidatorExit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ValidatorExit(ctx, req.(*MsgValidatorExit))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cosmos.staking.v1beta1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "JoinValidator",
			Handler:    _Msg_JoinValidator_Handler,
		},
		{
			MethodName: "StakeUpdate",
			Handler:    _Msg_StakeUpdate_Handler,
		},
		{
			MethodName: "SignerUpdate",
			Handler:    _Msg_SignerUpdate_Handler,
		},
		{
			MethodName: "ValidatorExit",
			Handler:    _Msg_ValidatorExit_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cosmos/staking/v1beta1/tx.proto",
}
