package main

import (
	"context"

	provider "github.com/alchematik/athanor-go/example/schema/output"
	plugin "github.com/alchematik/athanor-go/sdk/provider/plugin"
	sdk "github.com/alchematik/athanor-go/sdk/provider/value"
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

func main() {
	plugin.Serve(map[string]plugin.ResourceHandler{
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
	})
}
