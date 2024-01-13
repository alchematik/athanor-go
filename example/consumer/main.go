package main

import (
	"log"

	provider "github.com/alchematik/athanor-go/example/schema/output/consumer"
	sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

func main() {
	p := sdk.Provider("gcp", sdk.String("gcp"), sdk.String("v0.0.1"))

	bp := sdk.Blueprint{}

	bucketID := provider.BucketIdentifier{
		Alias:   "my-bucket",
		Account: sdk.String("12345"),
		Region:  sdk.String("us-east-1"),
		Name:    sdk.String("my-cool-bucket"),
	}
	bucketConfig := provider.BucketConfig{
		Expiration: sdk.String("12h"),
	}

	bp = bp.WithResource(sdk.Resource(sdk.Bool(true), p, bucketID, bucketConfig))

	objectID := provider.BucketObjectIdentifier{
		Alias:  "my-bucket-object",
		Bucket: bucketID,
		Name:   sdk.String("my-bucket-object"),
	}
	objectConfig := provider.BucketObjectConfig{
		Contents:  sdk.String("blabla"),
		SomeField: sdk.GetResource(bucketID.Alias).GetAttrs().IOGet("bar").IOGet("foo"),
	}

	bp = bp.WithResource(sdk.Resource(sdk.Bool(true), p, objectID, objectConfig))

	if err := sdk.Build(bp); err != nil {
		log.Fatalf("error building blueprint: %v\n", err)
	}
}
