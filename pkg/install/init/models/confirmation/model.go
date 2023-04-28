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

package confirmation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"k8c.io/kubermatic/v2/pkg/install/init/models/choice"
	"k8c.io/kubermatic/v2/pkg/install/init/models/input"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func New(domain *input.Model, exposeStrategy *choice.Model) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Model{
		domain:         domain,
		exposeStrategy: exposeStrategy,

		spin: s,
	}
}

type Model struct {
	domain         *input.Model
	exposeStrategy *choice.Model

	spin spinner.Model

	Generating bool
	Generated  bool
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.Type == tea.KeyEnter && !m.Generating {
			m.Generating = true
			// start the spinner by returning its tick
			return m, m.spin.Tick
		}
	}

	if m.Generating {
		m.spin, cmd = m.spin.Update(msg)
	}

	return m, cmd
}

func (m *Model) View() string {
	var b strings.Builder

	if m.Generated {

	} else if m.Generating {
		b.WriteString(fmt.Sprintf("%s Generating configuration ...", m.spin.View()))
	} else {
		b.WriteString("Please review your settings:\n\n")
		b.WriteString(fmt.Sprintf("DNS Name: %s\n", m.domain.Value()))
		b.WriteString(fmt.Sprintf("Expose Strategy: %s\n", m.exposeStrategy.Value()))

		b.WriteString("\n\n")
		b.WriteString("Hit <enter> to generate configuration files in the current directory.")
	}

	return b.String()
}
