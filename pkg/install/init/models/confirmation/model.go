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

	"k8c.io/kubermatic/v2/pkg/install/init/models/input"

	tea "github.com/charmbracelet/bubbletea"
)

func New(domain *input.Model) *Model {
	return &Model{
		domain: domain,
	}
}

type Model struct {
	domain *input.Model
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	return m, cmd
}

func (m *Model) View() string {
	return fmt.Sprintf("please review your settings: %s\n", m.domain.Value())
}
