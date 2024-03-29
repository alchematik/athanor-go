package plugin

import (
	"context"
	"errors"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	sdkerrors "github.com/alchematik/athanor-go/sdk/errors"
	"github.com/alchematik/athanor-go/sdk/provider/value"

	hcplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Serve(handlers map[string]ResoureceHandlerInitializer) {
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
	GetResource(context.Context, value.Identifier) (value.Resource, error)
	CreateResource(context.Context, value.Identifier, any) (value.Resource, error)
	UpdateResource(context.Context, value.Identifier, any, []value.UpdateMaskField) (value.Resource, error)
	DeleteResource(context.Context, value.Identifier) error

	Close() error
}

type ResoureceHandlerInitializer func(context.Context) (ResourceHandler, error)

type server struct {
	resourceHandlers map[string]ResoureceHandlerInitializer
}

func (s *server) GetResource(ctx context.Context, req *providerpb.GetResourceRequest) (*providerpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	initializer, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.GetResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	handler, err := initializer(ctx)
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer handler.Close()

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.GetResource(ctx, id)
	if err != nil {
		var notFound sdkerrors.ErrorNotFound
		if errors.As(err, &notFound) {
			return &providerpb.GetResourceResponse{}, status.Error(codes.NotFound, err.Error())
		}

		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	p, err := res.ToResourceProto()
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.GetResourceResponse{Resource: p}, nil
}

func (s *server) CreateResource(ctx context.Context, req *providerpb.CreateResourceRequest) (*providerpb.CreateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	initializer, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	handler, err := initializer(ctx)
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer handler.Close()

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

	p, err := res.ToResourceProto()
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.CreateResourceResponse{Resource: p}, nil
}

func (s *server) UpdateResource(ctx context.Context, req *providerpb.UpdateResourceRequest) (*providerpb.UpdateResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	initializer, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	handler, err := initializer(ctx)
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer handler.Close()

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	config, err := value.ParseProto(req.GetConfig())
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	protoMask := req.GetMask()
	mask := make([]value.UpdateMaskField, len(protoMask))
	for i := range protoMask {
		mask[i] = parseUpdateMaskFieldProto(protoMask[i])
	}

	res, err := handler.UpdateResource(ctx, id, config, mask)
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	p, err := res.ToResourceProto()
	if err != nil {
		return &providerpb.UpdateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.UpdateResourceResponse{Resource: p}, nil
}

func parseUpdateMaskFieldProto(field *providerpb.Field) value.UpdateMaskField {
	op := value.OperationUpdate
	if field.GetOperation() == providerpb.Operation_OPERATION_DELETE {
		op = value.OperationDelete
	}

	sub := make([]value.UpdateMaskField, len(field.GetSubFields()))
	for i := range field.GetSubFields() {
		sub[i] = parseUpdateMaskFieldProto(field.GetSubFields()[i])
	}

	return value.UpdateMaskField{
		Name:      field.GetName(),
		Operation: op,
		SubFields: sub,
	}
}

func (s *server) DeleteResource(ctx context.Context, req *providerpb.DeleteResourceRequest) (*providerpb.DeleteResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	initializer, ok := s.resourceHandlers[t]
	if !ok {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	handler, err := initializer(ctx)
	if err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer handler.Close()

	id, err := value.ParseIdentifierProto(req.GetIdentifier().GetIdentifier())
	if err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := handler.DeleteResource(ctx, id); err != nil {
		return &providerpb.DeleteResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.DeleteResourceResponse{}, nil
}
