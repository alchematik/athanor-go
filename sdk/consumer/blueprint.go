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

func (b Blueprint) WithResource(exists any, provider Provider, resource Resource) Blueprint {
	b.stmts = append(b.stmts, resourceStmt{
		exists:   exists,
		provider: provider,
		resource: resource,
	})
	return b
}

func (b Blueprint) WithBuild(alias string, repo Source, translator Translator, runtimeConfig any, config ...any) Blueprint {
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
	repo          Source
	translator    Translator
	configs       []any
	runtimeConfig any
}

type Translator struct {
	Source Source
	Name   string
}

type resourceStmt struct {
	exists   any
	resource Resource
	provider Provider
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
	Identifier any
	Config     any
}

type Provider struct {
	Name   string
	Source Source
}

type Source interface {
	isRepo()
}

type SourceFilePath struct {
	Source

	Path string
}

type SourceGitHubRelease struct {
	Source

	RepoOwner string
	RepoName  string
	Name      string
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

	fmt.Printf("BUILDING BLUEPRINT >>>> %v\n", configPath)
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
			res, err := toResourceExprProto(s.resource)
			if err != nil {
				log.Fatalf("error converting statement into proto: %v", err)
			}

			exists, err := toExprProto(s.exists)
			if err != nil {
				log.Fatalf("error converting statement into proto: %v", err)
			}

			provider, err := toProviderExprProto(s.provider)
			if err != nil {
				log.Fatalf("error converting statement into proto: %v", err)
			}

			p.Stmts = append(p.Stmts, &blueprintpb.Stmt{
				Type: &blueprintpb.Stmt_Resource{
					Resource: &blueprintpb.ResourceStmt{
						Exists:   exists,
						Resource: res,
						Provider: provider,
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

			r, err := sourceToProto(s.repo)
			if err != nil {
				log.Fatalf("error converting repo: %v", err)
			}

			tr, err := sourceToProto(s.translator.Source)
			if err != nil {
				log.Fatalf("error converting translator repo: %v", err)
			}

			p.Stmts = append(p.Stmts, &blueprintpb.Stmt{
				Type: &blueprintpb.Stmt_Build{
					Build: &blueprintpb.BuildStmt{
						Translator: &blueprintpb.Translator{
							Source: tr,
							Name:   s.translator.Name,
						},
						Build: &blueprintpb.BuildExpr{
							Alias:         s.alias,
							Source:        r,
							Config:        configs,
							RuntimeConfig: runtimeConfig,
						},
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

func sourceToProto(s Source) (*blueprintpb.Source, error) {
	switch s := s.(type) {
	case SourceFilePath:
		return &blueprintpb.Source{
			Type: &blueprintpb.Source_FilePath{
				FilePath: &blueprintpb.FilePathSource{
					Path: s.Path,
				},
			},
		}, nil
	case SourceGitHubRelease:
		return &blueprintpb.Source{
			Type: &blueprintpb.Source_GitHubRelease{
				GitHubRelease: &blueprintpb.GitHubReleaseSource{
					RepoOwner: s.RepoOwner,
					RepoName:  s.RepoName,
					Name:      s.Name,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid repo type: %v", s)
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
		res, err := toResourceExprProto(e)
		if err != nil {
			return nil, err
		}

		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Resource{
				Resource: res,
			},
		}, nil
	case Provider:
		p, err := toProviderExprProto(e)
		if err != nil {
			return nil, err
		}

		return &blueprintpb.Expr{
			Type: &blueprintpb.Expr_Provider{
				Provider: p,
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

func toResourceExprProto(res Resource) (*blueprintpb.ResourceExpr, error) {
	id, err := toExprProto(res.Identifier)
	if err != nil {
		return nil, err
	}

	config, err := toExprProto(res.Config)
	if err != nil {
		return nil, err
	}

	return &blueprintpb.ResourceExpr{
		Identifier: id,
		Config:     config,
	}, nil
}

func toProviderExprProto(p Provider) (*blueprintpb.ProviderExpr, error) {
	s, err := sourceToProto(p.Source)
	if err != nil {
		return nil, err
	}

	return &blueprintpb.ProviderExpr{
		Name:   p.Name,
		Source: s,
	}, nil
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
