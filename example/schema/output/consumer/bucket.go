// Code generated by athanor-go.
// DO NOT EDIT.

package gcp

import (
	sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

type BucketConfig struct {
	Expiration any
}

func (x BucketConfig) ToExpr() any {
	return map[string]any{
		"expiration": x.Expiration,
	}
}

type BucketIdentifier struct {
	Alias string

	Account any
	Region  any
	Name    any
}

func (x BucketIdentifier) ToExpr() any {
	return sdk.ResourceIdentifier{
		ResourceType: "bucket",
		Alias:        x.Alias,
		Value: map[string]any{
			"account": x.Account,
			"region":  x.Region,
			"name":    x.Name,
		},
	}
}
