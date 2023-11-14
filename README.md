
# metrics-gen

`metrics-gen` is a Go code generation tool that can inject code into Go source files based on directive comments specified within the source code. It allows you to automate the generation of metrics-related code, saving you time and ensuring consistency in your codebase.

## Features

- Automatic code generation based on directive comments.
- Support for injecting metrics-related code into Go source files.
- Easy customization and extensibility.

## Installation

```bash
go get code.byted.org/bge-infra/metrics-gen
```

## Compile
    
```bash
make build
```

## Usage

```bash
metrics-gen -h
This command will generate new files with the patched code
that captures the metrics for your code.

Usage:
  metrics-gen generate [flags]

Flags:
  -h, --help            help for generate
  -i, --inplace         patch files in place
  -s, --suffix string   suffix to add to generated files. If suffix is tracegen, then generated files will be named <filename>_tracegen.go

Global Flags:
  -d, --dir strings    directory to search for files
  -n, --dry-run        dry run
  -r, --rdir strings   recursive directory to search for files
  -v, --verbose        verbose output
```
