package parse

import (
	"fmt"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	log "github.com/sirupsen/logrus"

	"github.com/wilsonwang371/metrics-gen/metrics-gen/pkg/utils"

	"github.com/google/uuid"
)

type CollectInfo struct {
	fileSet        *token.FileSet
	filesDst       map[string]*dst.File    // map of file name to dst.File
	fileDirectives map[string][]*Directive // map of file name to slice of directives
	modifiedFiles  map[string]bool         // map of file name to bool

	defFileName string // file that contains the definition of the metric global variable

	goModPath string
	genUUID   string
}

// NewCollectInfo creates a new CollectInfo struct
func NewCollectInfo() *CollectInfo {
	tmpUUID := uuid.New().String()
	return &CollectInfo{
		fileSet:        token.NewFileSet(),
		filesDst:       make(map[string]*dst.File),
		fileDirectives: make(map[string][]*Directive),
		modifiedFiles:  make(map[string]bool),
		defFileName:    "",
		goModPath:      "",
		genUUID:        tmpUUID,
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
		// check if go.mod exists
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// save go.mod location
			if t.goModPath != "" {
				return fmt.Errorf("multiple go.mod files")
			}
			t.goModPath = filepath.Join(dir, "go.mod")
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

type PackageInfo struct {
	Name string
	Path string
}

// enum for hasPkgImport return
type PkgCheckResult int

const (
	PkgExistsAndNoChange PkgCheckResult = iota
	PkgNotExistsAndChangeName
	PkgExistsAndChangeName
	PkgNotExists
)

// hasPkgImport checks if a file already has a package import, and returns the new name for the import
func (t *CollectInfo) hasPkgImport(
	filename string,
	importName string,
	pkgUrl string,
) (PkgCheckResult, string) {
	f, ok := t.filesDst[filename]
	if !ok {
		panic("file not found")
	}

	// check if the file has import declaration
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok {
			if genDecl.Tok == token.IMPORT {
				for _, spec := range genDecl.Specs {
					if importSpec, ok := spec.(*dst.ImportSpec); ok {
						// check if the import is already there
						tmpName := ""
						if importSpec.Name != nil {
							tmpName = importSpec.Name.Name
						}
						if tmpName == "" {
							tmp := strings.Trim(importSpec.Path.Value, `"`)
							tmpName = filepath.Base(tmp)
						}
						tmpName2 := importName
						if tmpName2 == "" {
							tmpName2 = filepath.Base(pkgUrl)
						}
						if tmpName == tmpName2 {
							if importSpec.Path.Value == fmt.Sprintf(`"%s"`, pkgUrl) {
								return PkgExistsAndNoChange, ""
							}
							return PkgNotExistsAndChangeName, ""
						} else {
							if importSpec.Path.Value == fmt.Sprintf(`"%s"`, pkgUrl) {
								return PkgExistsAndChangeName, tmpName
							}
						}
					}
				}
			}
		}
	}
	return PkgNotExists, ""
}

// AddPkgImport adds a package import to a file
// if the name and pkgUrl are the same, then the name is omitted
func (t *CollectInfo) AddPkgImport(filename string, name string,
	pkgUrl string,
) error {
	log.Debugf("add package import: %s %s %s", filename, name, pkgUrl)

	res, newName := t.hasPkgImport(filename, name, pkgUrl)
	switch res {
	case PkgExistsAndNoChange:
		return nil
	case PkgExistsAndChangeName:
		return fmt.Errorf("use existing import name \"%s\"", newName)
	case PkgNotExistsAndChangeName:
		return fmt.Errorf("change import name")
	case PkgNotExists:
		// do nothing
	default:
		panic("unknown result")
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
func (t *CollectInfo) SetGlobalDefineFunc(d Directive,
	addedDecl *dst.FuncDecl,
	pkgsIn map[string]*PackageInfo,
	pkgPatchTable []*dst.Ident,
) error {
	// deep copy pkgs
	pkgs := make(map[string]*PackageInfo)
	for k, v := range pkgsIn {
		pkgs[k] = &PackageInfo{
			Name: v.Name,
			Path: v.Path,
		}
	}

	for _, decor := range d.declaration.Decorations().Start.All() {
		if d.text == decor {
			// add import
			pkgsUpdated := false
			for name, pkg := range pkgs {
				// loop until AddPkgImport succeeds
				for {
					if err := t.AddPkgImport(d.filename, pkg.Name, pkg.Path); err != nil {
						if err.Error() == "change import name" {
							pkg.Name = fmt.Sprintf("%s_%d", pkg.Name, rand.Intn(100))
							pkgsUpdated = true
							continue
						} else if strings.Contains(err.Error(), "use existing import name") {
							// extract the name from the error message that enclosed by double quotes
							// e.g. "use existing import name \"prometheus\""
							// then replace the pkg name with the new name
							newName := strings.Trim(
								strings.Split(err.Error(), "\"")[1],
								"\"")
							log.Infof("use existing import name \"%s\" for pkg %+v", newName, pkg)
							pkgs[name].Name = newName
							pkgsUpdated = true
							continue
						} else {
							return err
						}
					}
					break
				}
			}

			if pkgsUpdated {
				for _, ident := range pkgPatchTable {
					for name, pkg := range pkgs {
						// search if pkg name is a substring of ident name
						if strings.Contains(ident.Name, name) {
							// replace the substring with the new name
							ident.Name = strings.Replace(ident.Name, name, pkg.Name, 1)
						}
					}
				}
			}

			t.modifiedFiles[d.filename] = true
			// break 2 loops
			goto out
		}
	}
out:
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
			prevComment = append(
				prevComment,
				d.declaration.Decorations().Start.All()[:idx+1]...)
			nextComment = append(
				nextComment,
				d.declaration.Decorations().Start.All()[idx+1:]...)

			prevComment = append(prevComment, BeginUUID(t.genUUID))
			nextComment = append([]string{EndUUID(t.genUUID)}, nextComment...)

			addedDecl.Decorations().Start.Replace(
				append([]string{"\n"}, prevComment...)...)
			d.declaration.Decorations().Start.Replace(nextComment...)

			// insert code before the declaration index
			log.Debugf("add global define function for: %s", d.filename)
			file.Decls = append(file.Decls[:directiveIdx],
				append([]dst.Decl{addedDecl}, file.Decls[directiveIdx:]...)...)

			t.modifiedFiles[d.filename] = true
			return nil
		}
	}
	return fmt.Errorf("declaration not found")
}

// SetFunctionInnerTracing sets the function inner tracing
func (t *CollectInfo) SetFunctionInnerTracing(d Directive,
	globalDecl []dst.Decl,
	inFuncStmts []dst.Stmt,
	pkgsIn map[string]*PackageInfo,
	pkgPatchTable []*dst.Ident,
) error {
	// deep copy pkgs
	pkgs := make(map[string]*PackageInfo)
	for k, v := range pkgsIn {
		pkgs[k] = &PackageInfo{
			Name: v.Name,
			Path: v.Path,
		}
	}

	funDecl, ok := d.declaration.(*dst.FuncDecl)
	if !ok {
		return fmt.Errorf("declaration is not a function")
	}
	for _, stmt := range funDecl.Body.List {
		for _, decor := range stmt.Decorations().Start.All() {
			if d.text == decor {
				// add import
				pkgsUpdated := false
				for name, pkg := range pkgs {
					for {
						if err := t.AddPkgImport(d.filename, pkg.Name, pkg.Path); err != nil {
							if err.Error() == "change import name" {
								pkg.Name = fmt.Sprintf("%s_%d", pkg.Name, rand.Intn(100))
								pkgsUpdated = true
								continue
							} else if strings.Contains(err.Error(), "use existing import name") {
								// extract the name from the error message that enclosed by double quotes
								// e.g. "use existing import name \"prometheus\""
								// then replace the pkg name with the new name
								newName := strings.Trim(
									strings.Split(err.Error(), "\"")[1],
									"\"")
								log.Infof("use existing import name \"%s\" for pkg %+v", newName, pkg)
								pkgs[name].Name = newName
								pkgsUpdated = true
								continue
							} else {
								return err
							}
						}
						break
					}
				}

				if pkgsUpdated {
					for _, ident := range pkgPatchTable {
						for name, pkg := range pkgs {
							// search if pkg name is a substring of ident name
							if strings.Contains(ident.Name, name) {
								// replace the substring with the new name
								ident.Name = strings.Replace(
									ident.Name,
									name,
									pkg.Name,
									1,
								)
							}
						}
					}
				}

				t.modifiedFiles[d.filename] = true
				// break 2 loops
				goto out
			}
		}
	}
out:
	// add global statements
	directiveIdx := -1
	file := t.filesDst[d.filename]
	for idx, decl := range file.Decls {
		if _, ok := decl.(*dst.FuncDecl); ok {
			directiveIdx = idx
		}
	}
	if directiveIdx == -1 {
		return fmt.Errorf("declaration not found")
	}

	// insert code before the function declaration
	if len(globalDecl) != 0 {
		globalDecl[0].Decorations().Start.Prepend("\n", BeginUUID(t.genUUID))
		globalDecl[len(globalDecl)-1].Decorations().End.Append("\n", EndUUID(t.UUID()))
		file.Decls = append(file.Decls[:directiveIdx],
			append(globalDecl, file.Decls[directiveIdx:]...)...)
	}

	// add local statements
	for idx, stmt := range funDecl.Body.List {
		for idx2, decor := range stmt.Decorations().Start.All() {
			if d.text == decor {
				var prevComment, nextComment []string

				// copy decorations to prevComment and nextComment
				prevComment = append(
					prevComment,
					stmt.Decorations().Start.All()[:idx2+1]...)
				nextComment = append(
					nextComment,
					stmt.Decorations().Start.All()[idx2+1:]...)

				prevComment = append(prevComment, BeginUUID(t.genUUID))
				nextComment = append([]string{EndUUID(t.UUID())}, nextComment...)

				log.Debugf("prevComment: %v", prevComment)
				log.Debugf("nextComment: %v", nextComment)

				// \n prevComment BeginUUID
				inFuncStmts[0].Decorations().Start.Prepend(prevComment...)
				inFuncStmts[0].Decorations().Start.Prepend("\n")
				stmt.Decorations().Start.Replace(nextComment...)

				// insert code before the declaration index
				funDecl.Body.List = append(funDecl.Body.List[:idx],
					append(inFuncStmts, funDecl.Body.List[idx:]...)...)

				t.modifiedFiles[d.filename] = true
				return nil
			}
		}
	}
	return fmt.Errorf("not implemented")
}

// SetFunctionTracking sets the function time tracing
func (t *CollectInfo) SetFunctionTimeTracing(d Directive,
	globalDecl []dst.Decl,
	inFuncStmts []dst.Stmt,
	pkgsIn map[string]*PackageInfo,
	pkgPatchTable []*dst.Ident,
) error {
	if len(inFuncStmts) == 0 {
		return fmt.Errorf("no statements to insert")
	}

	// deep copy pkgs
	pkgs := make(map[string]*PackageInfo)
	for k, v := range pkgsIn {
		pkgs[k] = &PackageInfo{
			Name: v.Name,
			Path: v.Path,
		}
	}

	// add import
	pkgsUpdated := false
	for name, pkg := range pkgs {
		for {
			if err := t.AddPkgImport(d.filename, pkg.Name, pkg.Path); err != nil {
				if err.Error() == "change import name" {
					pkg.Name = fmt.Sprintf("%s_%d", pkg.Name, rand.Intn(100))
					pkgsUpdated = true
					continue
				} else if strings.Contains(err.Error(), "use existing import name") {
					// extract the name from the error message that enclosed by double quotes
					// e.g. "use existing import name \"prometheus\""
					// then replace the pkg name with the new name
					newName := strings.Trim(
						strings.Split(err.Error(), "\"")[1],
						"\"")
					log.Infof("use existing import name \"%s\" for pkg %+v", newName, pkg)
					pkgs[name].Name = newName
					pkgsUpdated = true
					continue
				} else {
					return err
				}
			}
			break
		}
	}
	if pkgsUpdated {
		for _, ident := range pkgPatchTable {
			for name, pkg := range pkgs {
				// search if pkg name is a substring of ident name
				if strings.Contains(ident.Name, name) {
					// replace the substring with the new name
					ident.Name = strings.Replace(ident.Name, name, pkg.Name, 1)
				}
			}
		}
	}

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
				prevComment = append(
					prevComment,
					d.declaration.Decorations().Start.All()[:idx+1]...)
				nextComment = append(
					nextComment,
					d.declaration.Decorations().Start.All()[idx+1:]...)

				prevComment = append(prevComment, BeginUUID(t.genUUID))
				nextComment = append([]string{EndUUID(t.UUID())}, nextComment...)

				globalDecl[0].Decorations().Start.Replace(
					append([]string{"\n"}, prevComment...)...)
				d.declaration.Decorations().Start.Replace(nextComment...)

				// insert code before the declaration index
				log.Debugf("add global define function for: %s", d.filename)
				file.Decls = append(file.Decls[:directiveIdx],
					append(globalDecl, file.Decls[directiveIdx:]...)...)

				t.modifiedFiles[d.filename] = true
				break
			}
		}
	}

	// insert code in the beginning of the function
	log.Infof(
		"add function time tracing for: %s",
		d.declaration.(*dst.FuncDecl).Name.Name,
	)

	inFuncStmts[0].Decorations().Start.Prepend("\n", BeginUUID(t.genUUID))
	inFuncStmts[len(inFuncStmts)-1].Decorations().End.Append("\n", EndUUID(t.UUID()))

	d.declaration.(*dst.FuncDecl).Body.List = append(inFuncStmts,
		d.declaration.(*dst.FuncDecl).Body.List...)

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
		// check all prefix comments and find out the directives
		for _, decor := range decl.Decorations().Start.All() {
			if traceType, err := ParseStringDirectiveType(decor); err == nil {
				if traceType != Invalid {
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
			} else {
				return nil, err
			}
		}

		// check declaration internal code and find out the directives
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			for _, stmt := range funcDecl.Body.List {
				for _, decor := range stmt.Decorations().Start.All() {
					if traceType, err := ParseStringDirectiveType(decor); err == nil {
						if traceType != Invalid {
							log.Debugf("found inner directive: %s", decor)
							d := &Directive{
								filename:    filename,
								declaration: funcDecl,
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
					} else {
						return nil, err
					}
				}
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

func (t *CollectInfo) UUID() string {
	return t.genUUID
}

func BeginUUID(uuid string) string {
	return fmt.Sprintf("// +trace:begin-generated uuid=%s", uuid)
}

func EndUUID(uuid string) string {
	return fmt.Sprintf("// +trace:end-generated uuid=%s", uuid)
}
