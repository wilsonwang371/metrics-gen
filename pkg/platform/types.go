package platform

import (
	"github.com/dave/dst"
	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/parse"
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

type MetricsProviderConfig struct {
	MetricsPrefix string
	Provider      string
	DryRun        bool
	Inplace       bool
	Suffix        string
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
