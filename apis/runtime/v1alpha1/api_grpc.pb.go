/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1alpha1

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

// RuntimeHookServiceClient is the client API for RuntimeHookService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RuntimeHookServiceClient interface {
	// PreRunPodSandboxHook calls RuntimeHookServer before pod creating, and would merge RunPodSandboxHookResponse
	// and Original RunPodSandboxRequest generating a new RunPodSandboxRequest to transfer to backend runtime engine.
	// RuntimeHookServer should ensure the correct operations basing on RunPodSandboxHookRequest.
	PreRunPodSandboxHook(ctx context.Context, in *PodSandboxHookRequest, opts ...grpc.CallOption) (*PodSandboxHookResponse, error)
	// PostStopPodSandbox calls RuntimeHookServer after pod deleted. RuntimeHookServer could do resource setting garbage collection
	// sanity check after PodSandBox stopped.
	PostStopPodSandbox(ctx context.Context, in *PodSandboxHookRequest, opts ...grpc.CallOption) (*PodSandboxHookRequest, error)
	// PreCreateContainerHook calls RuntimeHookServer before container creating. RuntimeHookServer could do some
	// resource setting before container launching.
	PreCreateContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookRequest, error)
	// PreStartContainerHook calls RuntimeHookServer before container starting, RuntimeHookServer could do some
	// resource adjustments before container launching.
	PreStartContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error)
	// PostStartContainerHook calls RuntimeHookServer after container starting. RuntimeHookServer could do resource
	// set checking after container launch.
	PostStartContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error)
	// PostStopContainerHook calls RuntimeHookServer after container stop. RuntimeHookServer could do resource setting
	// garbage collection.
	PostStopContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error)
	// PreUpdateContainerResourcesHook calls RuntimeHookServer before container resource update to keep resource policy
	// consistent
	PreUpdateContainerResourcesHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error)
}

type runtimeHookServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRuntimeHookServiceClient(cc grpc.ClientConnInterface) RuntimeHookServiceClient {
	return &runtimeHookServiceClient{cc}
}

func (c *runtimeHookServiceClient) PreRunPodSandboxHook(ctx context.Context, in *PodSandboxHookRequest, opts ...grpc.CallOption) (*PodSandboxHookResponse, error) {
	out := new(PodSandboxHookResponse)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PreRunPodSandboxHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PostStopPodSandbox(ctx context.Context, in *PodSandboxHookRequest, opts ...grpc.CallOption) (*PodSandboxHookRequest, error) {
	out := new(PodSandboxHookRequest)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PostStopPodSandbox", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PreCreateContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookRequest, error) {
	out := new(ContainerResourceHookRequest)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PreCreateContainerHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PreStartContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error) {
	out := new(ContainerResourceHookResponse)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PreStartContainerHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PostStartContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error) {
	out := new(ContainerResourceHookResponse)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PostStartContainerHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PostStopContainerHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error) {
	out := new(ContainerResourceHookResponse)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PostStopContainerHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runtimeHookServiceClient) PreUpdateContainerResourcesHook(ctx context.Context, in *ContainerResourceHookRequest, opts ...grpc.CallOption) (*ContainerResourceHookResponse, error) {
	out := new(ContainerResourceHookResponse)
	err := c.cc.Invoke(ctx, "/runtime.v1alpha1.RuntimeHookService/PreUpdateContainerResourcesHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RuntimeHookServiceServer is the server API for RuntimeHookService service.
// All implementations must embed UnimplementedRuntimeHookServiceServer
// for forward compatibility
type RuntimeHookServiceServer interface {
	// PreRunPodSandboxHook calls RuntimeHookServer before pod creating, and would merge RunPodSandboxHookResponse
	// and Original RunPodSandboxRequest generating a new RunPodSandboxRequest to transfer to backend runtime engine.
	// RuntimeHookServer should ensure the correct operations basing on RunPodSandboxHookRequest.
	PreRunPodSandboxHook(context.Context, *PodSandboxHookRequest) (*PodSandboxHookResponse, error)
	// PostStopPodSandbox calls RuntimeHookServer after pod deleted. RuntimeHookServer could do resource setting garbage collection
	// sanity check after PodSandBox stopped.
	PostStopPodSandbox(context.Context, *PodSandboxHookRequest) (*PodSandboxHookRequest, error)
	// PreCreateContainerHook calls RuntimeHookServer before container creating. RuntimeHookServer could do some
	// resource setting before container launching.
	PreCreateContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookRequest, error)
	// PreStartContainerHook calls RuntimeHookServer before container starting, RuntimeHookServer could do some
	// resource adjustments before container launching.
	PreStartContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error)
	// PostStartContainerHook calls RuntimeHookServer after container starting. RuntimeHookServer could do resource
	// set checking after container launch.
	PostStartContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error)
	// PostStopContainerHook calls RuntimeHookServer after container stop. RuntimeHookServer could do resource setting
	// garbage collection.
	PostStopContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error)
	// PreUpdateContainerResourcesHook calls RuntimeHookServer before container resource update to keep resource policy
	// consistent
	PreUpdateContainerResourcesHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error)
	mustEmbedUnimplementedRuntimeHookServiceServer()
}

