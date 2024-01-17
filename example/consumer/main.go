package main

import (
	"log"

	provider "github.com/alchematik/athanor-go/example/schema/output/consumer"
	sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

func main() {
	p := sdk.Provider{Name: "gcp", Version: "v0.0.1"}

	bp := sdk.Blueprint{}

	bucket := sdk.Resource{
		Exists:   true,
		Provider: p,
		Identifier: provider.BucketIdentifier{
			Alias:   "my-bucket",
			Account: "12345",
			Region:  "us-east-1",
			Name:    "my-cool-bucket",
		},
		Config: provider.BucketConfig{
			Expiration: "12h",
		},
	}

	bp = bp.WithResource(bucket)

	bucketObject := sdk.Resource{
		Exists:   true,
		Provider: p,
		Identifier: provider.BucketObjectIdentifier{
			Alias:  "my-bucket-object",
			Bucket: bucket.Identifier,
			Name:   "my-bucket-object",
		},
		Config: provider.BucketObjectConfig{
			Contents:  "blabla",
			SomeField: sdk.GetResource("my-bucket").Get("attrs").Get("bar").Get("foo"),
		},
	}

	bp = bp.WithResource(bucketObject)

	if err := sdk.Build(bp); err != nil {
		log.Fatalf("error building blueprint: %v", err)
	}
}
