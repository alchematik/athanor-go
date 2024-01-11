package plugin

import (
	"context"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor-go/sdk/provider/value"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Plugin struct {
	plugin.Plugin

	server providerpb.ProviderServer
}

func (p *Plugin) Serve() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"provider": p,
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	providerpb.RegisterProviderServer(s, p.server)
	return nil
}

func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return providerpb.NewProviderClient(conn), nil
}

type ResourceHandler interface {
	GetResource(ctx context.Context, identifier value.Type) error
}

type server struct {
}

func (s *server) GetResource(ctx context.Context, req *providerpb.GetResourceRequest) (*providerpb.GetResourceResponse, error) {
	return nil, nil
}

func (s *server) CreateResource(ctx context.Context, req *providerpb.CreateResourceRequest) (*providerpb.CreateResourceResponse, error) {
	return nil, nil
}

func (s *server) UpdateResource(ctx context.Context, req *providerpb.UpdateResourceRequest) (*providerpb.UpdateResourceResponse, error) {
	return nil, nil
}

func (s *server) DeleteResource(ctx context.Context, req *providerpb.DeleteResourceRequest) (*providerpb.DeleteResourceResponse, error) {
	return nil, nil
}
