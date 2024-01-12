package consumer

import (
	_ "embed"
)

//go:embed template/resource.tmpl
var resourceTmpl string

// func GenerateResource(name)
