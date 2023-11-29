package platform

import (
	"code.byted.org/bge-infra/metrics-gen/pkg/parse"
	"github.com/dave/dst"
)

// define common interface for metrics providers
type MetricsProvider interface {
	// pre patch
	PrePatch(info *parse.CollectInfo) error

	// patch
	Patch(info *parse.CollectInfo) error

	// post patch
	PostPatch(info *parse.CollectInfo) error
}

func DSTInitFunc(stmts []dst.Stmt) *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: dst.NewIdent("init"),
		Type: &dst.FuncType{},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}
