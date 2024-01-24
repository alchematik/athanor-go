package provider

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"path/filepath"
	"sort"
	"text/template"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"
	util "github.com/alchematik/athanor-go/internal/generate/template"
)

//go:embed resource.tmpl
var resourceTmpl string

//go:embed struct_type.tmpl
var structTypeTmpl string

//go:embed provider.tmpl
var providerTmpl string

//go:embed identifier.tmpl
var identifierTmpl string

//go:embed identifier_struct.tmpl
var identifierStructTmpl string

func GenerateProviderCommonSrc(module string, outputPath string, schema *providerpb.Schema) ([]byte, error) {
	resources := schema.GetResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].GetType() < resources[j].GetType()
	})

	resourceNames := make([]string, len(resources))
	for i, r := range resources {
		resourceNames[i] = r.GetType()
	}

	tmpl, err := template.New("provider").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(providerTmpl)
	if err != nil {
		return nil, err
	}

	providerData := map[string]any{
		"PackageName":  "identifier",
		"Types":        resourceNames,
		"ImportPrefix": filepath.Join(module, outputPath),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, providerData); err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return src, nil
}

func GenerateIdentifierSrc(resource *providerpb.ResourceSchema) ([]byte, error) {
	tmpl, err := template.New("identifier").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(identifierTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"PackageName": "identifier",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out := buf.Bytes()
	idName := fmt.Sprintf("%s_identifier", resource.GetType())

	// Override name to "identifier" if struct.
	id := resource.GetIdentifier()
	if id.GetStructSchema() != nil {
		id.GetStructSchema().Name = idName
	}

	typesMap := map[string]*providerpb.FieldSchema{
		idName: id,
	}
	findStructs(typesMap, resource.GetIdentifier())

	var names []string
	for k := range typesMap {
		names = append(names, k)
	}

	sort.Strings(names)

	resourceTypes := make([]*providerpb.FieldSchema, len(names))

	for i, name := range names {
		resourceTypes[i] = typesMap[name]
	}

	for _, t := range resourceTypes {
		switch t.GetType().(type) {
		case *providerpb.FieldSchema_StructSchema:
			o, err := generateIdentifierStructType(resource.GetType(), t.GetStructSchema())
			if err != nil {
				return nil, err
			}

			out = append(out, o...)
		default:
			return nil, fmt.Errorf("unsupported type: %s", t.GetType())
		}
	}

	return format.Source(out)
}

func GenerateResourceSrc(module, outputPath string, resource *providerpb.ResourceSchema) ([]byte, error) {
	name := resource.GetType()

	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(resourceTmpl)
	if err != nil {
		return nil, err
	}

	imports := []string{
		`"context"`,
		`"fmt"`,
		`sdk "github.com/alchematik/athanor-go/sdk/provider/value"`,
		fmt.Sprintf("\"%s\"", filepath.Join(module, outputPath, "identifier")),
	}

	data := map[string]any{
		"PackageName": name,
		"Type":        util.PascalCase(name),
		"Imports":     imports,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out := buf.Bytes()

	typesMap := map[string]*providerpb.FieldSchema{}

	config := resource.GetConfig()
	if config.GetStructSchema() != nil {
		config.GetStructSchema().Name = "config"
	}

	findStructs(typesMap, config)

	attrs := resource.GetAttrs()
	if attrs.GetStructSchema() != nil {
		attrs.GetStructSchema().Name = "attrs"
	}

	findStructs(typesMap, attrs)

	var names []string
	for k := range typesMap {
		names = append(names, k)
	}

	sort.Strings(names)

	resourceTypes := make([]*providerpb.FieldSchema, len(names))

	for i, name := range names {
		resourceTypes[i] = typesMap[name]
	}

	for _, t := range resourceTypes {
		switch t.GetType().(type) {
		case *providerpb.FieldSchema_StructSchema:
			o, err := generateStructType(t.GetStructSchema())
			if err != nil {
				return nil, err
			}

			out = append(out, o...)
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

func generateStructType(t *providerpb.StructSchema) ([]byte, error) {
	tmpl, err := template.New("struct_type").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
			"parseFieldFunc": func(f *providerpb.FieldSchema) (string, error) {
				switch val := f.GetType().(type) {
				case *providerpb.FieldSchema_StringSchema:
					return "sdk.String", nil
				case *providerpb.FieldSchema_StructSchema:
					return fmt.Sprintf("Parse%s", util.PascalCase(val.StructSchema.GetName())), nil
				case *providerpb.FieldSchema_MapSchema:
					subType, err := toType(val.MapSchema.GetValue())
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("sdk.Map[%s]", subType), nil
				case *providerpb.FieldSchema_ListSchema:
					subType, err := toType(val.ListSchema.GetElement())
					if err != nil {
						return "", err
					}

					return fmt.Sprintf("sdk.List[%s]", subType), nil
				case *providerpb.FieldSchema_FileSchema:
					return "sdk.ParseFile", nil
				case *providerpb.FieldSchema_IdentifierSchema:
					return "identifier.ParseIdentifier", nil
				case *providerpb.FieldSchema_BoolSchema:
					return "sdk.Bool", nil
				default:
					return "", fmt.Errorf("unsupported type %T", f.GetType())
				}
			},
			"toType":     toType,
			"toTypeFunc": toTypeFunc,
		}).
		Parse(structTypeTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"Type": t,
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
			"parseFieldFunc": func(f *providerpb.FieldSchema) (string, error) {
				switch val := f.GetType().(type) {
				case *providerpb.FieldSchema_StringSchema:
					return "sdk.String", nil
				case *providerpb.FieldSchema_StructSchema:
					return fmt.Sprintf("Parse%s", util.PascalCase(val.StructSchema.GetName())), nil
				case *providerpb.FieldSchema_MapSchema:
					subType, err := toType(val.MapSchema.GetValue())
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("sdk.Map[%s]", subType), nil
				case *providerpb.FieldSchema_ListSchema:
					subType, err := toType(val.ListSchema.GetElement())
					if err != nil {
						return "", err
					}

					return fmt.Sprintf("sdk.List[%s]", subType), nil
				case *providerpb.FieldSchema_FileSchema:
					return "sdk.ParseFile", nil
				case *providerpb.FieldSchema_IdentifierSchema:
					return "ParseIdentifier", nil
				case *providerpb.FieldSchema_BoolSchema:
					return "sdk.Bool", nil
				default:
					return "", fmt.Errorf("unsupported type %s", f.GetType())
				}
			},
			"toType":     toType,
			"toTypeFunc": toTypeFunc,
		}).
		Parse(identifierStructTmpl)
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

func toType(f *providerpb.FieldSchema) (string, error) {
	switch val := f.GetType().(type) {
	case *providerpb.FieldSchema_StringSchema:
		return "string", nil
	case *providerpb.FieldSchema_StructSchema:
		return util.PascalCase(val.StructSchema.GetName()), nil
	case *providerpb.FieldSchema_MapSchema:
		valType, err := toType(val.MapSchema.GetValue())
		if err != nil {
			return "", err
		}
		return "map[string]" + valType, nil
	case *providerpb.FieldSchema_ListSchema:
		valType, err := toType(val.ListSchema.GetElement())
		if err != nil {
			return "", err
		}
		return "[]" + valType, nil
	case *providerpb.FieldSchema_FileSchema:
		return "sdk.File", nil
	case *providerpb.FieldSchema_IdentifierSchema:
		return "sdk.ResourceIdentifier", nil
	case *providerpb.FieldSchema_BoolSchema:
		return "bool", nil
	default:
		return "", fmt.Errorf("unrecognized type: %s", f.GetType())
	}
}

func toTypeFunc(f *providerpb.FieldSchema) (string, error) {
	switch val := f.GetType().(type) {
	case *providerpb.FieldSchema_ListSchema:
		subType, err := toType(val.ListSchema.GetElement())
		if err != nil {
			return "", err

		}
		return fmt.Sprintf("sdk.ToType[%s]", subType), err
	case *providerpb.FieldSchema_MapSchema:
		subType, err := toType(val.MapSchema.GetValue())
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("sdk.ToType[%s]", subType), nil
	default:
		return "sdk.ToType[any]", nil
	}
}
