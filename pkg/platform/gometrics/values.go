package gometrics

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/parse"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/platform"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/utils"
)

var pkgsRequired = map[string]*parse.PackageInfo{
	"gometrics": {Name: "gometrics", Path: "github.com/hashicorp/go-metrics"},
	"time":      {Name: "time", Path: "time"},
}

type goMetricsProvider struct {
	inplace bool
	suffix  string
	dryRun  bool
}

func NewGoMetricsProvider(
	inplace bool,
	suffix string,
	dryRun bool,
) platform.MetricsProvider {
	return &goMetricsProvider{
		inplace: inplace,
		suffix:  suffix,
		dryRun:  dryRun,
	}
}

// PrePatch implements platform.MetricsProvider.
func (g *goMetricsProvider) PrePatch(info *parse.CollectInfo) error {
	if !info.HasDefinitionDirective() {
		return fmt.Errorf("no definition directive found")
	}
	return nil
}

// Patch implements platform.MetricsProvider.
func (g *goMetricsProvider) Patch(info *parse.CollectInfo) error {
	if err := PatchProject(info, g.dryRun); err != nil {
		return err
	}
	return nil
}

// PostPatch implements platform.MetricsProvider.
func (g *goMetricsProvider) PostPatch(info *parse.CollectInfo) error {
	if err := StoreFiles(info, g.inplace, g.suffix, g.dryRun); err != nil {
		return err
	}
	if err := UpdatePackages(info, g.dryRun); err != nil {
		return err
	}
	return nil
}

func TraceFuncTimeStmts(filename string, funcName string,
	directive *parse.Directive,
) (globalDecl []dst.Decl, inFuncStmts []dst.Stmt, identPatchTable []*dst.Ident) {
	cooldownTime := ""
	if v, ok := directive.Param("gm-cooldown-time"); ok {
		cooldownTime = v
		if _, err := time.ParseDuration(cooldownTime); err != nil {
			log.Fatalf("invalid gm-cooldown-time: %s, %s", err, cooldownTime)
		}
	}

	var varName string
	if v, ok := directive.Param("name"); ok {
		varName = v
		if varName == funcName {
			varName = fmt.Sprintf("fn_%s", funcName)
		}
	} else {
		varName = fmt.Sprintf(`%s_%s`, filename, funcName)
	}

	identPatchTable = []*dst.Ident{}
	g := []dst.Decl{}
	l := []dst.Stmt{}
	if cooldownTime != "" {
		cooldownTimeVarName, _ := timeConvertStatement("cooldown_time_", cooldownTime)
		g = []dst.Decl{
			&dst.GenDecl{
				Tok: token.VAR,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names: []*dst.Ident{
							{Name: fmt.Sprintf("lastInv_%s", funcName)},
						},
						Type: &dst.Ident{Name: "time.Time"},
						Values: []dst.Expr{
							&dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   &dst.Ident{Name: "time"},
									Sel: &dst.Ident{Name: "Now"},
								},
							},
						},
					},
				},
			},
			&dst.GenDecl{
				Tok: token.VAR,
				Specs: []dst.Spec{
					&dst.ValueSpec{
						Names: []*dst.Ident{
							{Name: cooldownTimeVarName},
							{Name: "_"},
						},
						Values: []dst.Expr{
							&dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   &dst.Ident{Name: "time"},
									Sel: &dst.Ident{Name: "ParseDuration"},
								},
								Args: []dst.Expr{
									&dst.BasicLit{
										Kind:  token.STRING,
										Value: fmt.Sprintf(`"%s"`, cooldownTime),
									},
								},
							},
						},
					},
				},
			},
		}
		// add time.Time
		identPatchTable = append(identPatchTable,
			g[0].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).Type.(*dst.Ident))
		// add 1st time
		identPatchTable = append(
			identPatchTable,
			g[0].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).Values[0].(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)
		// add 2nd time
		identPatchTable = append(
			identPatchTable,
			g[1].(*dst.GenDecl).Specs[0].(*dst.ValueSpec).Values[0].(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)

		l = []dst.Stmt{
			&dst.IfStmt{
				Cond: &dst.BinaryExpr{
					X: &dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   &dst.Ident{Name: "time"},
							Sel: &dst.Ident{Name: "Since"},
						},
						Args: []dst.Expr{
							&dst.Ident{Name: fmt.Sprintf("lastInv_%s", funcName)},
						},
					},
					Op: token.GTR,
					Y:  &dst.Ident{Name: cooldownTimeVarName},
				},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{
						&dst.AssignStmt{
							Lhs: []dst.Expr{
								&dst.Ident{Name: fmt.Sprintf("lastInv_%s", funcName)},
							},
							Tok: token.ASSIGN,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   &dst.Ident{Name: "time"},
										Sel: &dst.Ident{Name: "Now"},
									},
								},
							},
						},
						&dst.DeferStmt{
							Call: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   &dst.Ident{Name: "gometrics"},
									Sel: &dst.Ident{Name: "MeasureSince"},
								},
								Args: []dst.Expr{
									// value of []string{""}
									&dst.CompositeLit{
										Type: &dst.ArrayType{
											Elt: &dst.Ident{Name: "string"},
										},
										Elts: []dst.Expr{
											&dst.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf(`"%s"`, varName),
											},
										},
									},
									&dst.CallExpr{
										Fun: &dst.SelectorExpr{
											X:   &dst.Ident{Name: "time"},
											Sel: &dst.Ident{Name: "Now"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// add 1st time
		identPatchTable = append(
			identPatchTable,
			l[0].(*dst.IfStmt).Cond.(*dst.BinaryExpr).X.(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)
		// add 2nd time
		identPatchTable = append(
			identPatchTable,
			l[0].(*dst.IfStmt).Body.List[0].(*dst.AssignStmt).Rhs[0].(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)
		// add 3rd time
		identPatchTable = append(
			identPatchTable,
			l[0].(*dst.IfStmt).Body.List[1].(*dst.DeferStmt).Call.Args[1].(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)
		// add gometrics
		identPatchTable = append(
			identPatchTable,
			l[0].(*dst.IfStmt).Body.List[1].(*dst.DeferStmt).Call.
				Fun.(*dst.SelectorExpr).X.(*dst.Ident),
		)
	} else {
		l = []dst.Stmt{
			&dst.DeferStmt{
				Call: &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   &dst.Ident{Name: "gometrics"},
						Sel: &dst.Ident{Name: "MeasureSince"},
					},
					Args: []dst.Expr{
						// value of []string{""}
						&dst.CompositeLit{
							Type: &dst.ArrayType{
								Elt: &dst.Ident{Name: "string"},
							},
							Elts: []dst.Expr{
								&dst.BasicLit{
									Kind: token.STRING,
									Value: fmt.Sprintf(`"%s#%s"`,
										filename, funcName),
								},
							},
						},
						&dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   &dst.Ident{Name: "time"},
								Sel: &dst.Ident{Name: "Now"},
							},
						},
					},
				},
			},
		}
		// add gemetrics
		identPatchTable = append(identPatchTable,
			l[0].(*dst.DeferStmt).Call.Fun.(*dst.SelectorExpr).X.(*dst.Ident))
		// add time
		identPatchTable = append(identPatchTable,
			l[0].(*dst.DeferStmt).Call.Args[1].(*dst.CallExpr).
				Fun.(*dst.SelectorExpr).X.(*dst.Ident))
	}

	return g, l, identPatchTable
}

