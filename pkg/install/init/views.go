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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (m model) pagination() string {
	var b strings.Builder
	b.WriteString(m.pages.View())
	b.WriteString("\n\n")
	b.WriteString("←/→ page • esc: quit\n")
	return b.String()
}

func (m model) domainView() string {
	return fmt.Sprintf(
		"What DNS name should be used for your KKP configuration? \n\n%s\n\n%s",
		m.domain.View(),
		m.pagination(),
	) + "\n"
}

func (m model) domainUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			m.pages.NextPage()
			return m, nil
		}

	// We handle errors just like any other message
	case error:
		m.domain.Err = msg
		return m, nil
	}

	m.domain, cmd = m.domain.Update(msg)
	m.pages, cmd = m.pages.Update(msg)
	return m, cmd
}

func (m model) exposeStrategyView() string {
	return fmt.Sprintf(
		"How would you like to expose Kubernetes clusters created by KKP? \n\n%s\n\n%s",
		m.pagination(),
	) + "\n"
}

func (m model) exposeStrategyUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.pages, cmd = m.pages.Update(msg)
	return m, cmd
}
