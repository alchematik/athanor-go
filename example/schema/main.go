package main

import (
	"log"

	"github.com/alchematik/athanor-go/sdk/provider/schema"
)

func main() {
	bucket := schema.ResourceSchema{
		Type: "bucket",
		Identifier: schema.FieldSchema{
			Name:         "identifier",
			Type:         schema.FieldTypeStruct,
			IsIdentifier: true,
			Fields: []schema.FieldSchema{
				{
					Name: "account",
					Type: schema.FieldTypeString,
				},
				{
					Name: "region",
					Type: schema.FieldTypeString,
				},
				{
					Name: "name",
					Type: schema.FieldTypeString,
				},
			},
		},
		Config: schema.FieldSchema{
			Name: "config",
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "expiration",
					Type: schema.FieldTypeString,
				},
			},
		},
		Attrs: schema.FieldSchema{
			Name: "attrs",
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "bar",
					Type: schema.FieldTypeStruct,
					Fields: []schema.FieldSchema{
						{
							Name: "foo",
							Type: schema.FieldTypeString,
						},
					},
				},
			},
		},
	}

	bucketObject := schema.ResourceSchema{
		Type: "bucket_object",
		Identifier: schema.FieldSchema{
			Name:         "identifier",
			Type:         schema.FieldTypeStruct,
			IsIdentifier: true,
			Fields: []schema.FieldSchema{
				{
					Name:         "bucket",
					Type:         schema.FieldTypeStruct,
					Fields:       bucket.Identifier.Fields,
					IsIdentifier: true,
				},
				{
					Name: "name",
					Type: schema.FieldTypeString,
				},
			},
		},
		Config: schema.FieldSchema{
			Name: "config",
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "contents",
					Type: schema.FieldTypeString,
				},
				{
					Name: "some_field",
					Type: schema.FieldTypeString,
				},
			},
		},
		Attrs: schema.FieldSchema{
			Name: "attrs",
			Type: schema.FieldTypeStruct,
		},
	}

	s := schema.Schema{
		Name:    "gcp",
		Version: "v0.0.1",
		Resources: []schema.ResourceSchema{
			bucket,
			bucketObject,
		},
	}

	if err := schema.Build(s); err != nil {
		log.Fatalf("error building provider: %v", err)
	}
}
