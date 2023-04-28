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

package input

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// compile time check if interface is implemented
var _ tea.Model = &Model{}

func New(width, limit int, question, placeholder string) tea.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Focus()
	input.CharLimit = limit
	input.Width = width

	return &Model{
		question: question,
		input:    input,
	}
}

type Model struct {
	question string
	input    textinput.Model
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m *Model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.question,
		m.input.View(),
	) + "\n"

}
