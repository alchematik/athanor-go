// Code generated by athanor-go-translator.
// DO NOT EDIT.

package provider

import (
	sdk "github.com/alchematik/athanor-go/sdk/provider/value"
)

type Bar struct {
	Foo string
}

func (x Bar) ToValue() (sdk.Value, error) {
	foo, err := sdk.ToValue(x.Foo)
	if err != nil {
		return nil, err
	}

	return sdk.Map{
		"foo": foo,
	}, nil

}

type BucketAttrs struct {
	Bar Bar
}

func (x BucketAttrs) ToValue() (sdk.Value, error) {
	bar, err := sdk.ToValue(x.Bar)
	if err != nil {
		return nil, err
	}

	return sdk.Map{
		"bar": bar,
	}, nil

}

type BucketConfig struct {
	Expiration string
}

func (x BucketConfig) ToValue() (sdk.Value, error) {
	expiration, err := sdk.ToValue(x.Expiration)
	if err != nil {
		return nil, err
	}

	return sdk.Map{
		"expiration": expiration,
	}, nil

}

type BucketIdentifier struct {
	Account string
	Region  string
	Name    string
}

func (x BucketIdentifier) ResourceType() string {
	return "bucket"
}

func (x BucketIdentifier) ToValue() (sdk.Value, error) {
	account, err := sdk.ToValue(x.Account)
	if err != nil {
		return nil, err
	}
	region, err := sdk.ToValue(x.Region)
	if err != nil {
		return nil, err
	}
	name, err := sdk.ToValue(x.Name)
	if err != nil {
		return nil, err
	}

	return sdk.Identifier{
		ResourceType: x.ResourceType(),
		Value: sdk.Map{
			"account": account,
			"region":  region,
			"name":    name,
		},
	}, nil

}
