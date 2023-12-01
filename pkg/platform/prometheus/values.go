package prometheus

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"code.byted.org/bge-infra/metrics-gen/pkg/parse"
	"code.byted.org/bge-infra/metrics-gen/pkg/platform"
	"code.byted.org/bge-infra/metrics-gen/pkg/utils"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type prometheusProvider struct {
	inplace bool
	suffix  string
	dryRun  bool
}

const (
	// default prometheus port
	defaultPromPort = "9123"
	defaultPromPath = "/metrics-gen"
)

var (
	pkgsInitFuncRequired = map[string]*parse.PackageInfo{
		"http": {Name: "http", Path: "net/http"},
		"prometheus": {
			Name: "prometheus",
			Path: "github.com/prometheus/client_golang/prometheus",
		},
		"promhttp": {
			Name: "promhttp",
			Path: "github.com/prometheus/client_golang/prometheus/promhttp",
		},
		"globalvar": {Name: "globalvar", Path: "github.com/wilsonwang371/globalvar/pkg"},
	}

	pkgsTraceRequired = map[string]*parse.PackageInfo{
		"time": {Name: "time", Path: "time"},
		"sync": {Name: "sync", Path: "sync"},
		"prometheus": {
			Name: "prometheus",
			Path: "github.com/prometheus/client_golang/prometheus",
		},
		"globalvar": {Name: "globalvar", Path: "github.com/wilsonwang371/globalvar/pkg"},
		// "promauto":   {"promauto", "github.com/prometheus/client_golang/prometheus/promauto"},
	}

	pkgsNeedDownload = []string{
		"github.com/prometheus/client_golang/prometheus",
		"github.com/wilsonwang371/globalvar",
		// "github.com/prometheus/client_golang/prometheus/promauto",
		// "github.com/prometheus/client_golang/prometheus/promhttp",
	}
)

func NewPrometheusProvider(inplace bool, suffix string,
	dryRun bool,
) platform.MetricsProvider {
	return &prometheusProvider{
		inplace: inplace,
		suffix:  suffix,
		dryRun:  dryRun,
	}
}

func (p *prometheusProvider) PrePatch(d *parse.CollectInfo) error {
	if !d.HasDefinitionDirective() {
		return fmt.Errorf("no definition directive found")
	}
	return nil
}

