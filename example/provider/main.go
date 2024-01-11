package main

import (
	"context"

	provider "github.com/alchematik/athanor-go/example/schema/output"
	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	sdk "github.com/alchematik/athanor-go/sdk/provider/value"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	ResourceHandlers map[string]ResourceHandler
}

type ResourceHandler interface {
	GetResource(context.Context, sdk.Identifier) (sdk.ResourceValue, error)
	CreateResource(context.Context, sdk.Identifier, sdk.Value) (sdk.ResourceValue, error)
	UpdateResource(context.Context, sdk.Identifier, sdk.Value) (sdk.ResourceValue, error)
	DeleteResource(context.Context, sdk.Identifier) error
}

type Bucket struct {
}

func (b Bucket) GetBucket(ctx context.Context, id provider.BucketIdentifier) (provider.Bucket, error) {
	config := provider.BucketConfig{
		Expiration: "1d",
	}
	attrs := provider.BucketAttrs{
		Bar: provider.Bar{
			Foo: "hi",
		},
	}

	return provider.Bucket{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b Bucket) CreateBucket(ctx context.Context, id provider.BucketIdentifier, config provider.BucketConfig) (provider.Bucket, error) {
	config.Expiration = "12h"

	attrs := provider.BucketAttrs{
		Bar: provider.Bar{
			Foo: "hi",
		},
	}

	return provider.Bucket{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b Bucket) UpdateBucket(ctx context.Context, id provider.BucketIdentifier, config provider.BucketConfig) (provider.Bucket, error) {
	attrs := provider.BucketAttrs{
		Bar: provider.Bar{
			Foo: "hi",
		},
	}
	return provider.Bucket{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b Bucket) DeleteBucket(ctx context.Context, id provider.BucketIdentifier) error {
	return nil
}

type BucketObject struct {
}

func (b BucketObject) GetBucketObject(ctx context.Context, id provider.BucketObjectIdentifier) (provider.BucketObject, error) {
	config := provider.BucketObjectConfig{
		Contents:  "blablabla",
		SomeField: "hehe",
	}

	attrs := provider.BucketObjectAttrs{}

	return provider.BucketObject{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b BucketObject) CreateBucketObject(ctx context.Context, id provider.BucketObjectIdentifier, config provider.BucketObjectConfig) (provider.BucketObject, error) {
	return provider.BucketObject{
		Identifier: id,
		Config:     config,
		Attrs:      provider.BucketObjectAttrs{},
	}, nil
}

func (b BucketObject) UpdateBucketObject(ctx context.Context, id provider.BucketObjectIdentifier, config provider.BucketObjectConfig) (provider.BucketObject, error) {
	return provider.BucketObject{
		Identifier: id,
		Config:     config,
		Attrs:      provider.BucketObjectAttrs{},
	}, nil
}

func (b BucketObject) DeleteBucketObject(ctx context.Context, id provider.BucketObjectIdentifier) error {
	return nil
}

func (s *Server) GetResource(ctx context.Context, req *providerpb.GetResourceRequest) (*providerpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &providerpb.GetResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := sdk.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.GetResource(ctx, id)
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.GetResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *Server) CreateResource(ctx context.Context, req *providerpb.CreateResourceRequest) (*providerpb.CreateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := sdk.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	config, err := sdk.ParseProto(req.GetConfig())
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.CreateResource(ctx, id, config)
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.CreateResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *Server) UpdateResource(ctx context.Context, req *providerpb.UpdateResourceRequest) (*providerpb.UpdateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := sdk.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	config, err := sdk.ParseProto(req.GetConfig())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.UpdateResource(ctx, id, config)
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.UpdateResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *Server) DeleteResource(ctx context.Context, req *providerpb.DeleteResourceRequest) (*providerpb.DeleteResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := sdk.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := handler.DeleteResource(ctx, id); err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.DeleteResourceResponse{}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"backend": &Plugin{
				BackendServer: &Server{
					ResourceHandlers: map[string]ResourceHandler{
						"bucket": provider.BucketHandler{
							BucketGetter:  Bucket{},
							BucketCreator: Bucket{},
							BucketUpdator: Bucket{},
							BucketDeleter: Bucket{},
						},
						"bucket_object": provider.BucketObjectHandler{
							BucketObjectGetter:  BucketObject{},
							BucketObjectCreator: BucketObject{},
							BucketObjectUpdator: BucketObject{},
							BucketObjectDeleter: BucketObject{},
						},
					},
				},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

type Plugin struct {
	plugin.Plugin

	BackendServer providerpb.ProviderServer
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	providerpb.RegisterProviderServer(s, p.BackendServer)
	return nil
}

func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return providerpb.NewProviderClient(conn), nil
}
