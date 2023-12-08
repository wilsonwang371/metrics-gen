package parse

import (
	"fmt"
	"regexp"

	"code.byted.org/bge-infra/metrics-gen/pkg/utils"
	"github.com/dave/dst"
)

type TraceType int

const (
	Define TraceType = iota
	Set
	FuncExecTime
	InnerExecTime
	InnerCounter
	Empty
	GenBegine
	GenEnd
	Invalid
)

type Directive struct {
	filename    string
	declaration dst.Decl
	text        string
	traceType   TraceType
	params      map[string]string // map of parameter name to value
}

func (d *Directive) TraceType() TraceType {
	return d.traceType
}

func (d *Directive) Declaration() dst.Decl {
	return d.declaration
}

func (d *Directive) Param(name string) (string, bool) {
	res, ok := d.params[name]
	return res, ok
}

func (d *Directive) Params() map[string]string {
	return d.params
}

func ParseStringDirectiveType(comment string) (TraceType, error) {
	r := regexp.MustCompile(` ?\+ ?trace\:([a-zA-Z_\-0-9]*) ?(.*)`)
	sub := r.FindStringSubmatch(comment)
	if len(sub) >= 2 {
		switch sub[1] {
		case "define":
			return Define, nil
		case "set":
			return Set, nil
		case "func-exec-time":
			return FuncExecTime, nil
		case "inner-exec-time":
			return InnerExecTime, nil
		case "inner-counter":
			return InnerCounter, nil
		case "":
			return Empty, nil
		case "begin-generated":
			return GenBegine, nil
		case "end-generated":
			return GenEnd, nil
		default:
			return Invalid, fmt.Errorf("Unknown trace type: %s, %+v", comment, sub)
		}
	} else {
		return Invalid, nil
	}
}

// ParseDirectiveType parses arguments from a trace directive comment
func ParseDirectiveParams(comment string) (map[string]string, error) {
	r := regexp.MustCompile(` ?\+ ?trace\:([a-zA-Z_\-0-9]*) ?(.*)`)
	sub := r.FindStringSubmatch(comment)
	if len(sub) == 3 {
		return utils.ParseArguments(sub[2]), nil
	} else {
		return nil, fmt.Errorf("No match")
	}
}