func (p *prometheusProvider) Patch(d *parse.CollectInfo) error {
	for _, fullpath := range d.Files() {
		directives, err := d.FileDirectives(fullpath)
		if err != nil {
			return err
		}
		for _, directive := range directives {
			base := filepath.Base(
				fullpath,
			) // Get the base (filename) from the full path
			filename := base[:len(base)-len(filepath.Ext(base))] // Remove the extension
			if directive.TraceType() == parse.Define {
				initDst, patchTable := globalInitFuncDst(d, directive)
				if err := d.SetGlobalDefineFunc(*directive, initDst,
					pkgsInitFuncRequired, patchTable); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.FuncExecTime {
				// add function execution time metric
				globalDecl, inFuncStmts, patchTable := funcTraceStmtsDst(filename,
					directive.Declaration().(*dst.FuncDecl).Name.Name, directive)
				if err := d.SetFunctionTimeTracing(*directive, globalDecl,
					inFuncStmts, pkgsTraceRequired, patchTable); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.InnerExecTime {
				// TODO: implement inner execution time
				// panic("not implemented")
			} else if directive.TraceType() == parse.GenBegine ||
				directive.TraceType() == parse.GenEnd {
				return fmt.Errorf("metrics code already generated")
			}
		}
	}
	return nil
}

func (p *prometheusProvider) PostPatch(d *parse.CollectInfo) error {
	for _, filename := range d.Files() {
		if !d.IsModified(filename) {
			continue
		}

		fDst := d.FileDst(filename)

		var newFilename string
		if p.inplace {
			newFilename = filename
		} else {
			newFilename = utils.NewFilenameForTracing(filename, p.suffix)
		}
		// put new content into a buffer
		var buf bytes.Buffer
		if err := decorator.Fprint(&buf, fDst); err != nil {
			return err
		}

		log.Infof("writing to %s", newFilename)
		if p.dryRun {
			continue
		}

		// write to file
		f, err := os.Create(newFilename)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.Write(buf.Bytes()); err != nil {
			return err
		}
	}

	return p.dowloadNeededPackages(d)
}

func (p *prometheusProvider) dowloadNeededPackages(d *parse.CollectInfo) error {
	// download packages
	if p.dryRun {
		return nil
	}
	return utils.FetchPackages(d.GoModPath(), pkgsNeedDownload)
}

// get traced function execution duration declaration and statements
func funcTraceStmtsDst(filename string, funcname string,
	directive *parse.Directive,
) (globalDecl []dst.Decl, inFuncStmts []dst.Stmt, pkgsPatchTable []*dst.Ident) {
	g := []dst.Decl{}
	l := []dst.Stmt{}
	pkgsPatchTable = []*dst.Ident{}

	var varName string
	if val, ok := directive.Param("name"); ok {
		varName = val
		if varName == funcname {
			varName = fmt.Sprintf("fn_%s", funcname)
		}
	} else {
		varName = fmt.Sprintf("%s_%s_%s", filename, funcname, "duration")
	}

	// var historgram_initialized = false
	// var histogram_mutex sync.Mutex
	// var histogram = prometheus.NewHistogram(
	// 	prometheus.HistogramOpts{
	// 		Name: "my_histogram",
	// 		Help: "This is my histogram",
	// 	})
	g = []dst.Decl{
		&dst.GenDecl{
			Tok: token.VAR,
			Specs: []dst.Spec{
				&dst.ValueSpec{
					Names: []*dst.Ident{
						{Name: fmt.Sprintf("%s_initialized", varName)},
					},
					Type: &dst.Ident{Name: "bool"},
					Values: []dst.Expr{
						&dst.Ident{Name: "false"},
					},
				},
			},
		},
		&dst.GenDecl{
			Tok: token.VAR,
			Specs: []dst.Spec{
				&dst.ValueSpec{
					Names: []*dst.Ident{
						{Name: fmt.Sprintf("%s_mutex", varName)},
					},
					Type: &dst.SelectorExpr{
						X:   dst.NewIdent("sync"),
						Sel: dst.NewIdent("Mutex"),
					},
				},
			},
		},
		&dst.GenDecl{
			Tok: token.VAR,
			Specs: []dst.Spec{
				&dst.ValueSpec{
					Names: []*dst.Ident{
						{Name: fmt.Sprintf("%s", varName)},
					},
					Type: &dst.Ident{Name: "prometheus.Histogram"},
					Values: []dst.Expr{
						&dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   dst.NewIdent("prometheus"),
								Sel: dst.NewIdent("NewHistogram"),
							},
							Args: []dst.Expr{
								// construct a histogram options
								&dst.CompositeLit{
									Type: &dst.SelectorExpr{
										X:   dst.NewIdent("prometheus"),
										Sel: dst.NewIdent("HistogramOpts"),
									},
									Elts: []dst.Expr{
										&dst.KeyValueExpr{
											Key: dst.NewIdent("Name"),
											Value: &dst.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf("\"%s\"", varName),
											},
										},
										&dst.KeyValueExpr{
											Key: dst.NewIdent("Help"),
											Value: &dst.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf("\"%s\"", varName),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// add sync
	pkgsPatchTable = append(
		pkgsPatchTable,
		g[len(g)-2].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).
			Type.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add prometheus.Histogram
	pkgsPatchTable = append(
		pkgsPatchTable,
		g[len(g)-1].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).
			Type.(*dst.Ident),
	)
	// add 1st prometheus
	pkgsPatchTable = append(
		pkgsPatchTable,
		g[len(g)-1].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).
			Values[0].(*dst.CallExpr).Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add 2nd prometheus
	pkgsPatchTable = append(
		pkgsPatchTable,
		g[len(g)-1].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).
			Values[0].(*dst.CallExpr).Args[0].(*dst.CompositeLit).
			Type.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	// defer func(t time.Time) {
	// 	if !histogram_initialized {
	// 		histogram_mutex.Lock()
	// 		if !histogram_initialized {
	// 			reg, err := globalvar.Get("metrics-gen")
	// 			if err == nil {
	// 				histogram_initialized = true
	// 				reg.(*prometheus.Registry).MustRegister(histogram)
	// 			}
	// 		}
	// 		histogram_mutex.Unlock()
	// 	}
	// 	d := time.Since(t)
	// 	histogram.Observe(d.Milliseconds())
	// }(time.Now())
	l = append(l, &dst.DeferStmt{
		Call: &dst.CallExpr{
			Args: []dst.Expr{
				// time.Now()
				&dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent("time"),
						Sel: dst.NewIdent("Now"),
					},
				},
			},
			Fun: &dst.FuncLit{
				Type: &dst.FuncType{
					Params: &dst.FieldList{
						List: []*dst.Field{
							{
								Names: []*dst.Ident{
									dst.NewIdent("t"),
								},
								Type: &dst.Ident{Name: "time.Time"},
							},
						},
					},
				},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{
						&dst.IfStmt{
							Cond: &dst.UnaryExpr{
								Op: token.NOT,
								X: &dst.Ident{
									Name: fmt.Sprintf("%s_initialized", varName),
								},
							},
							Body: &dst.BlockStmt{
								List: []dst.Stmt{
									&dst.ExprStmt{
										X: &dst.CallExpr{
											Fun: &dst.SelectorExpr{
												X: dst.NewIdent(
													fmt.Sprintf("%s_mutex", varName),
												),
												Sel: dst.NewIdent("Lock"),
											},
										},
									},
									&dst.IfStmt{
										Cond: &dst.UnaryExpr{
											Op: token.NOT,
											X: &dst.Ident{
												Name: fmt.Sprintf(
													"%s_initialized",
													varName,
												),
											},
										},
										Body: &dst.BlockStmt{
											List: []dst.Stmt{
												&dst.AssignStmt{
													Lhs: []dst.Expr{
														dst.NewIdent("reg"),
														dst.NewIdent("err"),
													},
													Tok: token.DEFINE,
													Rhs: []dst.Expr{
														&dst.CallExpr{
															Fun: &dst.SelectorExpr{
																X: dst.NewIdent(
																	"globalvar",
																),
																Sel: dst.NewIdent("Get"),
															},
															Args: []dst.Expr{
																&dst.BasicLit{
																	Kind:  token.STRING,
																	Value: "\"metrics-gen\"",
																},
															},
														},
													},
												},
												&dst.IfStmt{
													Cond: &dst.BinaryExpr{
														X:  dst.NewIdent("err"),
														Op: token.EQL,
														Y:  dst.NewIdent("nil"),
													},
													Body: &dst.BlockStmt{
														List: []dst.Stmt{
															&dst.AssignStmt{
																Lhs: []dst.Expr{
																	dst.NewIdent(
																		fmt.Sprintf(
																			"%s_initialized",
																			varName,
																		),
																	),
																},
																Tok: token.ASSIGN,
																Rhs: []dst.Expr{
																	dst.NewIdent("true"),
																},
															},
															&dst.ExprStmt{
																X: &dst.CallExpr{
																	Fun: &dst.SelectorExpr{
																		X: dst.NewIdent(
																			"reg",
																		),
																		Sel: dst.NewIdent(
																			"(*prometheus.Registry).MustRegister",
																		),
																	},
																	Args: []dst.Expr{
																		dst.NewIdent(
																			varName,
																		),
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									&dst.ExprStmt{
										X: &dst.CallExpr{
											Fun: &dst.SelectorExpr{
												X: dst.NewIdent(
													fmt.Sprintf("%s_mutex", varName),
												),
												Sel: dst.NewIdent("Unlock"),
											},
										},
									},
								},
							},
						},
						&dst.AssignStmt{
							Lhs: []dst.Expr{dst.NewIdent("d")},
							Tok: token.DEFINE,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent("time"),
										Sel: dst.NewIdent("Since"),
									},
									Args: []dst.Expr{
										dst.NewIdent("t"),
									},
								},
							},
						},
						&dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent(varName),
									Sel: dst.NewIdent("Observe"),
								},
								Args: []dst.Expr{
									&dst.CallExpr{
										Fun: &dst.SelectorExpr{
											X:   dst.NewIdent("d"),
											Sel: dst.NewIdent("Seconds"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	// add arg time.Now
	pkgsPatchTable = append(
		pkgsPatchTable,
		l[len(l)-1].(*dst.DeferStmt).Call.Args[0].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add time.Time
	pkgsPatchTable = append(
		pkgsPatchTable,
		l[len(l)-1].(*dst.DeferStmt).Call.Fun.(*dst.FuncLit).
			Type.Params.List[0].Type.(*dst.Ident),
	)
	// add time.Since
	pkgsPatchTable = append(
		pkgsPatchTable,
		l[len(l)-1].(*dst.DeferStmt).Call.Fun.(*dst.FuncLit).
			Body.List[1].(*dst.AssignStmt).Rhs[0].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add 1st globalvar
	pkgsPatchTable = append(
		pkgsPatchTable,
		l[len(l)-1].(*dst.DeferStmt).Call.
			Fun.(*dst.FuncLit).Body.List[0].(*dst.IfStmt).
			Body.List[1].(*dst.IfStmt).
			Body.List[0].(*dst.AssignStmt).
			Rhs[0].(*dst.CallExpr).Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add (*prometheus.Registry).MustRegister
	pkgsPatchTable = append(
		pkgsPatchTable,
		l[len(l)-1].(*dst.DeferStmt).Call.Fun.(*dst.FuncLit).
			Body.List[0].(*dst.IfStmt).Body.List[1].(*dst.IfStmt).
			Body.List[1].(*dst.IfStmt).Body.List[1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	return g, l, pkgsPatchTable
}

func globalInitFuncDst(
	d *parse.CollectInfo,
	directive *parse.Directive,
) (*dst.FuncDecl, []*dst.Ident) {
	portNum := defaultPromPort
	if val, ok := directive.Param("prom-port"); ok {
		portNum = val
	}

	metricsRoute := fmt.Sprintf("\"%s\"", defaultPromPath)
	if val, ok := directive.Param("prom-route"); ok {
		metricsRoute = fmt.Sprintf("\"%s\"", val)
	}

	patchTable := []*dst.Ident{}

	stmts1 := []dst.Stmt{}
	// registry := prometheus.NewRegistry()
	stmts1 = append(stmts1, &dst.AssignStmt{
		Lhs: []dst.Expr{dst.NewIdent("registry")},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   dst.NewIdent("prometheus"),
					Sel: dst.NewIdent("NewRegistry"),
				},
			},
		},
	})
	// add prometheus
	patchTable = append(
		patchTable,
		stmts1[len(stmts1)-1].(*dst.AssignStmt).Rhs[0].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	// globalvar.Set("metrics-gen", registry)
	stmts1 = append(stmts1, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("globalvar"),
				Sel: dst.NewIdent("Set"),
			},
			Args: []dst.Expr{
				&dst.BasicLit{
					Kind:  token.STRING,
					Value: "\"metrics-gen\"",
				},
				dst.NewIdent("registry"),
			},
		},
	})
	// add globalvar
	patchTable = append(
		patchTable,
		stmts1[len(stmts1)-1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	stmts2 := []dst.Stmt{}
	// http.Handle("<route>", promhttp.HandlerFor(prometheus.Gatherers{
	// 	registry,
	// 	prometheus.DefaultGatherer,
	// }, promhttp.HandlerOpts{}))
	stmts2 = append(stmts2, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("http"),
				Sel: dst.NewIdent("Handle"),
			},
			Args: []dst.Expr{
				&dst.BasicLit{
					Kind:  token.STRING,
					Value: metricsRoute,
				},
				&dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent("promhttp"),
						Sel: dst.NewIdent("HandlerFor"),
					},
					Args: []dst.Expr{
						&dst.CompositeLit{
							Type: &dst.SelectorExpr{
								X:   dst.NewIdent("prometheus"),
								Sel: dst.NewIdent("Gatherers"),
							},
							Elts: []dst.Expr{
								dst.NewIdent("registry"),
								&dst.SelectorExpr{
									X:   dst.NewIdent("prometheus"),
									Sel: dst.NewIdent("DefaultGatherer"),
								},
							},
						},
						&dst.CompositeLit{
							Type: &dst.SelectorExpr{
								X:   dst.NewIdent("promhttp"),
								Sel: dst.NewIdent("HandlerOpts"),
							},
						},
					},
				},
			},
		},
	})
	// add http
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add promhttp
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).Args[1].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add 1st prometheus
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).Args[1].(*dst.CallExpr).
			Args[0].(*dst.CompositeLit).Type.(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add 2nd prometheus
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).Args[1].(*dst.CallExpr).
			Args[0].(*dst.CompositeLit).Elts[1].(*dst.SelectorExpr).X.(*dst.Ident),
	)
	// add promhttp
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).Args[1].(*dst.CallExpr).
			Args[1].(*dst.CompositeLit).Type.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	// http.ListenAndServe(":<port>", nil)
	stmts2 = append(stmts2, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   dst.NewIdent("http"),
				Sel: dst.NewIdent("ListenAndServe"),
			},
			Args: []dst.Expr{
				&dst.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf("\":%s\"", portNum),
				},
				dst.NewIdent("nil"),
			},
		},
	})
	// add http
	patchTable = append(
		patchTable,
		stmts2[len(stmts2)-1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	// put statements into a function that will be executed in a goroutine
	stmts2 = []dst.Stmt{
		&dst.GoStmt{
			Call: &dst.CallExpr{
				Fun: &dst.FuncLit{
					Type: &dst.FuncType{},
					Body: &dst.BlockStmt{
						List: stmts2,
					},
				},
			},
		},
	}
	// combine stmts1 and stmts2 into a function
	stmts2 = append(stmts1, stmts2...)
	return platform.DSTInitFunc(stmts2), patchTable
}
