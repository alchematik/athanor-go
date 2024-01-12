package consumer

import (
	"bytes"
	_ "embed"
	"go/format"
	"text/template"

	providerpb "github.com/alchematik/athanor-go/internal/gen/go/proto/provider/v1"

	util "github.com/alchematik/athanor-go/internal/generate/template"
)

//go:embed resource.tmpl
var resourceTmpl string

func GenerateResource(resource *providerpb.ResourceSchema) ([]byte, error) {
	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{
			"toPascalCase": util.PascalCase,
		}).
		Parse(resourceTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return src, nil
}
