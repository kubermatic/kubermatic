/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package choice

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

var (
	term = termenv.EnvColorProfile()
)

type Choice interface {
	Name() string
	Description() string
}

func New(question string, choices []Choice) *Model {
	return &Model{
		Question: question,
		Choices:  choices,
	}
}

type Model struct {
	Question     string
	Choices      []Choice
	ActiveChoice int
	Chosen       bool
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.ActiveChoice < len(m.Choices)-1 {
				m.ActiveChoice++
			}
		case "k", "up":
			if m.ActiveChoice > 0 {
				m.ActiveChoice--
			}
		case "enter":
			m.Chosen = true
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s\n\n", m.Question))

	for i, item := range m.Choices {
		b.WriteString(checkbox(fmt.Sprintf("%s", item.Name()), m.ActiveChoice == i))
		b.WriteString("\n")
	}

	b.WriteString("\n\n")
	b.WriteString(wordwrap.String(m.Choices[m.ActiveChoice].Description(), 80))

	return b.String()
}

func checkbox(label string, checked bool) string {
	if checked {
		return colorFg("[x] "+label, "212")
	}
	return fmt.Sprintf("[ ] %s", label)
}

// Color a string's foreground with the given value.
func colorFg(val, color string) string {
	return termenv.String(val).Foreground(term.Color(color)).String()
}
