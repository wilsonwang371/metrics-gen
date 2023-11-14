package gometrics

import (
	"fmt"
	"go/token"

	"github.com/dave/dst"
)

func TrackFuncTimeStmts(funcName string) []dst.Stmt {
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
