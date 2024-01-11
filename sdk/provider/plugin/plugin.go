package plugin

import (
	"context"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor-go/sdk/provider/value"

	hcplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Serve(handlers map[string]ResourceHandler) {
	hcplugin.Serve(&hcplugin.ServeConfig{
		HandshakeConfig: hcplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]hcplugin.Plugin{
			"provider": &plug{
				server: &server{
					resourceHandlers: handlers,
				},
			},
		},
		GRPCServer: hcplugin.DefaultGRPCServer,
	})
}

type plug struct {
	hcplugin.Plugin

	server providerpb.ProviderServer
}

func (p *plug) GRPCServer(_ *hcplugin.GRPCBroker, s *grpc.Server) error {
	providerpb.RegisterProviderServer(s, p.server)
	return nil
}

func (p *plug) GRPCClient(_ context.Context, _ *hcplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return providerpb.NewProviderClient(conn), nil
}

type ResourceHandler interface {
	GetResource(context.Context, value.Identifier) (value.ResourceValue, error)
	CreateResource(context.Context, value.Identifier, value.Value) (value.ResourceValue, error)
	UpdateResource(context.Context, value.Identifier, value.Value) (value.ResourceValue, error)
	DeleteResource(context.Context, value.Identifier) error
}

type server struct {
	resourceHandlers map[string]ResourceHandler
}

func (s *server) GetResource(ctx context.Context, req *providerpb.GetResourceRequest) (*providerpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.GetResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.GetResource(ctx, id)
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.GetResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *server) CreateResource(ctx context.Context, req *providerpb.CreateResourceRequest) (*providerpb.CreateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	config, err := value.ParseProto(req.GetConfig())
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.CreateResource(ctx, id, config)
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.CreateResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *server) UpdateResource(ctx context.Context, req *providerpb.UpdateResourceRequest) (*providerpb.UpdateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	config, err := value.ParseProto(req.GetConfig())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.UpdateResource(ctx, id, config)
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.UpdateResourceResponse{Resource: res.ToResourceProto()}, nil
}

func (s *server) DeleteResource(ctx context.Context, req *providerpb.DeleteResourceRequest) (*providerpb.DeleteResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := handler.DeleteResource(ctx, id); err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.DeleteResourceResponse{}, nil
}