// UnimplementedRuntimeHookServiceServer must be embedded to have forward compatible implementations.
type UnimplementedRuntimeHookServiceServer struct {
}

func (UnimplementedRuntimeHookServiceServer) PreRunPodSandboxHook(context.Context, *PodSandboxHookRequest) (*PodSandboxHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PreRunPodSandboxHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PostStopPodSandbox(context.Context, *PodSandboxHookRequest) (*PodSandboxHookRequest, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostStopPodSandbox not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PreCreateContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookRequest, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PreCreateContainerHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PreStartContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PreStartContainerHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PostStartContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostStartContainerHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PostStopContainerHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostStopContainerHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) PreUpdateContainerResourcesHook(context.Context, *ContainerResourceHookRequest) (*ContainerResourceHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PreUpdateContainerResourcesHook not implemented")
}
func (UnimplementedRuntimeHookServiceServer) mustEmbedUnimplementedRuntimeHookServiceServer() {}

// UnsafeRuntimeHookServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RuntimeHookServiceServer will
// result in compilation errors.
type UnsafeRuntimeHookServiceServer interface {
	mustEmbedUnimplementedRuntimeHookServiceServer()
}

func RegisterRuntimeHookServiceServer(s grpc.ServiceRegistrar, srv RuntimeHookServiceServer) {
	s.RegisterService(&RuntimeHookService_ServiceDesc, srv)
}

func _RuntimeHookService_PreRunPodSandboxHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PodSandboxHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PreRunPodSandboxHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PreRunPodSandboxHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PreRunPodSandboxHook(ctx, req.(*PodSandboxHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PostStopPodSandbox_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PodSandboxHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PostStopPodSandbox(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PostStopPodSandbox",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PostStopPodSandbox(ctx, req.(*PodSandboxHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PreCreateContainerHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerResourceHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PreCreateContainerHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PreCreateContainerHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PreCreateContainerHook(ctx, req.(*ContainerResourceHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PreStartContainerHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerResourceHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PreStartContainerHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PreStartContainerHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PreStartContainerHook(ctx, req.(*ContainerResourceHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PostStartContainerHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerResourceHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PostStartContainerHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PostStartContainerHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PostStartContainerHook(ctx, req.(*ContainerResourceHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PostStopContainerHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerResourceHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PostStopContainerHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PostStopContainerHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PostStopContainerHook(ctx, req.(*ContainerResourceHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RuntimeHookService_PreUpdateContainerResourcesHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerResourceHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RuntimeHookServiceServer).PreUpdateContainerResourcesHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/runtime.v1alpha1.RuntimeHookService/PreUpdateContainerResourcesHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RuntimeHookServiceServer).PreUpdateContainerResourcesHook(ctx, req.(*ContainerResourceHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RuntimeHookService_ServiceDesc is the grpc.ServiceDesc for RuntimeHookService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RuntimeHookService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "runtime.v1alpha1.RuntimeHookService",
	HandlerType: (*RuntimeHookServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PreRunPodSandboxHook",
			Handler:    _RuntimeHookService_PreRunPodSandboxHook_Handler,
		},
		{
			MethodName: "PostStopPodSandbox",
			Handler:    _RuntimeHookService_PostStopPodSandbox_Handler,
		},
		{
			MethodName: "PreCreateContainerHook",
			Handler:    _RuntimeHookService_PreCreateContainerHook_Handler,
		},
		{
			MethodName: "PreStartContainerHook",
			Handler:    _RuntimeHookService_PreStartContainerHook_Handler,
		},
		{
			MethodName: "PostStartContainerHook",
			Handler:    _RuntimeHookService_PostStartContainerHook_Handler,
		},
		{
			MethodName: "PostStopContainerHook",
			Handler:    _RuntimeHookService_PostStopContainerHook_Handler,
		},
		{
			MethodName: "PreUpdateContainerResourcesHook",
			Handler:    _RuntimeHookService_PreUpdateContainerResourcesHook_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}
