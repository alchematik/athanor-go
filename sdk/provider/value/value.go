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
	if val == nil {
		return nil, nil
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected %T, got %T", map[string]T{}, val)
	}

	out := map[string]T{}
	for k, v := range m {
		if v == nil {
			continue
		}

		val, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("expected value for %s to be %T, got %T", k, val, v)
		}
		out[k] = val
	}

	return out, nil
}

func List[T any](val any) ([]T, error) {
	l, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("expected list, got %T", val)
	}

	out := make([]T, len(l))
	for i, e := range l {
		val, ok := e.(T)
		if !ok {
			return nil, fmt.Errorf("expected %T, got %T", val, e)
		}

		out[i] = val
	}

	return out, nil
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
	case *providerpb.Value_List:
		list := make([]any, len(v.List.Elements))
		for i, e := range v.List.Elements {
			var err error
			list[i], err = ParseProto(e)
			if err != nil {
				return nil, err
			}
		}
		return list, nil
	case *providerpb.Value_Nil:
		return nil, nil
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

type Immutable struct {
	Value any
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
			l[i] = convert(v[i])
		}
		return l
	default:
		return v
	}
}

func convert(val any) any {
	if v, ok := val.(ResourceType); ok {
		return v.ToValue()
	}

	if v, ok := val.(ResourceIdentifier); ok {
		return v.ToValue()
	}

	return val
}

func ToImmutableType(subTypeConvertFunc func(any) any) func(any) Immutable {
	return func(val any) Immutable {
		return Immutable{Value: subTypeConvertFunc(val)}
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
	case []any:
		p := make([]*providerpb.Value, len(v))
		for i, e := range v {
			var err error
			p[i], err = ToValueProto(e)
			if err != nil {
				return nil, err
			}
		}
		return &providerpb.Value{
			Type: &providerpb.Value_List{
				List: &providerpb.ListValue{
					Elements: p,
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
	case Immutable:
		value, err := ToValueProto(v.Value)
		if err != nil {
			return nil, err
		}

		return &providerpb.Value{
			Type: &providerpb.Value_Immutable{
				Immutable: &providerpb.Immutable{
					Value: value,
				},
			},
		}, nil
	case nil:
		return &providerpb.Value{
			Type: &providerpb.Value_Nil{},
		}, nil
	default:
		return nil, fmt.Errorf("invalid type: %T", val)
	}
}
