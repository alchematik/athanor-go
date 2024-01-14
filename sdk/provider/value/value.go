package value

import (
	"fmt"
	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
)

type ResourceIdentifier interface {
	ResourceType() string
}

func Map(val any) (map[string]any, error) {
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", val)
	}

	return m, nil
}

func String(val any) (string, error) {
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", val)
	}

	return s, nil
}

func ParseProto(val *providerpb.Value) (any, error) {
	switch v := val.GetType().(type) {
	case *providerpb.Value_StringValue:
		return v.StringValue, nil
	case *providerpb.Value_BoolValue:
		return v.BoolValue, nil
	case *providerpb.Value_Map:
		m := map[string]any{}
		for k, v := range v.Map.GetEntries() {
			var err error
			m[k], err = ParseProto(v)
			if err != nil {
				return nil, err
			}
		}

		return m, nil
	case *providerpb.Value_Identifier:
		return ParseIdentifierProto(v.Identifier)
	default:
		return nil, fmt.Errorf("unhandled proto type: %T", v)
	}
}

func ParseIdentifierProto(id *providerpb.Identifier) (Identifier, error) {
	val, err := ParseProto(id.GetValue())
	if err != nil {
		return Identifier{}, err
	}

	return Identifier{
		ResourceType: id.Type,
		Value:        val,
	}, nil
}

type Identifier struct {
	ResourceType string
	Value        any
}

type ResourceType interface {
	ToValue() any
}

type Resource struct {
	Identifier any
	Config     any
	Attrs      any
}

func (r Resource) ToResourceProto() (*providerpb.Resource, error) {
	id, err := ToValueProto(r.Identifier)
	if err != nil {
		return nil, err
	}

	config, err := ToValueProto(r.Config)
	if err != nil {
		return nil, err
	}

	attrs, err := ToValueProto(r.Attrs)
	if err != nil {
		return nil, err
	}

	return &providerpb.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func ToValueProto(val any) (*providerpb.Value, error) {
	switch v := val.(type) {
	case string:
		return &providerpb.Value{
			Type: &providerpb.Value_StringValue{
				StringValue: string(v),
			},
		}, nil
	case bool:
		return &providerpb.Value{
			Type: &providerpb.Value_BoolValue{
				BoolValue: bool(v),
			},
		}, nil
	case map[string]any:
		p := map[string]*providerpb.Value{}
		for k, value := range v {
			converted, err := ToValueProto(value)
			if err != nil {
				return nil, err
			}
			p[k] = converted
		}

		return &providerpb.Value{
			Type: &providerpb.Value_Map{
				Map: &providerpb.MapValue{
					Entries: p,
				},
			},
		}, nil
	case Identifier:
		converted, err := ToValueProto(v.Value)
		if err != nil {
			return nil, err
		}
		return &providerpb.Value{
			Type: &providerpb.Value_Identifier{
				Identifier: &providerpb.Identifier{
					Type:  v.ResourceType,
					Value: converted,
				},
			},
		}, nil
	case ResourceType:
		return ToValueProto(v.ToValue())
	default:
		return nil, fmt.Errorf("invalid type: %T", v)
	}
}
