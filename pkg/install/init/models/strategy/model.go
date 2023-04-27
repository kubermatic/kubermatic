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

package strategy

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Choice interface {
	Name() string
	Description() string
}

type Model struct {
	Choices      []Choice
	ActiveChoice int
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

}
