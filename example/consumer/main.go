package main

import (
	"log"

	provider "github.com/alchematik/athanor-go/example/schema/output/consumer"
	sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

func main() {
	p := sdk.Provider("gcp", "v0.0.1")

	bp := sdk.Blueprint{}

	bucketID := provider.BucketIdentifier{
		Alias:   "my-bucket",
		Account: "12345",
		Region:  "us-east-1",
		Name:    "my-cool-bucket",
	}
	bucketConfig := provider.BucketConfig{
		Expiration: "12h",
	}

	bp = bp.WithResource(sdk.Resource(true, p, bucketID, bucketConfig))

	objectID := provider.BucketObjectIdentifier{
		Alias:  "my-bucket-object",
		Bucket: bucketID,
		Name:   "my-bucket-object",
	}
	objectConfig := provider.BucketObjectConfig{
		Contents:  "blabla",
		SomeField: sdk.IOGet(bucketID.Alias, nil).IOGet("attrs").IOGet("bar").IOGet("foo"),
	}

	bp = bp.WithResource(sdk.Resource(true, p, objectID, objectConfig))

	if err := sdk.Build(bp); err != nil {
		log.Fatalf("error building blueprint: %v", err)
	}
}
