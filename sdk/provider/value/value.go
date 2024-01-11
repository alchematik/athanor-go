package value

import (
	"fmt"
	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
)

type ResourceField interface {
	ToValue() (Value, error)
}

type ResourceIdentifier interface {
	ResourceField

	ResourceType() string
}

type Resource interface {
	ToResourceValue() ResourceValue
}

type Value interface {
	ToValueProto() *providerpb.Value
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

func ParseProto(val *providerpb.Value) (Value, error) {
	switch v := val.GetType().(type) {
	case *providerpb.Value_StringValue:
		return String(v.StringValue), nil
	default:
		return nil, fmt.Errorf("unhandled proto type: %T", v)

	}
}

func ToValue(val any) (Value, error) {
	switch v := val.(type) {
	case string:
		return String(v), nil
	case bool:
		return Bool(v), nil
	case ResourceIdentifier:
		return v.ToValue()
	default:
		return nil, fmt.Errorf("cannot convert type %T to Value", val)
	}
}

type Bool bool

func (b Bool) ToValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_BoolValue{
			BoolValue: bool(b),
		},
	}
}

func ToStringValue(str string) String {
	return String(str)
}

func ParseStringValue(val Value) (string, error) {
	s, ok := val.(String)
	if !ok {
		return "", fmt.Errorf("not a string")
	}

	return string(s), nil
}

type String string

func (s String) ToValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_StringValue{
			StringValue: string(s),
		},
	}
}

func ParseMap(val Value) (map[string]Value, error) {
	m, ok := val.(Map)
	if !ok {
		return nil, fmt.Errorf("not a map")
	}

	return map[string]Value(m), nil
}

type Map map[string]Value

func (m Map) ToValueProto() *providerpb.Value {
	p := map[string]*providerpb.Value{}
	for k, v := range m {
		p[k] = v.ToValueProto()
	}

	return &providerpb.Value{
		Type: &providerpb.Value_Map{
			Map: &providerpb.MapValue{
				Entries: p,
			},
		},
	}
}

type Identifier struct {
	ResourceType string
	Value        Value
}

func (id Identifier) ToValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_Identifier{
			Identifier: &providerpb.Identifier{
				Type:  id.ResourceType,
				Value: id.Value.ToValueProto(),
			},
		},
	}
}

type ResourceValue struct {
	Identifier Value
	Config     Value
	Attrs      Value
}

func (r ResourceValue) ToResourceProto() *providerpb.Resource {
	return &providerpb.Resource{
		Identifier: r.Identifier.ToValueProto(),
		Config:     r.Config.ToValueProto(),
		Attrs:      r.Attrs.ToValueProto(),
	}
}
