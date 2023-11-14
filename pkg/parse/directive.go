package parse

import (
	"fmt"
	"regexp"

	"github.com/dave/dst"
)

type TraceType int

const (
	DEFINE TraceType = iota
	ON
	OFF
	EMPTY
	INVALID
)

type Directive struct {
	filename    string
	declaration dst.Decl
	text        string
	traceType   TraceType
}

func (d *Directive) TraceType() TraceType {
	return d.traceType
}

func (d *Directive) Declaration() dst.Decl {
	return d.declaration
}

func ParseStringDirectiveType(comment string) (TraceType, error) {
	r := regexp.MustCompile(` ?\+ ?trace\:([a-zA-Z_0-9]*) ?.*`)
	sub := r.FindStringSubmatch(comment)
	if len(sub) == 2 {
		switch sub[1] {
		case "define":
			return DEFINE, nil
		case "on":
			return ON, nil
		case "off":
			return OFF, nil
		case "":
			return EMPTY, nil
		default:
			panic("No match")
		}
	} else {
		return INVALID, fmt.Errorf("No match")
	}
}
