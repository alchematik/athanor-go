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

type File struct {
	Path string
}

type ResourceIdentifier struct {
	Alias        string
	ResourceType string
	Value        any
}

type Resource struct {
	Exists     any
	Provider   any
	Identifier any
	Config     any
}

type Provider struct {
	Name    string
	Version string
}

type Get struct {
	Name   string
	Object any
}

func (g Get) Get(name string) Get {
	return Get{
		Name:   name,
		Object: g,
	}
}

func GetResource(alias string) Get {
	return Get{
		Name: alias,
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
	case []any:
		p := make([]*blueprintpb.Expr, len(e))
		for i, val := range e {
			var err error
			p[i], err = toExprProto(val)
			if err != nil {
				return nil, err
			}
		}
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_List{
				List: &blueprintpb.ListExpr{
					Elements: p,
				},
			},
		}, nil
	case File:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_File{
				File: &blueprintpb.FileExpr{
					Path: e.Path,
				},
			},
		}, nil
	case ResourceIdentifier:
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
	case Resource:
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
	case Provider:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Provider{
				Provider: &blueprintpb.ProviderExpr{
					Name:    e.Name,
					Version: e.Version,
				},
			},
		}, nil
	case Get:
		obj, err := toExprProto(e.Object)
		if err != nil {
			return nil, err
		}
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Get{
				Get: &blueprintpb.GetExpr{
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
		return nil, fmt.Errorf("unsupported expression: %T", expr)
	}
}
