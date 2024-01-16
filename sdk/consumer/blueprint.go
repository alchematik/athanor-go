package sdk

import (
	"encoding/json"
	"fmt"
	"os"

	blueprintpb "github.com/alchematik/athanor-go/internal/gen/go/proto/blueprint/v1"
)

type Blueprint struct {
	stmts []any
}

func (b Blueprint) WithResource(r any) Blueprint {
	b.stmts = append(b.stmts, resourceStmt{
		resource: r,
	})
	return b
}

type resourceStmt struct {
	resource any
}

type FileType struct {
	Path string
}

func ResourceIdentifier(resourceType, alias string, value any) ResourceIdentifierType {
	return ResourceIdentifierType{
		ResourceType: resourceType,
		Alias:        alias,
		Value:        value,
	}
}

type ResourceIdentifierType struct {
	Alias        string
	ResourceType string
	Value        any
}

func Resource(exists, provider, identifier, config any) ResourceType {
	return ResourceType{
		Exists:     exists,
		Provider:   provider,
		Identifier: identifier,
		Config:     config,
	}
}

type ResourceType struct {
	Exists     any
	Provider   any
	Identifier any
	Config     any
}

func Provider(name, version string) ProviderType {
	return ProviderType{
		Name:    name,
		Version: version,
	}
}

type ProviderType struct {
	Name    string
	Version string
}

func IOGet(name string, object any) IOGetType {
	return IOGetType{
		Name:   name,
		Object: object,
	}
}

type IOGetType struct {
	Name   string
	Object any
}

func (g IOGetType) IOGet(name string) IOGetType {
	return IOGetType{
		Name:   name,
		Object: g,
	}
}

func Build(bp Blueprint) error {
	outputPath := os.Args[1]

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	p := &blueprintpb.Blueprint{}
	for _, stmt := range bp.stmts {
		switch s := stmt.(type) {
		case resourceStmt:
			res, err := toExprProto(s.resource)
			if err != nil {
				return err
			}

			p.Stmts = append(p.Stmts, &blueprintpb.Stmt{
				Type: &blueprintpb.Stmt_Resource{
					Resource: &blueprintpb.ResourceStmt{
						Expr: res,
					},
				},
			})
		default:
			return fmt.Errorf("invalid statement type: %T", stmt)
		}
	}

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}

type exprConvertable interface {
	ToExpr() any
}

func toExprProto(expr any) (*blueprintpb.Expr, error) {
	switch e := expr.(type) {
	case string:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_StringLiteral{
				StringLiteral: e,
			},
		}, nil
	case bool:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_BoolLiteral{
				BoolLiteral: bool(e),
			},
		}, nil
	case map[string]any:
		p := map[string]*blueprintpb.Expr{}
		for k, v := range e {
			var err error
			p[k], err = toExprProto(v)
			if err != nil {
				return nil, err
			}
		}

		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Map{
				Map: &blueprintpb.MapExpr{
					Entries: p,
				},
			},
		}, nil
	case FileType:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_File{
				File: &blueprintpb.FileExpr{
					Path: e.Path,
				},
			},
		}, nil
	case ResourceIdentifierType:
		val, err := toExprProto(e.Value)
		if err != nil {
			return nil, err
		}
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_ResourceIdentifier{
				ResourceIdentifier: &blueprintpb.ResourceIdentifierExpr{
					Alias: e.Alias,
					Type:  e.ResourceType,
					Value: val,
				},
			},
		}, nil
	case ResourceType:
		provider, err := toExprProto(e.Provider)
		if err != nil {
			return nil, err
		}

		id, err := toExprProto(e.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := toExprProto(e.Config)
		if err != nil {
			return nil, err
		}

		exists, err := toExprProto(e.Exists)
		if err != nil {
			return nil, err
		}

		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Resource{
				Resource: &blueprintpb.ResourceExpr{
					Provider:   provider,
					Identifier: id,
					Config:     config,
					Exists:     exists,
				},
			},
		}, nil
	case ProviderType:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Provider{
				Provider: &blueprintpb.ProviderExpr{
					Name:    e.Name,
					Version: e.Version,
				},
			},
		}, nil
	case IOGetType:
		obj, err := toExprProto(e.Object)
		if err != nil {
			return nil, err
		}
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_IoGet{
				IoGet: &blueprintpb.IOGetExpr{
					Name:   e.Name,
					Object: obj,
				},
			},
		}, nil
	case exprConvertable:
		return toExprProto(e.ToExpr())
	case nil:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Nil{},
		}, nil
	default:
		return nil, fmt.Errorf("unsuppoprted expression: %T", expr)
	}
}
