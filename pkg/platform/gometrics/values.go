package gometrics

import (
	"fmt"
	"go/token"
	"os"

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

func TraceFuncTimeStmts(funcName string) []dst.Stmt {
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
								Value: fmt.Sprintf(`"%s"`, funcName),
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

func DefineFuncInitDecl() *dst.FuncDecl {
	decl1 := &dst.AssignStmt{
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
						X:  &dst.BasicLit{Kind: token.INT, Value: "10"},
						Op: token.MUL,
						Y: &dst.SelectorExpr{
							X:   &dst.Ident{Name: "time"},
							Sel: &dst.Ident{Name: "Second"},
						},
					},
					&dst.BinaryExpr{
						X:  &dst.BasicLit{Kind: token.INT, Value: "24"},
						Op: token.MUL,
						Y: &dst.SelectorExpr{
							X:   &dst.Ident{Name: "time"},
							Sel: &dst.Ident{Name: "Hour"},
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
	}
	decl2 := &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "metrics"},
				Sel: &dst.Ident{Name: "DefaultInmemSignal"},
			},
			Args: []dst.Expr{
				&dst.Ident{Name: "inm"},
			},
		},
	}

	decl3 := &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "metrics"},
				Sel: &dst.Ident{Name: "NewGlobal"},
			},
			Args: []dst.Expr{
				&dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   &dst.Ident{Name: "metrics"},
						Sel: &dst.Ident{Name: "DefaultConfig"},
					},
					Args: []dst.Expr{
						&dst.BasicLit{
							Kind:  token.STRING,
							Value: `"service-name"`,
						},
					},
				},
				&dst.Ident{Name: "inm"},
			},
		},
	}

	res := &dst.FuncDecl{
		Name: dst.NewIdent("init"),
		Type: &dst.FuncType{},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				decl1,
				decl2,
				decl3,
			},
		},
	}
	return res
}

func PatchProject(d *parse.CollectInfo) error {
	if !d.HasDefinitionDirective() {
		return fmt.Errorf("no definition directive found")
	}
	for _, filename := range d.Files() {
		directives, err := d.FileDirectives(filename)
		if err != nil {
			return err
		}
		for _, directive := range directives {
			if directive.TraceType() == parse.DEFINE {
				// add the init function
				initDecl := DefineFuncInitDecl()
				pkgs := DefineFuncInitPkgs()
				if err := d.SetGlobalDefineFunc(*directive, initDecl, pkgs); err != nil {
					return err
				}
			} else if directive.TraceType() == parse.ON {
				// add the defer statement
				stmts := TraceFuncTimeStmts(directive.Declaration().(*dst.FuncDecl).Name.Name)
				pkgs := TraceFuncTimesPkgs()
				if err := d.SetFunctionTimeTracing(*directive, stmts, pkgs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func StoreFiles(d *parse.CollectInfo, suffix string, dryRun bool) error {
	allFiles := d.Files()
	for _, filename := range allFiles {
		if !d.IsModified(filename) {
			continue
		}

		fDst := d.FileDst(filename)
		newFilename := utils.NewFilenameForTracing(filename, suffix)

		log.Infof("writing to %s", newFilename)
		if dryRun {
			continue
		}

		// create file
		f, err := os.Create(newFilename)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := decorator.Fprint(f, fDst); err != nil {
			return err
		}
	}
	return nil
}
