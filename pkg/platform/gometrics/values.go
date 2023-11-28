package gometrics

import (
	"bytes"
	"fmt"
	"go/token"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"code.byted.org/bge-infra/metrics-gen/pkg/parse"
	"code.byted.org/bge-infra/metrics-gen/pkg/utils"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TraceFuncTimesPkgs() map[string]string {
	resMap := map[string]string{}
	resMap["gometrics"] = "github.com/hashicorp/go-metrics"
	resMap["time"] = "time"
	return resMap
}

func TraceFuncTimeStmts(filename string, funcName string,
	directive *parse.Directive,
) (globalDecl []dst.Decl, inFuncStmts []dst.Stmt) {
	cooldownTime := ""
	if v, ok := directive.Param("cooldown-time"); ok {
		cooldownTime = v
		if _, err := time.ParseDuration(cooldownTime); err != nil {
			log.Fatalf("invalid cooldown-time: %s, %s", err, cooldownTime)
		}
	}

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
					},
				},
			},
		}
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
	}

	return g, l
}

func DefineFuncInitPkgs() map[string]string {
	resMap := map[string]string{}
	resMap["gometrics"] = "github.com/hashicorp/go-metrics"
	resMap["time"] = "time"
	return resMap
}

// Function to generate a random number string of a specified length
func generateRandomNumberString(length int) string {
	const charset = "0123456789" // You can add more characters if needed
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}

func timeConvertStatement(varPrefix string, timeStr string) (string, dst.Stmt) {
	varName := fmt.Sprintf("%s%s", varPrefix, generateRandomNumberString(8))
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
) *dst.FuncDecl {
	var interval, duration string

	runtimeMetrics := "false"
	var runtimeMetricsInterval string

	if val, ok := directive.Param("interval"); ok {
		interval = val
	} else {
		interval = "10s"
	}
	// parse interval, fail if invalid
	_, err := time.ParseDuration(interval)
	if err != nil {
		log.Fatalf("invalid interval: %s, %s", err, interval)
	}

	if val, ok := directive.Param("duration"); ok {
		duration = val
	} else {
		duration = "3600s"
	}
	// parse duration, fail if invalid
	_, err = time.ParseDuration(duration)
	if err != nil {
		log.Fatalf("invalid duration: %s, %s", err, duration)
	}

	if val, ok := directive.Param("runtime-metrics"); ok {
		if val == "true" {
			runtimeMetrics = "true"
		}
	}

	if val, ok := directive.Param("runtime-metrics-interval"); ok {
		runtimeMetricsInterval = val
	} else {
		runtimeMetricsInterval = "10s"
	}
	// parse runtime-metrics-interval, fail if invalid
	_, err = time.ParseDuration(runtimeMetricsInterval)
	if err != nil {
		log.Fatalf("invalid runtime-metrics-interval: %s, %s", err, runtimeMetricsInterval)
	}

	stmts := []dst.Stmt{}

	// generate the statements to parse the interval and duration
	intervalVarName, tmp := timeConvertStatement("interval_", interval)
	stmts = append(stmts, tmp)

	durationVarName, tmp := timeConvertStatement("duration_", duration)
	stmts = append(stmts, tmp)

	runtimeMetricsIntervalVarName, tmp := timeConvertStatement("runtime_metrics_interval_",
		runtimeMetricsInterval)
	if runtimeMetrics == "true" {
		stmts = append(stmts, tmp)
	}

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

	res := &dst.FuncDecl{
		Name: dst.NewIdent("init"),
		Type: &dst.FuncType{},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
	return res
}

func PatchProject(d *parse.CollectInfo, _ bool) error {
	if !d.HasDefinitionDirective() {
		return fmt.Errorf("no definition directive found")
	}
	for _, fullpath := range d.Files() {
		directives, err := d.FileDirectives(fullpath)
		if err != nil {
			return err
		}
		for _, directive := range directives {
			base := filepath.Base(fullpath)                      // Get the base (filename) from the full path
			filename := base[:len(base)-len(filepath.Ext(base))] // Remove the extension
			if directive.TraceType() == parse.Define {
				// add the init function
				initDecl := DefineFuncInitDecl(d, filename, directive)
				pkgs := DefineFuncInitPkgs()
				if err := d.SetGlobalDefineFunc(*directive, initDecl, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.FuncExecTime {
				// add the defer statement
				g, l := TraceFuncTimeStmts(filename,
					directive.Declaration().(*dst.FuncDecl).Name.Name, directive)
				pkgs := TraceFuncTimesPkgs()
				if err := d.SetFunctionTimeTracing(*directive, g, l, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.InnerExecTime {
				// add the defer statement
				if _, ok := directive.Param("cooldown-time"); ok {
					return fmt.Errorf("cooldown-time is not supported for inner-exec-time")
				}

				_, l := TraceFuncTimeStmts(filename,
					directive.Declaration().(*dst.FuncDecl).Name.Name, directive)
				pkgs := TraceFuncTimesPkgs()
				if err := d.SetFunctionInnerTimeTracing(*directive, l, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.GenBegine ||
				directive.TraceType() == parse.GenEnd {
				return fmt.Errorf("metrics code already generated")
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

func PostPatch(d *parse.CollectInfo, dryRun bool) error {
	if dryRun {
		return nil
	}
	log.Infof("updating go.mod...")
	return utils.FetchPackages(d.GoModPath(),
		[]string{"github.com/hashicorp/go-metrics"})
}
