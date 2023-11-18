package parse

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	log "github.com/sirupsen/logrus"

	"code.byted.org/bge-infra/metrics-gen/pkg/utils"
)

type CollectInfo struct {
	fileSet        *token.FileSet
	filesDst       map[string]*dst.File    // map of file name to dst.File
	fileDirectives map[string][]*Directive // map of file name to slice of directives
	modifiedFiles  map[string]bool         // map of file name to bool

	defFileName string // file that contains the definition of the metric global variable

	goModPath string
}

// NewCollectInfo creates a new CollectInfo struct
func NewCollectInfo() *CollectInfo {
	return &CollectInfo{
		fileSet:        token.NewFileSet(),
		filesDst:       make(map[string]*dst.File),
		fileDirectives: make(map[string][]*Directive),
		modifiedFiles:  make(map[string]bool),
		defFileName:    "",
		goModPath:      "",
	}
}

// AddTraceFile adds a file to the CollectInfo struct
func (t *CollectInfo) AddTraceFile(filename string) error {
	file, err := decorator.ParseFile(t.fileSet, filename, nil,
		parser.ParseComments)
	if err != nil {
		return err
	}
	t.filesDst[filename] = file // add to map

	allDirectives, err := t.readFileDirectives(filename)
	if err != nil {
		return err
	}
	t.fileDirectives[filename] = allDirectives

	for _, directive := range allDirectives {
		if directive.traceType == Define {
			if t.defFileName != "" {
				return fmt.Errorf("multiple define files")
			}
			t.defFileName = filename
		}
	}

	return nil
}

// AddTraceFiles adds multiple files to the CollectInfo struct
func (t *CollectInfo) AddTraceFiles(filenames []string) error {
	for _, filename := range filenames {
		if err := t.AddTraceFile(filename); err != nil {
			return err
		}
	}
	return nil
}

// AddTraceDir adds all .go files in a directory to the CollectInfo struct
func (t *CollectInfo) AddTraceDir(dir string, recursive bool,
	needIgnore func(filename string) bool,
) error {
	// search all .go files
	files := []string{}
	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo,
			err error,
		) error {
			if filepath.Ext(path) == ".go" {
				// add go files to list
				files = append(files, path)
			} else if filepath.Ext(path) == ".mod" {
				// save go.mod location
				if t.goModPath != "" {
					return fmt.Errorf("multiple go.mod files")
				}
				t.goModPath = path
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		var err error
		files, err = filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			return err
		}
	}

	filteredFiles := []string{}
	for _, filename := range files {
		if needIgnore != nil && needIgnore(filename) {
			continue
		}
		log.Debugf("add traced file %s", filename)
		filteredFiles = append(filteredFiles, filename)

		continue
	}

	// reduce same file names in the list
	filteredFiles = utils.DeduplicateStrings(filteredFiles)

	t.AddTraceFiles(filteredFiles)
	return nil
}

