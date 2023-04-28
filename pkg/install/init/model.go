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

package init

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/termenv"
)

const (
	StepDomain int = iota
	StepExposeStrategy
)

var (
	term   = termenv.EnvColorProfile()
	subtle = makeFgStyle("241")
)

type model struct {
	Quitting bool

	models []tea.Model

	pages paginator.Model
}

type item struct {
	name, desc string
}

func (i item) Name() string        { return i.name }
func (i item) Description() string { return i.desc }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "esc" || k == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	m.pages, cmd = m.pages.Update(msg)

	m.models[m.pages.Page], cmd = m.models[m.pages.Page].Update(msg)

	return m, cmd
}

func (m model) View() string {
	var b strings.Builder

	if m.Quitting {
		return indent.String("See you later!\n\n", 2)
	}

	b.WriteString(m.models[m.pages.Page].View())

	b.WriteString("\n\n")
	b.WriteString(m.pages.View())
	b.WriteString("\n\n")
	b.WriteString(subtle("alt+←/→: page • esc: quit\n"))

	return fmt.Sprintf("\n%s", indent.String(b.String(), 2))
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// Return a function that will colorize the foreground of a given string.
func makeFgStyle(color string) func(string) string {
	return termenv.Style{}.Foreground(term.Color(color)).Styled
}
