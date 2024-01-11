package main

import (
	"context"
	"fmt"

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

func ParseBucketIdentifier(val sdk.Value) (BucketIdentifier, error) {
	m, err := sdk.ParseMap(val)
	if err != nil {
		return BucketIdentifier{}, fmt.Errorf("expected MapType, got %T", val)
	}

	account, err := sdk.ParseStringValue(m["account"])
	if err != nil {
		return BucketIdentifier{}, fmt.Errorf("expected StringType, got %T", m["account"])
	}

	region, err := sdk.ParseStringValue(m["region"])
	if err != nil {
		return BucketIdentifier{}, fmt.Errorf("expected StringType, got %T", m["account"])
	}

	name, err := sdk.ParseStringValue(m["name"])
	if err != nil {
		return BucketIdentifier{}, fmt.Errorf("expected StringType, got %T", m["account"])
	}

	return BucketIdentifier{
		Account: account,
		Region:  region,
		Name:    name,
	}, nil
}

type BucketIdentifier struct {
	Account string
	Region  string
	Name    string
}

func (id BucketIdentifier) ResourceType() string {
	return "bucket"
}

func (id BucketIdentifier) ToValue() sdk.Value {
	return sdk.Identifier{
		ResourceType: "bucket",
		Value: sdk.Map{
			"account": sdk.ToStringValue(id.Account),
			"region":  sdk.ToStringValue(id.Region),
			"name":    sdk.ToStringValue(id.Name),
		},
	}
}

func ParseBucketConfig(val sdk.Value) BucketConfig {
	expiration := val.(sdk.Map)["expiration"].(sdk.String)
	return BucketConfig{
		Expiration: expiration,
	}
}

type BucketConfig struct {
	Expiration sdk.String
}

func (c BucketConfig) ToStateValue() sdk.Value {
	return sdk.MapValue{
		"expiration": c.Expiration,
	}
}

type BucketAttrs struct {
	Bar Bar
}

func (c BucketAttrs) ToStateValue() sdk.Value {
	return sdk.Map(map[string]sdk.Value{
		"bar": c.Bar.ToStateValue(),
	})
}

type Bar struct {
	Foo sdk.StringValue
}

func (b Bar) ToStateValue() sdk.Value {
	return sdk.Map(map[string]sdk.Value{
		"foo": b.Foo,
	})
}

type BucketObjectIdentifier struct {
	Bucket sdk.ResourceIdentifier
	Name   string
}

func (id BucketObjectIdentifier) ResourceType() string {
	return "bucket_object"
}

func (id BucketObjectIdentifier) ToStateValue() sdk.Value {
	return sdk.Identifier{
		ResourceType: id.ResourceType(),
		Value: sdk.Map{
			"bucket": id.Bucket.ToValue(),
			"name":   sdk.String(id.Name),
		},
	}
}

func ParseBucketObjectConfig(val sdk.Value) BucketObjectConfig {
	return BucketObjectConfig{
		Contents:  val.(sdk.MapValue)["contents"].(sdk.StringValue),
		SomeField: val.(sdk.MapValue)["some_field"].(sdk.StringValue),
	}
}

type BucketObjectConfig struct {
	Contents  sdk.StringValue
	SomeField sdk.StringValue
}

func (b BucketObjectConfig) ToStateValue() sdk.Value {
	return sdk.Map(map[string]sdk.Value{
		"contents":   b.Contents,
		"some_field": b.SomeField,
	})
}

type BucketObjectAttrs struct {
}

func (b BucketObjectAttrs) ToStateValue() sdk.Value {
	return sdk.Map(map[string]sdk.Value{})
}

func ParseBucketObjectIdentifier(val sdk.Value) (BucketObjectIdentifier, error) {
	m, _ := sdk.ParseMap(val)

	bucket, err := ParseIdentifier(m["bucket"])
	if err != nil {
		return BucketObjectIdentifier{}, err
	}

	name := m["name"].(sdk.StringValue)

	return BucketObjectIdentifier{
		Bucket: bucket,
		Name:   name,
	}, nil
}

func ParseIdentifier(val sdk.Value) (sdk.ResourceIdentifier, error) {
	id, ok := val.(sdk.Identifier)
	if !ok {
		return sdk.Identifier{}, fmt.Errorf("not an identifier")
	}

	switch id.ResourceType {
	case "bucket":
		return ParseBucketIdentifier(id.Value)
	case "bucket_object":
		return ParseBucketObjectIdentifier(id.Value)
	default:
		return nil, fmt.Errorf("not a valid resource: %v", val.ResourceType)
	}
}

type BucketGetter interface {
	GetBucket(context.Context, BucketIdentifier) (Bucket, error)
}

type BucketCreator interface {
	CreateBucket(context.Context, BucketIdentifier, BucketConfig) (Bucket, error)
}

type BucketUpdator interface {
	UpdateBucket(context.Context, BucketIdentifier, BucketConfig) (Bucket, error)
}

type BucketDeleter interface {
	DeleteBucket(context.Context, BucketIdentifier) error
}

type BucketObjectGetter interface {
	GetBucketObject(context.Context, BucketObjectIdentifier) (BucketObject, error)
}

type BucketObjectCreator interface {
	CreateBucketObject(context.Context, BucketObjectIdentifier, BucketObjectConfig) (BucketObject, error)
}

type BucketObjectUpdator interface {
	UpdateBucketObject(context.Context, BucketObjectIdentifier, BucketObjectConfig) (BucketObject, error)
}

type BucketObjectDeleter interface {
	DeleteBucketObject(context.Context, BucketObjectIdentifier) error
}

type ResourceHandler interface {
	GetResource(context.Context, sdk.IdentifierValue) (sdk.ResourceValue, error)
	CreateResource(context.Context, sdk.IdentifierValue, sdk.Value) (sdk.Resource, error)
}

type BucketHandler struct {
	BucketGetter  BucketGetter
	BucketCreator BucketCreator
}

type BucketObjectHandler struct {
	BucketObjectGetter  BucketObjectGetter
	BucketObjectCreator BucketObjectCreator
}

func (h BucketHandler) GetResource(ctx context.Context, id sdk.IdentifierValue) (sdk.ResourceValue, error) {
	if h.BucketGetter == nil {
		return sdk.ResourceValue{}, fmt.Errorf("unimplemented")
	}

	bucketID, err := ParseBucketIdentifier(id.Value)
	if err != nil {
		return sdk.ResourceValue{}, err
	}

	b, err := h.BucketGetter.GetBucket(ctx, bucketID)
	if err != nil {
		return sdk.ResourceValue{}, err
	}

	return b.ToResourceValue(), nil
}

func (h BucketHandler) CreateResource(ctx context.Context, id sdk.IdentifierValue, config sdk.Value) (sdk.ResourceValue, error) {
	if h.BucketCreator == nil {
		return sdk.ResourceValue{}, fmt.Errorf("unimplemented")
	}

	bucketID, err := ParseBucketIdentifier(id.Value)
	if err != nil {
		return sdk.ResourceValue{}, nil
	}

	bucketConfig := ParseBucketConfig(config)

	b, err := h.BucketCreator.CreateBucket(ctx, bucketID, bucketConfig)
	if err != nil {
		return sdk.ResourceValue{}, err
	}

	return b.ToResourceValue(), nil
}

func (h BucketObjectHandler) GetResource(ctx context.Context, id sdk.IdentifierValue) (sdk.ResourceValue, error) {
	if h.BucketObjectGetter == nil {
		return sdk.ResourceValue{}, fmt.Errorf("unimplemented")
	}

	bucketObjectID, err := ParseBucketObjectIdentifier(id.Value)
	if err != nil {
		return sdk.ResourceValue{}, err
	}

	bo, err := h.BucketObjectGetter.GetBucketObject(ctx, bucketObjectID)
	if err != nil {
		return sdk.ResourceValue{}, err
	}

	return bo.ToResourceValue(), nil
}

func (h BucketObjectHandler) CreateResource(ctx context.Context, id *providerpb.Value, config *providerpb.Value) (*providerpb.Resource, error) {
	// if h.BucketObjectCreator == nil {
	// 	return nil, fmt.Errorf("unimplemented")
	// }
	//
	// bucketObjectID := ParseBucketObjectIdentifier(id)
	// bucketObjectConfig := ParseBucketObjectConfig(config)
	//
	// res, err := h.BucketObjectCreator.CreateBucketObject(ctx, bucketObjectID, bucketObjectConfig)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// return res.ToStateValue().ToStateValueProto(), nil
	return nil, nil
}

type Bucket struct {
	Identifier BucketIdentifier
	Config     BucketConfig
	Attrs      BucketAttrs
}

func (b Bucket) ToResourceValue() sdk.ResourceValue {
	return sdk.ResourceValue{
		Identifier: b.Identifier.ToStateValue(),
		Config:     b.Config.ToStateValue(),
		Attrs:      b.Attrs.ToStateValue(),
	}
}

func (b Bucket) GetBucket(ctx context.Context, id BucketIdentifier) (Bucket, error) {
	config := BucketConfig{
		Expiration: "1d",
	}
	attrs := BucketAttrs{
		Bar: Bar{
			Foo: "hi",
		},
	}

	return Bucket{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b Bucket) CreateBucket(ctx context.Context, id BucketIdentifier, config BucketConfig) (sdk.Resource, error) {
	config.Expiration = "12h"

	attrs := BucketAttrs{
		Bar: Bar{
			Foo: "hi",
		},
	}

	return Bucket{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

type BucketObject struct {
	Identifier BucketObjectIdentifier
	Config     BucketObjectConfig
	Attrs      BucketObjectAttrs
}

func (b BucketObject) ToResourceValue() sdk.ResourceValue {
	return sdk.ResourceValue{
		Identifier: b.Identifier.ToStateValue(),
		Config:     b.Config.ToStateValue(),
		Attrs:      b.Attrs.ToStateValue(),
	}
}

func (b BucketObject) GetBucketObject(ctx context.Context, id BucketObjectIdentifier) (BucketObject, error) {
	config := BucketObjectConfig{
		Contents:  "blablabla",
		SomeField: "hehe",
	}

	attrs := BucketObjectAttrs{}

	return BucketObject{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b BucketObject) CreateBucketObject(ctx context.Context, id BucketObjectIdentifier, config BucketObjectConfig) (sdk.Resource, error) {
	return sdk.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      BucketAttrs{},
	}, nil
}

func (s *Server) GetResource(ctx context.Context, req *providerpb.GetResourceRequest) (*providerpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &providerpb.GetResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	val, err := sdk.ParseProto(req.GetIdentifier())
	if err != nil {
		return &providerpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	res, err := handler.GetResource(ctx, val.(sdk.IdentifierValue))
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

	res, err := handler.CreateResource(ctx, req.GetIdentifier(), req.GetConfig())
	if err != nil {
		return &providerpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &providerpb.CreateResourceResponse{Resource: res}, nil
}

func (s *Server) UpdateResource(ctx context.Context, req *providerpb.UpdateResourceRequest) (*providerpb.UpdateResourceResponse, error) {
	// var r *providerpb.Resource
	// switch req.GetIdentifier().GetIdentifier().GetType() {
	// case "bucket":
	// 	id := ParseBucketIdentifier(req.GetIdentifier())
	//
	// 	config := ParseBucketConfig(req.GetConfig())
	//
	// 	config.Expiration = "12h"
	//
	// 	attrs := BucketAttrs{
	// 		Bar: Bar{
	// 			Foo: "hi",
	// 		},
	// 	}
	//
	// 	r = sdk.Resource{
	// 		Identifier: id,
	// 		Config:     config,
	// 		Attrs:      attrs,
	// 	}.ToStateValue().ToStateValueProto()
	// case "bucket_object":
	// 	id := ParseBucketObjectIdentifier(req.GetIdentifier())
	//
	// 	config := ParseBucketObjectConfig(req.GetConfig())
	// 	config.Contents = "blablablablablablabla"
	// 	config.SomeField = "hi"
	//
	// 	attrs := BucketAttrs{}
	//
	// 	r = sdk.Resource{
	// 		Identifier: id,
	// 		Config:     config,
	// 		Attrs:      attrs,
	// 	}.ToStateValue().ToStateValueProto()
	// default:
	// 	return &providerpb.UpdateResourceResponse{}, status.Errorf(codes.InvalidArgument, "requires resource type")
	// }
	// return &providerpb.UpdateResourceResponse{
	// 	Resource: r,
	// }, nil
	return nil, nil
}

func (s *Server) DeleteResource(ctx context.Context, req *providerpb.DeleteResourceRequest) (*providerpb.DeleteResourceResponse, error) {
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
						"bucket": BucketHandler{
							BucketGetter: Bucket{},
							// BucketCreator: Bucket{},
						},
						"bucket_object": BucketObjectHandler{
							BucketObjectGetter:  BucketObject{},
							BucketObjectCreator: BucketObject{},
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
