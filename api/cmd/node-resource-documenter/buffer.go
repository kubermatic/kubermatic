package main

// buffer contains the lines of the YAML file for the parsing
// state machine and allows to push a line back when it doesn't
// match the current state.
type buffer struct {
	lines []string
}

func (b *buffer) push(lines ...string) {
	b.lines = append(b.lines, lines...)
}

func (b *buffer) pop() (string, bool) {
	if len(b.lines) == 0 {
		return "", false
	}
	line := b.lines[0]
	b.lines = b.lines[1:]
	return line, true
}

func (b *buffer) pushAll(ba *buffer) {
	if ba != nil {
		b.lines = append(b.lines, ba.lines...)
	}
}
