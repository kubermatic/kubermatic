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

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func Run() error {
	items := tem{
		item{title: "Nutella", desc: "It's good on toast"},
		item{title: "Nutella2", desc: "It's good on toast"},
		item{title: "Nutella3", desc: "It's good on toast"},
	}

	p := tea.NewProgram(initialModel(items), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("could not start program: %w", err)
	}

	return nil
}

func initialModel(items []list.Item) model {
	domainInput := textinput.New()
	domainInput.Placeholder = "kubermatic.example.com"
	domainInput.Focus()
	domainInput.CharLimit = 156
	domainInput.Width = 40

	exposeStrategyList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	exposeStrategyList.SetShowHelp(false)
	exposeStrategyList.SetShowStatusBar(false)
	exposeStrategyList.SetShowTitle(false)

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(2)

	return model{
		domain:         domainInput,
		exposeStrategy: exposeStrategyList,
		pages:          p,
	}
}