func timeConvertStatement(varPrefix string, timeStr string) (string, dst.Stmt) {
	varName := fmt.Sprintf("%s%s", varPrefix, utils.GenerateRandNumString(8))
	return varName, &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.Ident{Name: varName},
			// underscore is used to ignore the error
			&dst.Ident{Name: "_"},
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "time"},
					Sel: &dst.Ident{Name: "ParseDuration"},
				},
				Args: []dst.Expr{
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: fmt.Sprintf(`"%s"`, timeStr),
					},
				},
			},
		},
	}
}

func DefineFuncInitDecl(d *parse.CollectInfo, name string,
	directive *parse.Directive,
) (*dst.FuncDecl, []*dst.Ident) {
	var interval, duration string

	runtimeMetrics := "false"
	var runtimeMetricsInterval string

	if val, ok := directive.Param("gm-interval"); ok {
		interval = val
	} else {
		interval = "10s"
	}
	// parse interval, fail if invalid
	_, err := time.ParseDuration(interval)
	if err != nil {
		log.Fatalf("invalid gm-interval: %s, %s", err, interval)
	}

	if val, ok := directive.Param("gm-duration"); ok {
		duration = val
	} else {
		duration = "3600s"
	}
	// parse duration, fail if invalid
	_, err = time.ParseDuration(duration)
	if err != nil {
		log.Fatalf("invalid gm-duration: %s, %s", err, duration)
	}

	if val, ok := directive.Param("gm-runtime-metrics"); ok {
		if val == "true" {
			runtimeMetrics = "true"
		}
	}

	if val, ok := directive.Param("runtime-metrics-interval"); ok {
		runtimeMetricsInterval = val
	} else {
		runtimeMetricsInterval = "10s"
	}
	// parse gm-runtime-metrics-interval, fail if invalid
	_, err = time.ParseDuration(runtimeMetricsInterval)
	if err != nil {
		log.Fatalf("invalid gm-runtime-metrics-interval: %s, %s",
			err, runtimeMetricsInterval)
	}

	stmts := []dst.Stmt{}

	// generate the statements to parse the interval and duration
	intervalVarName, tmp := timeConvertStatement("interval_", interval)
	stmts = append(stmts, tmp)

	durationVarName, tmp := timeConvertStatement("duration_", duration)
	stmts = append(stmts, tmp)

	runtimeMetricsIntervalVarName, tmp := timeConvertStatement(
		"runtime_metrics_interval_",
		runtimeMetricsInterval,
	)
	if runtimeMetrics == "true" {
		stmts = append(stmts, tmp)
	}

	identPatchTable := []*dst.Ident{}

	stmts = append(stmts, &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.Ident{Name: "inm"},
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "gometrics"},
					Sel: &dst.Ident{Name: "NewInmemSink"},
				},
				Args: []dst.Expr{
					&dst.Ident{Name: intervalVarName},
					&dst.Ident{Name: durationVarName},
				},
			},
		},
		Decs: dst.AssignStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				Start:  []string{"// Setup the inmem sink and signal handler"},
			},
		},
	})
	identPatchTable = append(
		identPatchTable,
		stmts[len(stmts)-1].(*dst.AssignStmt).Rhs[0].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	stmts = append(stmts, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "gometrics"},
				Sel: &dst.Ident{Name: "DefaultInmemSignal"},
			},
			Args: []dst.Expr{
				&dst.Ident{Name: "inm"},
			},
		},
	})
	identPatchTable = append(
		identPatchTable,
		stmts[len(stmts)-1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	stmts = append(stmts, &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.Ident{Name: "cfg"},
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "gometrics"},
					Sel: &dst.Ident{Name: "DefaultConfig"},
				},
				Args: []dst.Expr{
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: fmt.Sprintf(`"%s"`, name),
					},
				},
			},
		},
	})
	identPatchTable = append(
		identPatchTable,
		stmts[len(stmts)-1].(*dst.AssignStmt).Rhs[0].(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	stmts = append(stmts, &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.SelectorExpr{
				X:   &dst.Ident{Name: "cfg"},
				Sel: &dst.Ident{Name: "EnableRuntimeMetrics"},
			},
		},
		Tok: token.ASSIGN,
		Rhs: []dst.Expr{
			&dst.Ident{Name: runtimeMetrics},
		},
	})
	if runtimeMetrics == "true" {
		stmts = append(stmts, &dst.AssignStmt{
			Lhs: []dst.Expr{
				&dst.SelectorExpr{
					X:   &dst.Ident{Name: "cfg"},
					Sel: &dst.Ident{Name: "ProfileInterval"},
				},
			},
			Tok: token.ASSIGN,
			Rhs: []dst.Expr{
				&dst.Ident{Name: runtimeMetricsIntervalVarName},
			},
		})
	}
	stmts = append(stmts, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "gometrics"},
				Sel: &dst.Ident{Name: "NewGlobal"},
			},
			Args: []dst.Expr{
				&dst.Ident{Name: "cfg"},
				&dst.Ident{Name: "inm"},
			},
		},
	})
	identPatchTable = append(
		identPatchTable,
		stmts[len(stmts)-1].(*dst.ExprStmt).X.(*dst.CallExpr).
			Fun.(*dst.SelectorExpr).X.(*dst.Ident),
	)

	res := platform.DSTInitFunc(stmts)
	return res, identPatchTable
}

