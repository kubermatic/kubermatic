/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package collectors

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	projectPrefix = "kubermatic_project_"
)

// ProjectCollector exports metrics for project resources.
type ProjectCollector struct {
	client ctrlruntimeclient.Reader

	projectInfo *prometheus.Desc
}

// MustRegisterProjectCollector registers the project collector at the given prometheus registry.
func MustRegisterProjectCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	cc := &ProjectCollector{
		client: client,
		projectInfo: prometheus.NewDesc(
			projectPrefix+"info",
			"Additional project information",
			[]string{
				"name",
				"display_name",
				"owner",
				"phase",
			},
			nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors.
func (cc ProjectCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

// Collect gets called by prometheus to collect the metrics.
func (cc ProjectCollector) Collect(ch chan<- prometheus.Metric) {
	projects := &kubermaticv1.ProjectList{}
	if err := cc.client.List(context.Background(), projects); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list projects in ProjectCollector: %w", err))
		return
	}

	kubernetesLabelSet := sets.New[string]()
	for _, project := range projects.Items {
		kubernetesLabelSet = kubernetesLabelSet.Union(sets.KeySet(project.Labels))
	}

	kubernetesLabels := caseInsensitiveSort(sets.List(kubernetesLabelSet))

	prometheusLabels := convertToPrometheusLabels(kubernetesLabels)
	labelsGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: projectPrefix + "labels",
		Help: "Kubernetes labels on Project resources",
	}, append([]string{"name"}, prometheusLabels...))

	for _, project := range projects.Items {
		cc.collectProject(ch, &project, kubernetesLabels, labelsGauge)
	}
}

func (cc *ProjectCollector) collectProject(ch chan<- prometheus.Metric, p *kubermaticv1.Project, kubernetesLabels []string, labelsGaugeVec *prometheus.GaugeVec) {
	owner := ""
	for _, ref := range p.OwnerReferences {
		if ref.APIVersion == kubermaticv1.SchemeGroupVersion.String() && ref.Kind == "User" {
			owner = ref.Name
			break
		}
	}

	ch <- prometheus.MustNewConstMetric(
		cc.projectInfo,
		prometheus.GaugeValue,
		1,
		p.Name,
		p.Spec.Name,
		owner,
		string(p.Status.Phase),
	)

	// assemble the labels for this project, in the order given by kubernetesLabels, but
	// taking special care of label key conflicts
	projectLabels := []string{p.Name}
	usedLabels := sets.New[string]()
	for _, key := range kubernetesLabels {
		prometheusLabel := convertToPrometheusLabel(key)
		if !usedLabels.Has(prometheusLabel) {
			projectLabels = append(projectLabels, p.Labels[key])
			usedLabels.Insert(prometheusLabel)
		}
	}

	labelsGaugeVec.WithLabelValues(projectLabels...).Collect(ch)
}
