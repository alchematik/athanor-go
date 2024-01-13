// Code generated by athanor-go.
// DO NOT EDIT.

package provider

import (
	sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

type BucketConfig struct {
	Expiration sdk.Type
}

func (x BucketConfig) ToExpr() sdk.Expr {
	return sdk.Map(map[string]sdk.Type{
		"expiration": x.Expiration,
	}).ToExpr()
}

type BucketIdentifier struct {
	Alias string

	Account sdk.Type
	Region  sdk.Type
	Name    sdk.Type
}

func (x BucketIdentifier) ToExpr() sdk.Expr {
	return sdk.ResourceIdentifier(
		"bucket",
		x.Alias,
		sdk.Map(map[string]sdk.Type{
			"account": x.Account,
			"region":  x.Region,
			"name":    x.Name,
		}),
	).ToExpr()
}