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
	defFileName    string                  // file that contains the definition of the metric global variable
}

// NewCollectInfo creates a new CollectInfo struct
func NewCollectInfo() *CollectInfo {
	return &CollectInfo{
		fileSet:        token.NewFileSet(),
		filesDst:       make(map[string]*dst.File),
		fileDirectives: make(map[string][]*Directive),
		modifiedFiles:  make(map[string]bool),
		defFileName:    "",
	}
}

// AddTraceFile adds a file to the CollectInfo struct
func (t *CollectInfo) AddTraceFile(filename string) error {
	file, err := decorator.ParseFile(t.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	t.filesDst[filename] = file // add to map

	allDirectives, err := t.FileDirectives(filename)
	if err != nil {
		return err
	}
	t.fileDirectives[filename] = allDirectives

	for _, directive := range allDirectives {
		if directive.traceType == DEFINE {
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
	needIgnore func(filename string) bool) error {
	// search all .go files
	files := []string{}
	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == ".go" {
				files = append(files, path)
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
		log.Infof("add traced file %s", filename)
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
			log.Infof("package %s already imported", pkgUrl)
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
func (t *CollectInfo) AddPkgImport(filename string, name string, pkgUrl string) error {
	if t.hasPkgImport(filename, pkgUrl) {
		return fmt.Errorf("package already imported")
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
	log.Infof("add import %s %s", name, pkgUrl)
	f.Decls = append([]dst.Decl{
		importDecl,
	}, f.Decls...)

	t.modifiedFiles[filename] = true
	return nil
}

// SetGlobalDefineFunc sets the global define function
func (t *CollectInfo) SetGlobalDefineFunc(d Directive, addedDecl *dst.FuncDecl, pkgs map[string]string) error {
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
			prevComment := d.declaration.Decorations().Start.All()[:idx+1]
			nextComment := d.declaration.Decorations().Start.All()[idx+1:]

			d.declaration.Decorations().Start.Replace(nextComment...)
			addedDecl.Decorations().Start.Replace(append([]string{"\n"}, prevComment...)...)

			// insert code before the declaration index
			log.Infof("add global define function for: %s", d.filename)
			file.Decls = append(file.Decls[:directiveIdx], append([]dst.Decl{addedDecl}, file.Decls[directiveIdx:]...)...)

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
func (t *CollectInfo) SetFunctionTimeTracing(d Directive, addedStmts []dst.Stmt, pkgs map[string]string) error {
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
	log.Infof("add function time tracing for: %s", d.declaration.(*dst.FuncDecl).Name.Name)
	d.declaration.(*dst.FuncDecl).Body.List = append(addedStmts, d.declaration.(*dst.FuncDecl).Body.List...)

	// add import
	for name, pkgUrl := range pkgs {
		if err := t.AddPkgImport(d.filename, name, pkgUrl); err != nil {
			log.Info(err) // ignore error
		}
	}

	t.modifiedFiles[d.filename] = true
	return nil
}

// return all the directives in a file
func (t *CollectInfo) FileDirectives(filename string) ([]*Directive, error) {
	res := []*Directive{}
	file, ok := t.filesDst[filename]
	if !ok {
		log.Errorf("file %s not found", filename)
		return res, fmt.Errorf("file not found")
	}
	for _, decl := range file.Decls {
		for _, decor := range decl.Decorations().Start.All() {
			if traceType, err := ParseStringDirectiveType(decor); err == nil {
				res = append(res, &Directive{
					filename:    filename,
					declaration: decl,
					text:        decor,
					traceType:   traceType,
				})
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
