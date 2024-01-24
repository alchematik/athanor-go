package schema

import (
	"fmt"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
)

type Schema struct {
	Name      string
	Version   string
	Resources []ResourceSchema
}

func (s Schema) ToProto() (*providerpb.Schema, error) {
	resources := make([]*providerpb.ResourceSchema, len(s.Resources))
	for i, r := range s.Resources {
		var err error
		resources[i], err = r.ToProto()
		if err != nil {
			return nil, err
		}
	}

	return &providerpb.Schema{
		Name:      s.Name,
		Version:   s.Version,
		Resources: resources,
	}, nil
}

type ResourceSchema struct {
	Type       string
	Identifier FieldSchema
	Config     FieldSchema
	Attrs      FieldSchema
}

func (r ResourceSchema) ToProto() (*providerpb.ResourceSchema, error) {
	idProto, err := FieldSchemaToProto(r.Identifier)
	if err != nil {
		return nil, err
	}

	configProto, err := FieldSchemaToProto(r.Config)
	if err != nil {
		return nil, err
	}

	attrProto, err := FieldSchemaToProto(r.Attrs)
	if err != nil {
		return nil, err
	}

	return &providerpb.ResourceSchema{
		Type:       r.Type,
		Identifier: idProto,
		Config:     configProto,
		Attrs:      attrProto,
	}, nil
}

func String() StringSchema {
	return StringSchema{}
}

func Bool() BoolSchema {
	return BoolSchema{}
}

func Map(value FieldSchema) MapSchema {
	return MapSchema{
		Value: value,
	}
}

func File() FileSchema {
	return FileSchema{}
}

func Identifier() IdentifierSchema {
	return IdentifierSchema{}
}

func List(element FieldSchema) ListSchema {
	return ListSchema{
		Element: element,
	}
}

func Struct(name string, fields map[string]FieldSchema) StructSchema {
	return StructSchema{
		Name:   name,
		Fields: fields,
	}
}

type FieldSchema interface {
}

type StringSchema struct {
	FieldSchema
}

type BoolSchema struct {
	FieldSchema
}

type MapSchema struct {
	FieldSchema

	Value FieldSchema
}

type FileSchema struct {
	FieldSchema
}

type IdentifierSchema struct {
	FieldSchema
}

type ListSchema struct {
	FieldSchema

	Element FieldSchema
}

type StructSchema struct {
	FieldSchema

	Name   string
	Fields map[string]FieldSchema
}

func FieldSchemaToProto(f FieldSchema) (*providerpb.FieldSchema, error) {
	switch val := f.(type) {
	case StringSchema:
		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_StringSchema{
				StringSchema: &providerpb.StringSchema{},
			},
		}, nil
	case BoolSchema:
		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_BoolSchema{
				BoolSchema: &providerpb.BoolSchema{},
			},
		}, nil
	case MapSchema:
		valProto, err := FieldSchemaToProto(val.Value)
		if err != nil {
			return nil, err
		}

		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_MapSchema{
				MapSchema: &providerpb.MapSchema{
					Value: valProto,
				},
			},
		}, nil
	case IdentifierSchema:
		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_IdentifierSchema{
				IdentifierSchema: &providerpb.IdentifierSchema{},
			},
		}, nil
	case ListSchema:
		elementProto, err := FieldSchemaToProto(val.Element)
		if err != nil {
			return nil, err
		}

		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_ListSchema{
				ListSchema: &providerpb.ListSchema{
					Element: elementProto,
				},
			},
		}, nil
	case StructSchema:
		m := map[string]*providerpb.FieldSchema{}
		for k, v := range val.Fields {
			var err error
			m[k], err = FieldSchemaToProto(v)
			if err != nil {
				return nil, err
			}
		}

		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_StructSchema{
				StructSchema: &providerpb.StructSchema{
					Name:   val.Name,
					Fields: m,
				},
			},
		}, nil
	case FileSchema:
		return &providerpb.FieldSchema{
			Type: &providerpb.FieldSchema_FileSchema{
				FileSchema: &providerpb.FileSchema{},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid type for schema: %T", f)
	}
}
