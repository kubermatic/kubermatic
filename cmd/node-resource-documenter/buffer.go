package main

import (
	"io"
)

// buffer contains the lines of the YAML file for the parsing
// state machine and allows to push a line back when it doesn't
// match the current state.
type buffer struct {
	lines []string
}

func newBuffer() *buffer {
	return &buffer{}
}

func (b *buffer) push(lines ...string) {
	b.lines = append(b.lines, lines...)
}

func (b *buffer) pushAll(ba *buffer) {
	if ba != nil {
		b.lines = append(b.lines, ba.lines...)
	}
}

func (b *buffer) writeAll(w io.Writer) error {
	for _, line := range b.lines {
		_, err := w.Write([]byte(line))
		if err != nil {
			return err
		}
	}
	return nil
}