// hasPkgImport checks if a file already has a package import
func (t *CollectInfo) hasPkgImport(filename string, pkgUrl string) bool {
	f, ok := t.filesDst[filename]
	if !ok {
		panic("file not found")
	}

	// check if the file already has the import
	for _, imp := range f.Imports {
		// check if the import is already there
		if imp.Path.Value == fmt.Sprintf(`"%s"`, pkgUrl) {
			return true
		}
	}

	// check if the file has import declaration
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok {
			if genDecl.Tok == token.IMPORT {
				for _, spec := range genDecl.Specs {
					if importSpec, ok := spec.(*dst.ImportSpec); ok {
						if importSpec.Path.Value == pkgUrl {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// AddPkgImport adds a package import to a file
// if the name and pkgUrl are the same, then the name is omitted
func (t *CollectInfo) AddPkgImport(filename string, name string,
	pkgUrl string,
) error {
	if t.hasPkgImport(filename, pkgUrl) {
		return fmt.Errorf(fmt.Sprintf("package \"%s\" already imported",
			pkgUrl))
	}
	f, ok := t.filesDst[filename]
	if !ok {
		panic("file not found")
	}

	importSpec := &dst.ImportSpec{
		// Name: &dst.Ident{Name: name},
		Path: &dst.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`"%s"`, pkgUrl),
		},
	}
	if name != pkgUrl {
		importSpec.Name = &dst.Ident{Name: name}
	}
	f.Imports = append([]*dst.ImportSpec{
		importSpec,
	}, f.Imports...)

	importDecl := &dst.GenDecl{
		Tok: token.IMPORT,
		Specs: []dst.Spec{
			&dst.ImportSpec{
				// Name: &dst.Ident{Name: name},
				Path: &dst.BasicLit{
					Kind:  token.STRING,
					Value: fmt.Sprintf(`"%s"`, pkgUrl),
				},
			},
		},
	}
	if name != pkgUrl {
		importDecl.Specs[0].(*dst.ImportSpec).Name = &dst.Ident{Name: name}
	}
	log.Debugf("import package \"%s\" as \"%s\"", pkgUrl, name)
	f.Decls = append([]dst.Decl{
		importDecl,
	}, f.Decls...)

	t.modifiedFiles[filename] = true
	return nil
}

// SetGlobalDefineFunc sets the global define function
func (t *CollectInfo) SetGlobalDefineFunc(d Directive, addedDecl *dst.FuncDecl,
	pkgs map[string]string,
) error {
	directiveIdx := -1
	file := t.filesDst[d.filename]
	for idx, decl := range file.Decls {
		if decl == d.declaration {
			directiveIdx = idx
			break
		}
	}
	for idx, decor := range d.declaration.Decorations().Start.All() {
		if d.text == decor {
			var prevComment, nextComment []string

			// copy decorations to prevComment and nextComment
			prevComment = append(prevComment, d.declaration.Decorations().Start.All()[:idx+1]...)
			nextComment = append(nextComment, d.declaration.Decorations().Start.All()[idx+1:]...)

			prevComment = append(prevComment, "// +trace:begin-generated")
			nextComment = append([]string{"// +trace:end-generated"}, nextComment...)

			addedDecl.Decorations().Start.Replace(append([]string{"\n"}, prevComment...)...)
			d.declaration.Decorations().Start.Replace(nextComment...)

			// insert code before the declaration index
			log.Debugf("add global define function for: %s", d.filename)
			file.Decls = append(file.Decls[:directiveIdx],
				append([]dst.Decl{addedDecl}, file.Decls[directiveIdx:]...)...)

			// add import
			for name, pkgUrl := range pkgs {
				t.AddPkgImport(d.filename, name, pkgUrl)
			}

			t.modifiedFiles[d.filename] = true
			return nil
		}
	}
	return fmt.Errorf("declaration not found")
}

// SetFunctionTracking sets the function time tracing
func (t *CollectInfo) SetFunctionTimeTracing(d Directive, globalDecl []dst.Decl,
	inFuncStmts []dst.Stmt, pkgs map[string]string,
) error {
	directiveIdx := -1
	file := t.filesDst[d.filename]
	for idx, decl := range file.Decls {
		if decl == d.declaration {
			directiveIdx = idx
			break
		}
	}
	if directiveIdx == -1 {
		return fmt.Errorf("declaration not found")
	}

	// insert code before the function declaration
	if len(globalDecl) != 0 {
		for idx, decor := range d.declaration.Decorations().Start.All() {
			if d.text == decor {
				var prevComment, nextComment []string

				// copy decorations to prevComment and nextComment
				prevComment = append(prevComment, d.declaration.Decorations().Start.All()[:idx+1]...)
				nextComment = append(nextComment, d.declaration.Decorations().Start.All()[idx+1:]...)

				prevComment = append(prevComment, "// +trace:begin-generated")
				nextComment = append([]string{"// +trace:end-generated"}, nextComment...)

				globalDecl[0].Decorations().Start.Replace(append([]string{"\n"}, prevComment...)...)
				d.declaration.Decorations().Start.Replace(nextComment...)

				// insert code before the declaration index
				log.Debugf("add global define function for: %s", d.filename)
				file.Decls = append(file.Decls[:directiveIdx],
					append(globalDecl, file.Decls[directiveIdx:]...)...)

				// add import
				for name, pkgUrl := range pkgs {
					t.AddPkgImport(d.filename, name, pkgUrl)
				}

				t.modifiedFiles[d.filename] = true
				break
			}
		}
	}

	// insert code in the beginning of the function
	log.Infof("add function time tracing for: %s", d.declaration.(*dst.FuncDecl).Name.Name)

	inFuncStmts[0].Decorations().Start.Prepend("\n", "// +trace:begin-generated")
	inFuncStmts[len(inFuncStmts)-1].Decorations().End.Append("\n", "// +trace:end-generated")

	d.declaration.(*dst.FuncDecl).Body.List = append(inFuncStmts,
		d.declaration.(*dst.FuncDecl).Body.List...)

	// add import
	for name, pkgUrl := range pkgs {
		if err := t.AddPkgImport(d.filename, name, pkgUrl); err != nil {
			log.Debug(err) // ignore error
		}
	}

	t.modifiedFiles[d.filename] = true
	return nil
}

// return all the directives in a file
func (t *CollectInfo) readFileDirectives(filename string) ([]*Directive, error) {
	res := []*Directive{}
	file, ok := t.filesDst[filename]
	if !ok {
		log.Errorf("file %s not found", filename)
		return res, fmt.Errorf("file not found")
	}
	for _, decl := range file.Decls {
		for _, decor := range decl.Decorations().Start.All() {
			if traceType, err := ParseStringDirectiveType(decor); err == nil {
				d := &Directive{
					filename:    filename,
					declaration: decl,
					text:        decor,
					traceType:   traceType,
				}
				// find all arguments
				if params, err := ParseDirectiveParams(decor); err == nil {
					if len(params) != 0 {
						log.Debugf("found directive params: %v", params)
					}
					d.params = params
				}
				res = append(res, d)
			}
		}
	}
	return res, nil
}

// HasDefinitionDirective checks if the CollectInfo struct has a definition directive
func (t *CollectInfo) HasDefinitionDirective() bool {
	return t.defFileName != ""
}

// Files returns all the files in the CollectInfo struct
func (t *CollectInfo) Files() []string {
	res := []string{}
	for filename := range t.filesDst {
		res = append(res, filename)
	}
	return res
}

// FileDst returns the dst.File for a file
func (t *CollectInfo) FileDst(filename string) *dst.File {
	return t.filesDst[filename]
}

func (t *CollectInfo) IsModified(filename string) bool {
	return t.modifiedFiles[filename]
}

func (t *CollectInfo) GoModPath() string {
	if t.goModPath == "" {
		log.Infof("go.mod not found")
	}
	return t.goModPath
}

func (t *CollectInfo) FileDirectives(filename string) ([]*Directive, error) {
	if _, ok := t.fileDirectives[filename]; !ok {
		return nil, fmt.Errorf("file %s not found", filename)
	}
	return t.fileDirectives[filename], nil
}
