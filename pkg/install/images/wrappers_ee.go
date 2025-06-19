//go:build ee

/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package images

import (
	"fmt"
	"iter"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/metering"
	velero "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/user-cluster/velero-controller/resources"
	appcat "k8c.io/kubermatic/v2/pkg/ee/default-application-catalog"
	kubelb "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/seed-cluster"
	kyverno "k8c.io/kubermatic/v2/pkg/ee/kyverno"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
)

// getAdditionalImagesFromReconcilers returns the images used by the reconcilers for Enterprise Edition addons/components.
func getAdditionalImagesFromReconcilers(templateData *resources.TemplateData) (images []string, err error) {
	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		kubelb.DeploymentReconciler(templateData),
		velero.DeploymentReconciler(templateData),
	}

	deploymentReconcilers = append(deploymentReconcilers, kyverno.GetDeploymentReconcilers(templateData)...)

	for _, createFunc := range deploymentReconcilers {
		_, dpCreator := createFunc()
		deployment, err := dpCreator(&appsv1.Deployment{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(deployment.Spec.Template.Spec)...)
	}

	_, dsCreator := velero.DaemonSetReconciler(templateData)()
	daemonset, err := dsCreator(&appsv1.DaemonSet{})
	if err != nil {
		return nil, err
	}
	images = append(images, getImagesFromPodSpec(daemonset.Spec.Template.Spec)...)

	_, stsCreator := metering.MeteringPrometheusReconciler(templateData.RewriteImage, templateData.Seed())()
	statefulset, err := stsCreator(&appsv1.StatefulSet{})
	if err != nil {
		return nil, err
	}
	images = append(images, getImagesFromPodSpec(statefulset.Spec.Template.Spec)...)

	return images, err
}

func DefaultAppsHelmCharts(
	config *kubermaticv1.KubermaticConfiguration,
	logger logrus.FieldLogger,
	helmClient helm.Client,
	helmTimeout time.Duration,
	registryPrefix string,
) iter.Seq2[*AppsHelmChart, error] {
	log := kubermaticlog.NewDefault().Sugar()
	defaultAppDefReconcilers, err := appcat.ApplicationDefinitionReconcilerFactories(log, config, true)
	if err != nil {
		return func(yield func(*AppsHelmChart, error) bool) {
			yield(nil, fmt.Errorf("failed to get default application definition reconciler factories: %w", err))
		}
	}

	return getHelmChartRenderFunc(config, logger, helmClient, helmTimeout, registryPrefix, defaultAppDefReconcilers)
}
