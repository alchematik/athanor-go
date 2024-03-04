package sdk

import (
	"encoding/json"
	"fmt"
	"log"
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

func (b Blueprint) WithBuild(alias string, repo Repo, translator Translator, runtimeConfig any, config ...any) Blueprint {
	b.stmts = append(b.stmts, buildStmt{
		alias:         alias,
		repo:          repo,
		translator:    translator,
		configs:       config,
		runtimeConfig: runtimeConfig,
	})

	return b
}

type buildStmt struct {
	alias         string
	repo          Repo
	translator    Translator
	configs       []any
	runtimeConfig any
}

type Translator struct {
	Repo    Repo
	Name    string
	Version string
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
	Repo    Repo
}

type Repo interface {
	isRepo()
}

type RepoLocal struct {
	Repo

	Path string
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

type RuntimeConfig struct{}

type BlueprintFunc func(args ...any) (Blueprint, error)

func Build(bf BlueprintFunc) {
	configPath := os.Args[1]
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("error opening config file: %v", err)
	}

	configs := []*blueprintpb.Expr{}
	if err := json.Unmarshal(configData, &configs); err != nil {
		log.Fatalf("error unmarshaling config: %v", err)
	}

	configExprs := make([]any, len(configs))
	for i, c := range configs {
		config, err := fromProtoToExpr(c)
		if err != nil {
			log.Fatalf("error converting config: %v", err)
		}

		configExprs[i] = config
	}

	bp, err := bf(configExprs...)
	if err != nil {
		log.Fatalf("error building blueprint: %v", err)
	}

	outputPath := os.Args[2]

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Fatalf("error opening output path: %v", err)
	}

	p := &blueprintpb.Blueprint{}
	for _, stmt := range bp.stmts {
		switch s := stmt.(type) {
		case resourceStmt:
			res, err := toExprProto(s.resource)
			if err != nil {
				log.Fatalf("error converting statement into proto: %v", err)
			}

			p.Stmts = append(p.Stmts, &blueprintpb.Stmt{
				Type: &blueprintpb.Stmt_Resource{
					Resource: &blueprintpb.ResourceStmt{
						Expr: res,
					},
				},
			})
		case buildStmt:
			configs := make([]*blueprintpb.Expr, len(s.configs))
			for i, c := range s.configs {
				config, err := toExprProto(c)
				if err != nil {
					log.Fatalf("error converting build config: %v", err)
				}

				configs[i] = config
			}

			runtimeConfig, err := toExprProto(s.runtimeConfig)
			if err != nil {
				log.Fatalf("error converting runtime config: %v", err)
			}

			r, err := repoToProto(s.repo)
			if err != nil {
				log.Fatalf("error converting repo: %v", err)
			}

			tr, err := repoToProto(s.translator.Repo)
			if err != nil {
				log.Fatalf("error converting translator repo: %v", err)
			}

			p.Stmts = append(p.Stmts, &blueprintpb.Stmt{
				Type: &blueprintpb.Stmt_Build{
					Build: &blueprintpb.BuildStmt{
						Alias: s.alias,
						Repo:  r,
						Translator: &blueprintpb.Translator{
							Repo:    tr,
							Name:    s.translator.Name,
							Version: s.translator.Version,
						},
						Config:        configs,
						RuntimeConfig: runtimeConfig,
					},
				},
			})
		default:
			log.Fatalf("invalid statement type: %T", stmt)
		}
	}

	data, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("error marshaling blueprint: %v", err)
	}

	if _, err := f.Write(data); err != nil {
		log.Fatalf("error writing blueprint to file: %v", err)
	}
}

func repoToProto(r Repo) (*blueprintpb.Repo, error) {
	switch r := r.(type) {
	case RepoLocal:
		return &blueprintpb.Repo{
			Type: &blueprintpb.Repo_Local{
				Local: &blueprintpb.LocalRepo{
					Path: r.Path,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid repo type: %v", r)
	}
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
		var repo *blueprintpb.Repo
		switch r := e.Repo.(type) {
		case RepoLocal:
			repo = &blueprintpb.Repo{
				Type: &blueprintpb.Repo_Local{
					Local: &blueprintpb.LocalRepo{
						Path: r.Path,
					},
				},
			}
		default:
			return nil, fmt.Errorf("invalid repo type: %G", e.Repo)
		}

		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Provider{
				Provider: &blueprintpb.ProviderExpr{
					Name:    e.Name,
					Version: e.Version,
					Repo:    repo,
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
	case RuntimeConfig:
		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_GetRuntimeConfig{
				GetRuntimeConfig: &blueprintpb.GetRuntimeConfig{},
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

func fromProtoToExpr(p *blueprintpb.Expr) (any, error) {
	switch t := p.GetType().(type) {
	case *blueprintpb.Expr_StringLiteral:
		return t.StringLiteral, nil
	case *blueprintpb.Expr_BoolLiteral:
		return t.BoolLiteral, nil
	case *blueprintpb.Expr_List:
		l := make([]any, len(t.List.GetElements()))
		for i, e := range t.List.GetElements() {
			converted, err := fromProtoToExpr(e)
			if err != nil {
				return nil, err
			}

			l[i] = converted
		}

		return l, nil
	case *blueprintpb.Expr_Map:
		m := map[string]any{}
		for k, v := range t.Map.GetEntries() {
			converted, err := fromProtoToExpr(v)
			if err != nil {
				return nil, err
			}

			m[k] = converted
		}

		return m, nil
	case *blueprintpb.Expr_Nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid expr type: %T", p)
	}
}
