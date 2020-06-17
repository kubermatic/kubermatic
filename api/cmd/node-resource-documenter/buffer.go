/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
