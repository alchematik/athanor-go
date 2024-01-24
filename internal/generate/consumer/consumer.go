package consumer

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"sort"
	"text/template"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"

	util "github.com/alchematik/athanor-go/internal/generate/template"
)

//go:embed resource.tmpl
var resourceTmpl string

//go:embed struct_type.tmpl
var structTypeTmpl string

//go:embed identifier_struct_type.tmpl
var identifierStructTypeTmpl string

func GenerateResourceSrc(schema *providerpb.Schema, resource *providerpb.ResourceSchema) ([]byte, error) {
	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(resourceTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"PackageName": resource.GetType(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out := buf.Bytes()

	typesMap := map[string]*providerpb.FieldSchema{}

	id := resource.GetIdentifier()
	if id.GetStructSchema() != nil {
		id.GetStructSchema().Name = "identifier"
	}

	findStructs(typesMap, id)

	config := resource.GetConfig()
	if config.GetStructSchema() != nil {
		config.GetStructSchema().Name = "config"
	}

	findStructs(typesMap, config)

	var names []string
	for k := range typesMap {
		names = append(names, k)
	}

	sort.Strings(names)

	for _, name := range names {
		t := typesMap[name]
		switch t.GetType().(type) {
		case *providerpb.FieldSchema_StructSchema:
			if name == "identifier" {
				o, err := generateIdentifierStructType(resource.GetType(), t.GetStructSchema())
				if err != nil {
					return nil, err
				}

				out = append(out, o...)
			} else {
				o, err := generateStructType(resource.GetType(), t.GetStructSchema())
				if err != nil {
					return nil, err
				}

				out = append(out, o...)
			}
		default:
			return nil, fmt.Errorf("unsupported type: %s", t.GetType())
		}
	}

	return format.Source(out)
}

func findStructs(m map[string]*providerpb.FieldSchema, field *providerpb.FieldSchema) {
	switch t := field.GetType().(type) {
	case *providerpb.FieldSchema_StructSchema:
		m[t.StructSchema.GetName()] = field
		for _, v := range t.StructSchema.GetFields() {
			findStructs(m, v)
		}
	case *providerpb.FieldSchema_MapSchema:
		findStructs(m, t.MapSchema.GetValue())
	case *providerpb.FieldSchema_ListSchema:
		findStructs(m, t.ListSchema.GetElement())
	}
}

func generateStructType(name string, t *providerpb.StructSchema) ([]byte, error) {
	tmpl, err := template.New("struct_type").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(structTypeTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"ResourceName": name,
		"Type":         t,
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func generateIdentifierStructType(name string, t *providerpb.StructSchema) ([]byte, error) {
	tmpl, err := template.New("identifier_struct_type").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(identifierStructTypeTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"ResourceName": name,
		"Type":         t,
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
