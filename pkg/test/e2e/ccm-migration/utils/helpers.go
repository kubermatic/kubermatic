/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package utils

import (
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DefaultMachineDeployment(modifiers ...func(deployment *clusterv1alpha1.MachineDeployment)) *clusterv1alpha1.MachineDeployment {
	machineDeployment := &clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MachineDeploymentName,
			Namespace: MachineDeploymentNamespace,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": MachineDeploymentName,
				},
			},
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": MachineDeploymentName,
					},
				},
				Spec: clusterv1alpha1.MachineSpec{},
			},
		},
	}

	for _, m := range modifiers {
		m(machineDeployment)
	}

	return machineDeployment
}

func DefaultCredentialSecret(secretName string, modifiers ...func(secret *corev1.Secret)) *corev1.Secret {
	credentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: resources.KubermaticNamespace,
			Labels: map[string]string{
				"secretName": secretName,
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	for _, m := range modifiers {
		m(credentialSecret)
	}

	return credentialSecret
}

func DefaultProject(projectName string) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: "test-e2e",
		},
	}
}

func DefaultCluster(clusterName string, version semver.Semver, cloudSpec kubermaticv1.CloudSpec, projectID string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: cloudSpec,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.240.16.0/20"},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				ProxyMode: "ipvs",
			},
			ExposeStrategy:    kubermaticv1.ExposeStrategyNodePort,
			HumanReadableName: "test",
			Version:           version,
			Features: map[string]bool{
				// normally new clusters would get the external CCM enabled by default,
				// but for this test scenario we must create a cluster without external CCM.
				kubermaticv1.ClusterFeatureExternalCloudProvider: false,
			},
		},
	}
}
