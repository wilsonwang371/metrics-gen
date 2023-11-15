package gometrics

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"code.byted.org/bge-infra/metrics-gen/pkg/parse"
	"code.byted.org/bge-infra/metrics-gen/pkg/utils"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TraceFuncTimesPkgs() map[string]string {
	resMap := map[string]string{}
	resMap["metrics"] = "github.com/hashicorp/go-metrics"
	resMap["time"] = "time"
	return resMap
}

func TraceFuncTimeStmts(filename string, funcName string) []dst.Stmt {
	return []dst.Stmt{
		&dst.DeferStmt{
			Call: &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "metrics"},
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
								Value: fmt.Sprintf(`"%s#%s"`, filename, funcName),
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

func DefineFuncInitPkgs() map[string]string {
	resMap := map[string]string{}
	resMap["metrics"] = "github.com/hashicorp/go-metrics"
	resMap["time"] = "time"
	return resMap
}

func DefineFuncInitDecl(d *parse.CollectInfo, name string) *dst.FuncDecl {
	var interval, duration string

	runtimeMetrics := "false"
	var runtimeMetricsInterval string

	if val, ok := d.DefParam("interval"); ok {
		interval = val
	} else {
		interval = "10"
	}

	if val, ok := d.DefParam("duration"); ok {
		duration = val
	} else {
		duration = "3600"
	}

	if val, ok := d.DefParam("runtime-metrics"); ok {
		if val == "true" {
			runtimeMetrics = "true"
		}
	}

	if val, ok := d.DefParam("runtime-metrics-interval"); ok {
		runtimeMetricsInterval = val
	} else {
		runtimeMetricsInterval = "10"
	}

	stmts := []dst.Stmt{}

	stmts = append(stmts, &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.Ident{Name: "inm"},
		},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "metrics"},
					Sel: &dst.Ident{Name: "NewInmemSink"},
				},
				Args: []dst.Expr{
					&dst.BinaryExpr{
						X:  &dst.BasicLit{Kind: token.INT, Value: interval},
						Op: token.MUL,
						Y: &dst.SelectorExpr{
							X:   &dst.Ident{Name: "time"},
							Sel: &dst.Ident{Name: "Second"},
						},
					},
					&dst.BinaryExpr{
						X:  &dst.BasicLit{Kind: token.INT, Value: duration},
						Op: token.MUL,
						Y: &dst.SelectorExpr{
							X:   &dst.Ident{Name: "time"},
							Sel: &dst.Ident{Name: "Second"},
						},
					},
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
				X:   &dst.Ident{Name: "metrics"},
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
					X:   &dst.Ident{Name: "metrics"},
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
				&dst.BinaryExpr{
					X:  &dst.BasicLit{Kind: token.INT, Value: runtimeMetricsInterval},
					Op: token.MUL,
					Y: &dst.SelectorExpr{
						X:   &dst.Ident{Name: "time"},
						Sel: &dst.Ident{Name: "Second"},
					},
				},
			},
		})
	}
	stmts = append(stmts, &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "metrics"},
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
				initDecl := DefineFuncInitDecl(d, filename)
				pkgs := DefineFuncInitPkgs()
				if err := d.SetGlobalDefineFunc(*directive, initDecl, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.On {
				// add the defer statement
				stmts := TraceFuncTimeStmts(filename, directive.Declaration().(*dst.FuncDecl).Name.Name)
				pkgs := TraceFuncTimesPkgs()
				if err := d.SetFunctionTimeTracing(*directive, stmts, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.GenBegine || directive.TraceType() == parse.GenEnd {
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
	return utils.FetchPackages(d.GoModPath(), []string{"github.com/hashicorp/go-metrics"})
}
