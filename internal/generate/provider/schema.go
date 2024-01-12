package provider

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed template/resource.tmpl
var resourceHeaderTmpl string

//go:embed template/struct_type.tmpl
var structTypeTmpl string

//go:embed template/provider.tmpl
var providerTmpl string

func GenerateProvider(typeNames []string) ([]byte, error) {
	tmpl, err := template.New("provider").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(providerTmpl)
	if err != nil {
		return nil, err
	}

	sort.Strings(typeNames)

	providerData := map[string]any{
		"PackageName": "provider",
		"Types":       typeNames,
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

func GenerateResource(name string, types []*providerpb.FieldSchema) ([]byte, error) {
	var out []byte

	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(resourceHeaderTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"PackageName": "provider",
		"Type":        toPascalCase(name),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out = append(out, buf.Bytes()...)
	for _, t := range types {
		switch t.GetType() {
		case providerpb.FieldType_STRUCT:
			o, err := generateStructType(name, t)
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

func generateStructType(name string, t *providerpb.FieldSchema) ([]byte, error) {
	tmpl, err := template.New("struct_type").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
			"parseFieldFunc": func(f *providerpb.FieldSchema) (string, error) {
				if f.IsIdentifier {
					return "ParseIdentifier", nil
				}
				switch f.GetType() {
				case providerpb.FieldType_STRING:
					return "sdk.ParseStringValue", nil
				case providerpb.FieldType_STRUCT:
					return fmt.Sprintf("Parse%s", toPascalCase(f.Name)), nil
				default:
					return "", fmt.Errorf("unsupported type %s", f.GetType())
				}
			},
			"toType": func(f *providerpb.FieldSchema) (string, error) {
				if f.GetIsIdentifier() {
					return "sdk.ResourceIdentifier", nil
				}

				switch f.GetType() {
				case providerpb.FieldType_STRING:
					return "string", nil
				case providerpb.FieldType_STRUCT:
					return toPascalCase(f.GetName()), nil
				default:
					return "", fmt.Errorf("unrecognized type: %s", f.GetType())
				}
			},
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

func toPascalCase(str string) string {
	titleCaser := cases.Title(language.Und)
	upperCaser := cases.Upper(language.Und)
	splitter := func(r rune) bool {
		return r == '_' || r == ' '
	}
	parts := strings.FieldsFunc(str, splitter)
	if len(parts) == 1 {
		part := parts[0]
		if upperCaser.String(part) == part {
			return part
		}

		return titleCaser.String(parts[0])
	}

	var transformed []string
	for _, part := range parts {
		if upperCaser.String(part) == part {
			transformed = append(transformed, part)
			continue
		}

		transformed = append(transformed, titleCaser.String(part))
	}

	return strings.Join(transformed, "")
}

func findStructFields(field *providerpb.FieldSchema) []*providerpb.FieldSchema {
	// Use map?
	// Put all structs across resources in same package?
	var structFields []*providerpb.FieldSchema
	for _, f := range field.GetFields() {
		if f.GetType() == providerpb.FieldType_STRUCT {
			structFields = append(structFields, f)
		}
	}

	return structFields
}
