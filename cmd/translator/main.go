package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	translatorpb "github.com/alchematik/athanor-go/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor-go/internal/generate/consumer"
	"github.com/alchematik/athanor-go/internal/generate/provider"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"translator": &Plugin{
				TranslatorServer: &Server{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

type Plugin struct {
	plugin.Plugin

	TranslatorServer translatorpb.TranslatorServer
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	translatorpb.RegisterTranslatorServer(s, p.TranslatorServer)
	return nil
}

func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return translatorpb.NewTranslatorClient(conn), nil
}

type Server struct {
}

func (s *Server) TranslateProviderSchema(ctx context.Context, req *translatorpb.TranslateProviderSchemaRequest) (*translatorpb.TranslateProviderSchemaResponse, error) {
	if err := exec.Command("go", "run", req.GetInputPath(), req.GetOutputPath()).Run(); err != nil {
		return &translatorpb.TranslateProviderSchemaResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &translatorpb.TranslateProviderSchemaResponse{}, nil
}

func (s *Server) TranslateBlueprint(ctx context.Context, req *translatorpb.TranslateBlueprintRequest) (*translatorpb.TranslateBlueprintResponse, error) {
	log.Printf("GETTING TRANSLATE BLUEPRINT %v\n", req.GetInputPath())
	if err := exec.Command("go", "run", req.GetInputPath(), req.GetOutputPath()).Run(); err != nil {
		log.Printf("ERROR WHILE TRANSLATING BLUEPRINT: %v\n", err.Error())
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &translatorpb.TranslateBlueprintResponse{}, nil
}

func (s *Server) GenerateProviderSDK(ctx context.Context, req *translatorpb.GenerateProviderSDKRequest) (*translatorpb.GenerateProvierSDKResponse, error) {
	data, err := os.ReadFile(req.GetInputPath())
	if err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	var schema providerpb.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := os.MkdirAll(req.GetOutputPath(), 0777); err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	providerFile, err := os.Create(filepath.Join(req.GetOutputPath(), "provider.go"))
	if err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	commonSrc, err := provider.GenerateProviderCommonSrc(&schema)
	if err != nil {
		return nil, err
	}

	if _, err := providerFile.Write(commonSrc); err != nil {
		return nil, err
	}

	resources := schema.GetResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].GetType() < resources[j].GetType()
	})

	for _, resource := range resources {
		f, err := os.Create(filepath.Join(req.GetOutputPath(), resource.GetType()+".go"))
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		src, err := provider.GenerateResourceSrc(resource)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if _, err := f.Write(src); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	return &translatorpb.GenerateProvierSDKResponse{}, nil
}

func (s *Server) GenerateConsumerSDK(ctx context.Context, req *translatorpb.GenerateConsumerSDKRequest) (*translatorpb.GenerateConsumerSDKResponse, error) {
	data, err := os.ReadFile(req.GetInputPath())
	if err != nil {
		return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	var schema providerpb.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := os.MkdirAll(req.GetOutputPath(), 0777); err != nil {
		return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	resources := schema.GetResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].GetType() < resources[j].GetType()
	})

	for _, resource := range resources {
		f, err := os.Create(filepath.Join(req.GetOutputPath(), resource.GetType()+".go"))
		if err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		src, err := consumer.GenerateResourceSrc(resource)
		if err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if _, err := f.Write(src); err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	return &translatorpb.GenerateConsumerSDKResponse{}, nil
}
