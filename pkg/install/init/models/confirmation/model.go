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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/init/generator"
	"k8c.io/kubermatic/v2/pkg/install/init/types"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"k8c.io/kubermatic/v2/pkg/install/init/models/choice"
	"k8c.io/kubermatic/v2/pkg/install/init/models/input"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func New(domain *input.Model, exposeStrategy *choice.Model, configCh chan<- generator.Config) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	columns := []table.Column{
		{Title: "Option", Width: 30},
		{Title: "Value", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	style.Selected = style.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(style)

	return &Model{
		domain:         domain,
		exposeStrategy: exposeStrategy,

		spin:        s,
		configTable: t,

		configCh: configCh,
	}
}

type Model struct {
	domain         *input.Model
	exposeStrategy *choice.Model

	spin        spinner.Model
	configTable table.Model

	Generating bool
	Generated  bool

	configCh chan<- generator.Config
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if !m.Generating {
		m.configTable.SetRows([]table.Row{
			{"DNS Name", m.domain.Value()},
			{"Expose Strategy", m.exposeStrategy.Value()},
		})

		m.configTable, cmd = m.configTable.Update(msg)
	}

	switch msg.(type) {
	case types.TickMsg:
		// we only want to make the spinner update if we're actually generating config right now.
		if m.Generating {
			m.spin, cmd = m.spin.Update(spinner.TickMsg{})
		}
		return m, cmd
	case tea.KeyMsg:
		keyMsg := msg.(tea.KeyMsg)
		if keyMsg.Type == tea.KeyEnter && !m.Generating {
			m.configCh <- generator.Config{
				DNS:            m.domain.Value(),
				ExposeStrategy: kubermaticv1.ExposeStrategy(m.exposeStrategy.Value()),
			}
			m.Generating = true
			return m, cmd
		}
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

		b.WriteString(baseStyle.Render(m.configTable.View()) + "\n")

		b.WriteString("\n\n")
		b.WriteString("Hit <enter> to generate configuration files in the current directory.")
	}

	return b.String()
}
