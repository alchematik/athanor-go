package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	translatorpb "github.com/alchematik/athanor-go/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor-go/internal/generate/consumer"
	"github.com/alchematik/athanor-go/internal/generate/provider"

	wasmtime "github.com/bytecodealliance/wasmtime-go/v19"
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
	inputPath := req.GetInputPath()
	outputPath := req.GetOutputPath()
	configPath := req.GetConfigPath()

	buildDir, err := os.MkdirTemp("", "translator")
	if err != nil {
		log.Printf("FAILED TO create temp dir >> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	buildPath := filepath.Join(buildDir, "main.wasm")
	//defer os.RemoveAll(buildDir)

	configSource, err := os.Open(configPath)
	if err != nil {
		log.Printf("failed to open config source >> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer configSource.Close()

	buildConfigPath := filepath.Join(buildDir, "config")
	buildConfigFile, err := os.Create(buildConfigPath)
	if err != nil {
		log.Printf("failed to create build config file >> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	if _, err := io.Copy(buildConfigFile, configSource); err != nil {
		log.Printf("failed to copy config file >> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	if err := buildConfigFile.Close(); err != nil {
		log.Printf("failed to close config file >> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	cmd := exec.Command("go", "build", "-o", buildPath, inputPath)
	cmd.Env = append(cmd.Environ(), "GOOS=wasip1", "GOARCH=wasm")

	log.Printf("BUILD PATH >>> %v\n", buildPath)
	log.Printf("INPUT PATH >>> %v\n", inputPath)

	// cmd := exec.Command("go", "run", inputPath, configPath, outputPath)
	logger := log.Default()
	cmd.Stdout = logger.Writer()
	cmd.Stderr = logger.Writer()

	if err := cmd.Run(); err != nil {
		log.Printf("FAILED TO BUILD WASM>> %v\n", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	engine := wasmtime.NewEngine()
	// cfg := wasmtime.NewConfig()
	// wasmtime.NewEngineWithConfig(cfg)

	// module, err := wasmtime.NewModuleFromFile(engine, "./sub/main.wasm")
	module, err := wasmtime.NewModuleFromFile(engine, buildPath)
	if err != nil {
		log.Printf("failed to create module: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	// Create a linker with WASI functions defined within it
	linker := wasmtime.NewLinker(engine)
	err = linker.DefineWasi()
	if err != nil {
		log.Printf("failed to link: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	buildOutputPath := filepath.Join(buildDir, "output")
	f, err := os.Create(buildOutputPath)
	if err != nil {
		log.Printf("failed to create output file: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}
	if err := f.Close(); err != nil {
		log.Printf("failed to close output file: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.InheritStderr()
	wasiConfig.InheritStdout()
	wasiConfig.SetArgv([]string{"config", "output"})

	if err := wasiConfig.PreopenDir(buildDir, "/"); err != nil {
		log.Printf("FAILED TO PREOPEN DIR: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	store := wasmtime.NewStore(engine)
	store.SetWasi(wasiConfig)
	instance, err := linker.Instantiate(store, module)
	if err != nil {
		log.Printf("failed to instantiate: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	nom := instance.GetFunc(store, "_start")
	_, err = nom.Call(store)
	if err != nil {
		// log.Fatal("failed to call _start", err)
		var wasmtimeError *wasmtime.Error
		if errors.As(err, &wasmtimeError) {
			st, ok := wasmtimeError.ExitStatus()
			fmt.Printf("err >>> %v, %v\n", st, ok)
		} else {
			fmt.Printf("unknown error: %v\n", err)
		}
	}

	outputFile, err := os.Open(buildOutputPath)
	if err != nil {
		log.Printf("failed to open output file: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer outputFile.Close()

	log.Printf("OUTPUT FILE >> %v", outputPath)
	outputDestFile, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Printf("failed to open output dest file: %v", err)
		return &translatorpb.TranslateBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}
	defer outputDestFile.Close()

	if _, err := io.Copy(outputDestFile, outputFile); err != nil {
		log.Printf("failed to copy output: %v", err)
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

	if err := os.MkdirAll(filepath.Join(req.GetOutputPath(), "identifier"), 0777); err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	providerFile, err := os.Create(filepath.Join(req.GetOutputPath(), "identifier", "identifier.go"))
	if err != nil {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
	}

	mod := req.GetArgs()["module"]
	if mod == "" {
		return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, "missing module")
	}

	commonSrc, err := provider.GenerateProviderCommonSrc(mod, req.GetOutputPath(), &schema)
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
		if err := os.MkdirAll(filepath.Join(req.GetOutputPath(), resource.GetType()), 0777); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		f, err := os.Create(filepath.Join(req.GetOutputPath(), resource.GetType(), resource.GetType()+".go"))
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		src, err := provider.GenerateResourceSrc(mod, req.GetOutputPath(), resource)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, "error generating resource src: "+err.Error())
		}

		if _, err := f.Write(src); err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		// Make id file.
		idFile, err := os.Create(filepath.Join(req.GetOutputPath(), "identifier", resource.GetType()+".go"))
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		src, err = provider.GenerateIdentifierSrc(resource)
		if err != nil {
			return &translatorpb.GenerateProvierSDKResponse{}, status.Error(codes.Internal, "error generating identifier src: "+err.Error())
		}

		if _, err := idFile.Write(src); err != nil {
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

	resources := schema.GetResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].GetType() < resources[j].GetType()
	})

	for _, resource := range resources {
		outPath := filepath.Join(req.GetOutputPath(), resource.GetType())

		if err := os.MkdirAll(outPath, 0777); err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		f, err := os.Create(filepath.Join(outPath, "resource.go"))
		if err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		src, err := consumer.GenerateResourceSrc(&schema, resource)
		if err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}

		if _, err := f.Write(src); err != nil {
			return &translatorpb.GenerateConsumerSDKResponse{}, status.Error(codes.Internal, err.Error())
		}
	}

	return &translatorpb.GenerateConsumerSDKResponse{}, nil
}