func PatchProject(d *parse.CollectInfo, _ bool) error {
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
				// add the init function
				initDecl, patchTable := DefineFuncInitDecl(d, filename, directive)
				if err := d.SetGlobalDefineFunc(*directive, initDecl, pkgsRequired,
					patchTable); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.FuncExecTime {
				// add the defer statement
				g, l, patchTable := TraceFuncTimeStmts(filename,
					directive.Declaration().(*dst.FuncDecl).Name.Name, directive)
				if err := d.SetFunctionTimeTracing(*directive, g, l, pkgsRequired,
					patchTable); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.InnerExecTime {
				// add the defer statement
				if _, ok := directive.Param("gm-cooldown-time"); ok {
					return fmt.Errorf("gm-cooldown-time is not supported for inner-exec-time")
				}
				g, l, patchTable := TraceFuncTimeStmts(filename,
					directive.Declaration().(*dst.FuncDecl).Name.Name, directive)
				if err := d.SetFunctionInnerTracing(*directive, g, l, pkgsRequired,
					patchTable); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.GenBegine ||
				directive.TraceType() == parse.GenEnd {
				return fmt.Errorf("metrics code already generated")
			} else if directive.TraceType() == parse.Set {
				return fmt.Errorf("set is not supported")
			} else if directive.TraceType() == parse.InnerCounter {
				return fmt.Errorf("inner-counter is not supported")
			}
		}
	}
	return nil
}

func StoreFiles(d *parse.CollectInfo, inplace bool, suffix string, dryRun bool) error {
	if inplace && suffix != "" {
		return fmt.Errorf("cannot specify both inplace and suffix")
	}
	allFiles := d.Files()
	for _, filename := range allFiles {
		if !d.IsModified(filename) {
			continue
		}

		fDst := d.FileDst(filename)

		var newFilename string
		if inplace {
			newFilename = filename
		} else {
			newFilename = utils.NewFilenameForTracing(filename, suffix)
		}

		// put new content into a buffer
		var buf bytes.Buffer
		if err := decorator.Fprint(&buf, fDst); err != nil {
			return err
		}

		log.Infof("writing to %s", newFilename)
		if dryRun {
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

	return nil
}

func UpdatePackages(d *parse.CollectInfo, dryRun bool) error {
	if dryRun {
		return nil
	}
	log.Infof("updating go.mod...")
	return utils.FetchPackages(d.GoModPath(),
		[]string{"github.com/hashicorp/go-metrics"})
}
