/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package ini

import (
	"fmt"
	"io"
	"strconv"
)

type File interface {
	Section(name string, subsection string) Section
	Render(os io.Writer) error
}

type Section interface {
	AddStringKey(key, value string)
	AddBoolKey(key string, value bool)
}

type file struct {
	sections []*section
}

func New() File {
	return &file{}
}

func (f *file) Section(name string, subsection string) Section {
	if subsection != "" {
		name = fmt.Sprintf("%s %s", name, quote(subsection))
	}

	s := &section{
		name: name,
	}

	f.sections = append(f.sections, s)

	return s
}

func (f *file) Render(out io.Writer) error {
	for i, section := range f.sections {
		if err := section.render(out); err != nil {
			return err
		}

		if i < len(f.sections)-1 {
			if _, err := out.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}

	return nil
}

type section struct {
	name  string
	pairs []fmt.Stringer
}

func (s *section) AddStringKey(key, value string) {
	s.pairs = append(s.pairs, &stringPair{
		key:   key,
		value: value,
	})
}

func (s *section) AddBoolKey(key string, value bool) {
	s.pairs = append(s.pairs, &boolPair{
		key:   key,
		value: value,
	})
}

func (s *section) render(out io.Writer) error {
	if _, err := fmt.Fprintf(out, "[%s]\n", s.name); err != nil {
		return err
	}

	for _, pair := range s.pairs {
		if _, err := fmt.Fprintf(out, "%s\n", pair.String()); err != nil {
			return err
		}
	}

	return nil
}

type stringPair struct {
	key   string
	value string
}

func (p *stringPair) String() string {
	return fmt.Sprintf("%s = %s", p.key, quote(p.value))
}

type boolPair struct {
	key   string
	value bool
}

func (p *boolPair) String() string {
	return fmt.Sprintf("%s = %s", p.key, strconv.FormatBool(p.value))
}
