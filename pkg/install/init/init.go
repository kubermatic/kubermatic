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

	"k8c.io/kubermatic/v2/pkg/install/init/models/choice"
	"k8c.io/kubermatic/v2/pkg/install/init/models/input"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func Run() error {

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("could not start program: %w", err)
	}

	return nil
}

func initialModel() model {

	strategies := []choice.Choice{
		item{name: "Tunneling", desc: "The Tunneling expose strategy addresses both the scaling issues of the NodePort strategy and cost issues of the LoadBalancer strategy. With this strategy, the traffic is routed to the based on a combination of SNI and HTTP/2 tunnels by the nodeport-proxy."},
		item{name: "LoadBalancer", desc: "In the LoadBalancer expose strategy, a dedicated service of type LoadBalancer will be created for each user cluster. This strategy requires services of type LoadBalancer to be available on the Seed cluster and usually results into higher cost of cloud resources."},
		item{name: "NodePort", desc: "NodePort is the default expose strategy in KKP. With this strategy a k8s service of type NodePort is created for each exposed component (e.g. Kubernetes API Server) of each user cluster. This implies, that each apiserver will be exposed on a randomly assigned TCP port from the nodePort range configured for the seed cluster."},
	}

	generateSecretChoices := []choice.Choice{
		item{name: "Yes", desc: "Secrets for kubermatic-api and dex will be generated on the current system. Please note that the entropy on your system might not be sufficient for your organisation's security standards."},
		item{name: "No", desc: "Secrets will not be generated and need to be manually added before installing KKP."},
	}

	// all steps in the wizard.
	hostnameInput := input.New(40, 156, "What DNS name should be used for your KKP installation?", "kubermatic.example.com")

	exposeStrategyChoice := choice.New("What expose strategy do you want to use for Kubernetes clusters created by this environment?", strategies)

	secretGenerationChoice := choice.New("Would you like this wizard to generate secrets for your configuration?", generateSecretChoices)

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	// we don't want to "hijack" the vim-style keybindings that happen by default here.
	// instead we use alt+arrows and enter for a more natural navigation.
	p.KeyMap.NextPage = key.NewBinding(key.WithKeys("alt+right", "enter"))
	p.KeyMap.PrevPage = key.NewBinding(key.WithKeys("alt+left"))

	p.SetTotalPages(3)

	return model{
		domain:           hostnameInput,
		exposestrategy:   exposeStrategyChoice,
		secretGeneration: secretGenerationChoice,

		pages: p,
	}
}
