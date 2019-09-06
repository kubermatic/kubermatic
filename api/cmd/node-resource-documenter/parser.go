package main

import (
	"strings"
)

// parseFunc represents the current scanning state of the parser.
type parseFunc func() bool

// parser is a state machine for analyzing the YAML line of the
// addons, including the templates which make using a YAML lib
// unsuccessful.
type parser struct {
	parse  parseFunc
	indent string
	in     *buffer
	out    *buffer
}

func newParser(in *buffer) *parser {
	p := &parser{
		in:  in,
		out: &buffer{},
	}
	p.parse = p.container
	return p
}

func (p *parser) do() {
	// Parse as long as parse functions don't tell to stop.
	for p.parse() {
	}
}

// container parses until "containers:" is found.
func (p *parser) container() bool {
	line, ok := p.in.pop()
	if !ok {
		return false
	}
	index := strings.Index(line, "containers:")
	if index != -1 {
		p.indent = line[:index]
		p.parse = p.containerName
	}
	return true
}

// containerName parses for the container name in  "- name:".
func (p *parser) containerName() bool {
	line, ok := p.in.pop()
	if !ok {
		return false
	}
	namePrefix := p.indent + "- name: "
	index := strings.IndexFunc(line, func(r rune) bool {
		return r != ' ' && r != '\t'
	})
	switch {
	case index < len(p.indent):
		// Higher level block.
		p.in.pushBack(line)
		p.parse = p.container
	case index == len(p.indent) && !strings.HasPrefix(line, namePrefix):
		// Different block on same level.
		p.in.pushBack(line)
		p.parse = p.container
	case strings.HasPrefix(line, namePrefix):
		// Found name.
		name := line[len(namePrefix):]
		p.out.push("container: " + name)
		p.parse = p.resources
	}
	return true
}

// resources parses for until "resources:" is found.
func (p *parser) resources() bool {
	line, ok := p.in.pop()
	if !ok {
		return false
	}
	resourcesPrefix := p.indent + "  resources:"
	if strings.HasPrefix(line, resourcesPrefix) {
		p.parse = p.resourceValues
	}
	return true
}

// resourceValues parses the resource values of the current
// resource block.
func (p *parser) resourceValues() bool {
	line, ok := p.in.pop()
	if !ok {
		return false
	}
	trimmed := strings.Trim(line, " \t")
	switch {
	case trimmed == "limits:" || trimmed == "requests:":
		p.out.push(trimmed)
	case strings.HasPrefix(trimmed, "memory:") || strings.HasPrefix(trimmed, "cpu:"):
		p.out.push(trimmed)
	case strings.HasPrefix(trimmed, "#"):
		// Ignore comment.
	default:
		// All done, move back to possible next container name.
		p.in.pushBack(line)
		p.parse = p.containerName
	}
	return true
}
