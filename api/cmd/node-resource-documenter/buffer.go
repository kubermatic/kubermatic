package main

// buffer contains the lines of the YAML file for the parsing
// state machine and allows to push a line back when it doesn't
// match the current state.
type buffer struct {
	lines []string
}

func (b *buffer) isEmpty() bool {
	return len(b.lines) == 0
}

func (b *buffer) push(line string) {
	b.lines = append(b.lines, line)
}

func (b *buffer) pop() (string, bool) {
	if len(b.lines) == 0 {
		return "", false
	}
	line := b.lines[0]
	b.lines = b.lines[1:]
	return line, true
}

func (b *buffer) pushBack(line string) {
	b.lines = append([]string{line}, b.lines...)
}

func (b *buffer) pushAll(ba *buffer) {
	b.lines = append(b.lines, ba.lines...)
}
