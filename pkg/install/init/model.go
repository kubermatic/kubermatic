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
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/indent"
)

const (
	StepDomain int = iota
	StepExposeStrategy
)

type model struct {
	Quitting bool

	domain textinput.Model
	pages  paginator.Model
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.pages.Page {
	case StepDomain:
		return m.domainUpdate(msg)
	case StepExposeStrategy:
		return m.exposeStrategyUpdate(msg)
	}

	// Make sure these keys always quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "esc" || k == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	var s string

	if m.Quitting {
		s = "See you later!\n\n"
	}

	switch m.pages.Page {
	case StepDomain:
		s = m.domainView()
		break
	case StepExposeStrategy:
		s = m.exposeStrategyView()
		break
	}

	return fmt.Sprintf("\n%s", indent.String(s, 2))
}
