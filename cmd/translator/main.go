package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	translatorpb "github.com/alchematik/athanor-go/internal/gen/go/proto/translator/v1"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	types := map[string][]*providerpb.FieldSchema{}
	for _, resource := range schema.GetResources() {

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
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if err := addTypes(m, config); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if err := addTypes(m, attrs); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
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

		resourceName := resource.GetType()
		types[resourceName] = resourceTypes
	}

	providerFile, err := os.Create(filepath.Join(req.GetOutputPath(), "provider.go"))
	if err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	tmpl, err := template.New("provider").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(providerTmpl)
	if err != nil {
		return nil, err
	}

	typeNames := make([]string, 0, len(types))
	for k := range types {
		typeNames = append(typeNames, k)
	}

	sort.Strings(typeNames)

	providerData := map[string]any{
		"PackageName": "provider",
		"Types":       typeNames,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, providerData); err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
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

		d, err := GenerateResourceType(resource, resourceTypes)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if _, err := f.Write(d); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	/*

		TODO:
			- Generate resource
				- identifier
				- config
				- attrs
				- any structs
	*/

	return &translatorpb.GenerateProvierSDKResponse{}, nil
}

func (s *Server) GenerateConsumerSDK(ctx context.Context, req *translatorpb.GenerateConsumerSDKRequest) (*translatorpb.GenerateConsumerSDKResponse, error) {
	return nil, nil
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

//go:embed resource_header.tmpl
var resourceHeaderTmpl string

//go:embed struct_type.tmpl
var structTypeTmpl string

//go:embed string_type.tmpl
var stringTypeTmpl string

//go:embed provider.tmpl
var providerTmpl string

func GenerateResourceType(name string, types []*providerpb.FieldSchema) ([]byte, error) {
	var out []byte

	tmpl, err := template.New("resource_header").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(resourceHeaderTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"PackageName": "provider",
		"Type":        toPascalCase(name),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out = append(out, buf.Bytes()...)

	for _, t := range types {
		switch t.GetType() {
		case providerpb.FieldType_STRING:
		case providerpb.FieldType_STRUCT:
			tmpl, err := template.New("struct_type").
				Funcs(template.FuncMap{
					"toPascalCase": toPascalCase,
					"parseFieldFunc": func(f *providerpb.FieldSchema) (string, error) {
						if f.IsIdentifier {
							return "ParseIdentifier", nil
						}
						switch f.GetType() {
						case providerpb.FieldType_STRING:
							return "sdk.ParseStringValue", nil
						case providerpb.FieldType_STRUCT:
							return fmt.Sprintf("Parse%s", toPascalCase(f.Name)), nil
						default:
							return "", fmt.Errorf("unsupported type %s", f.GetType())
						}
					},
					"toType": func(f *providerpb.FieldSchema) (string, error) {
						if f.GetIsIdentifier() {
							return "sdk.ResourceIdentifier", nil
						}

						switch f.GetType() {
						case providerpb.FieldType_STRING:
							return "string", nil
						case providerpb.FieldType_STRUCT:
							return toPascalCase(f.GetName()), nil
						default:
							return "", fmt.Errorf("unrecognized type: %s", f.GetType())
						}
					},
				}).
				Parse(structTypeTmpl)
			if err != nil {
				return nil, err
			}

			data := map[string]any{
				"ResourceName": name,
				"Type":         t,
			}

			var buffer bytes.Buffer
			if err := tmpl.Execute(&buffer, data); err != nil {
				return nil, err
			}

			out = append(out, buffer.Bytes()...)
		}
	}

	src, err := format.Source(out)
	if err != nil {
		return nil, err
	}

	return src, nil
}

func toPascalCase(str string) string {
	titleCaser := cases.Title(language.Und)
	upperCaser := cases.Upper(language.Und)
	splitter := func(r rune) bool {
		return r == '_' || r == ' '
	}
	parts := strings.FieldsFunc(str, splitter)
	if len(parts) == 1 {
		part := parts[0]
		if upperCaser.String(part) == part {
			return part
		}

		return titleCaser.String(parts[0])
	}

	var transformed []string
	for _, part := range parts {
		if upperCaser.String(part) == part {
			transformed = append(transformed, part)
			continue
		}

		transformed = append(transformed, titleCaser.String(part))
	}

	return strings.Join(transformed, "")
}

func findStructFields(field *providerpb.FieldSchema) []*providerpb.FieldSchema {
	// Use map?
	// Put all structs across resources in same package?
	var structFields []*providerpb.FieldSchema
	for _, f := range field.GetFields() {
		if f.GetType() == providerpb.FieldType_STRUCT {
			structFields = append(structFields, f)
		}
	}

	return structFields
}
