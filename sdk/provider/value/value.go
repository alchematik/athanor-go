package value

import (
	"fmt"
	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
)

type ResourceIdentifier interface {
	ResourceType() string
	ToValue() Identifier
}

func Map[T any](val any) (map[string]T, error) {
	m, ok := val.(map[string]T)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", val)
	}

	return m, nil
}

func List[T any](val any) ([]T, error) {
	l, ok := val.([]T)
	if !ok {
		return nil, fmt.Errorf("expected list, got %T", val)
	}

	return l, nil
}

func String(val any) (string, error) {
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", val)
	}

	return s, nil
}

func Bool(val any) (bool, error) {
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("expected bool, got %T", val)
	}

	return b, nil
}

func ParseIdentifier(val any) (Identifier, error) {
	id, ok := val.(Identifier)
	if !ok {
		return Identifier{}, fmt.Errorf("expected Identifier, got %T", val)
	}

	return id, nil
}

func ParseFile(val any) (File, error) {
	f, ok := val.(File)
	if !ok {
		return File{}, fmt.Errorf("expected File, got %T", val)
	}

	return f, nil
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
	case *providerpb.Value_File:
		return File{
			Path:     v.File.Path,
			Checksum: v.File.Checksum,
		}, nil
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

type UpdateMaskField struct {
	Name      string
	SubFields []UpdateMaskField
	Operation Operation
}

type File struct {
	Path     string
	Checksum string
}

type Operation string

const (
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

type ResourceType interface {
	ToValue() any
}

type Resource struct {
	Identifier Identifier
	Config     any
	Attrs      any
}

func (r Resource) ToResourceProto() (*providerpb.Resource, error) {
	idProto, err := ToValueProto(r.Identifier)
	if err != nil {
		return nil, err
	}

	configProto, err := ToValueProto(r.Config)
	if err != nil {
		return nil, err
	}

	attrsProto, err := ToValueProto(r.Attrs)
	if err != nil {
		return nil, err
	}

	return &providerpb.Resource{
		Identifier: idProto,
		Config:     configProto,
		Attrs:      attrsProto,
	}, nil
}

func ToType[T any](val any) any {
	switch v := val.(type) {
	case ResourceIdentifier:
		return v.ToValue()
	case ResourceType:
		return v.ToValue()
	case map[string]T:
		m := map[string]any{}
		for k, v := range v {
			m[k] = v
		}
		return m
	case []T:
		l := make([]any, len(v))
		for i := range v {
			l[i] = v[i]
		}
		return l
	default:
		return v
	}
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
	// case ResourceType:
	// 	return ToValueProto(v.ToValue())
	case File:
		return &providerpb.Value{
			Type: &providerpb.Value_File{
				File: &providerpb.FileValue{
					Path:     v.Path,
					Checksum: v.Checksum,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid type: %T", v)
	}
}
