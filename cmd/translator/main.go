package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	translatorpb "github.com/alchematik/athanor-go/internal/gen/go/proto/translator/v1"
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

	resources := schema.GetResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].GetType() < resources[j].GetType()
	})

	var resourceNames []string
	types := map[string][]*providerpb.FieldSchema{}
	for _, resource := range schema.GetResources() {
		resourceNames = append(resourceNames, resource.GetType())

		resourceTypes, err := findResourceTypes(resource)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		resourceName := resource.GetType()
		types[resourceName] = resourceTypes
	}

	providerFile, err := os.Create(filepath.Join(req.GetOutputPath(), "provider.go"))
	if err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	src, err := provider.GenerateProvider(resourceNames)
	if err != nil {
		return nil, err
	}

	if _, err := providerFile.Write(src); err != nil {
		return nil, err
	}

	for resource, resourceTypes := range types {
		f, err := os.Create(filepath.Join(req.GetOutputPath(), resource+".go"))
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		d, err := provider.GenerateResource(resource, resourceTypes)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if _, err := f.Write(d); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	return &translatorpb.GenerateProvierSDKResponse{}, nil
}

func (s *Server) GenerateConsumerSDK(ctx context.Context, req *translatorpb.GenerateConsumerSDKRequest) (*translatorpb.GenerateConsumerSDKResponse, error) {
	return nil, nil
}

func findResourceTypes(resource *providerpb.ResourceSchema) ([]*providerpb.FieldSchema, error) {
	m := map[string]*providerpb.FieldSchema{}

	id := resource.GetIdentifier()
	id.Name = resource.GetType() + "_identifier"
	m[id.Name] = id

	config := resource.GetConfig()
	config.Name = resource.GetType() + "_config"
	m[config.Name] = config

	attrs := resource.GetAttrs()
	attrs.Name = resource.GetType() + "_attrs"
	m[attrs.Name] = attrs

	if err := addTypes(m, id); err != nil {
		return nil, err
	}

	if err := addTypes(m, config); err != nil {
		return nil, err
	}

	if err := addTypes(m, attrs); err != nil {
		return nil, err
	}

	var names []string
	for k := range m {
		names = append(names, k)
	}

	sort.Strings(names)

	resourceTypes := make([]*providerpb.FieldSchema, len(names))
	for i, name := range names {
		resourceTypes[i] = m[name]
	}

	return resourceTypes, nil
}

func addTypes(types map[string]*providerpb.FieldSchema, field *providerpb.FieldSchema) error {
	for _, f := range field.GetFields() {
		if f.GetType() == providerpb.FieldType_STRUCT {
			// Skip nested identifiers because those should get generated by the resource which the identifier belongs to.
			if f.GetIsIdentifier() {
				continue
			}

			if _, ok := types[f.GetName()]; ok {
				return fmt.Errorf("duplicate type definition: %s\n", f.GetName())
			}

			types[f.GetName()] = f
			if err := addTypes(types, f); err != nil {
				return err
			}
		}
	}

	return nil
}
