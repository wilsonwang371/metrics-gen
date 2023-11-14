package parse

import (
	"fmt"
	"regexp"

	"github.com/dave/dst"
)

type TraceType int

const (
	Define TraceType = iota
	On
	Off
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
}

func (d *Directive) TraceType() TraceType {
	return d.traceType
}

func (d *Directive) Declaration() dst.Decl {
	return d.declaration
}

func ParseStringDirectiveType(comment string) (TraceType, error) {
	r := regexp.MustCompile(` ?\+ ?trace\:([a-zA-Z_\-0-9]*) ?.*`)
	sub := r.FindStringSubmatch(comment)
	if len(sub) == 2 {
		switch sub[1] {
		case "define":
			return Define, nil
		case "on":
			return On, nil
		case "off":
			return Off, nil
		case "":
			return Empty, nil
		case "begin-generated":
			return GenBegine, nil
		case "end-generated":
			return GenEnd, nil
		default:
			panic("No match")
		}
	} else {
		return Invalid, fmt.Errorf("No match")
	}
}
