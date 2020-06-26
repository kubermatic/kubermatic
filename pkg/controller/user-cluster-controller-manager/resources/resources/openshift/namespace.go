/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package openshift

import (
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/openshift/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func APIServerNSCreatorGetter() (string, reconciling.NamespaceCreator) {
	return "openshift-apiserver", func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}

		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c20,c15"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000410000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000410000/10000"
		ns.Labels["openshift.io/run-level"] = "1"

		return ns, nil
	}
}

func ControllerManagerNSCreatorGetter() (string, reconciling.NamespaceCreator) {
	return "openshift-controller-manager", func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}

		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c17,c14"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000300000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000300000/10000"
		ns.Labels["openshift.io/run-level"] = "0"

		return ns, nil
	}
}

func KubeSchedulerNSCreatorGetter() (string, reconciling.NamespaceCreator) {
	return "openshift-kube-scheduler", func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}

		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c19,c14"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000370000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000370000/10000"
		ns.Labels["openshift.io/run-level"] = "0"

		return ns, nil
	}
}

// The network-operator runs in the seed but still creates some stuff in this NS
func NetworkOperatorNSGetter() (string, reconciling.NamespaceCreator) {
	return "openshift-network-operator", func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}

		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c19,c4"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000350000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000350000/10000"
		ns.Labels["openshift.io/run-level"] = "0"

		return ns, nil
	}
}

// RegistryNSGetter is used to create the namespace in which the registry operator
// creates the registry
func RegistryNSGetter() (string, reconciling.NamespaceCreator) {
	return openshiftresources.RegistryNamespaceName, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c17,c9"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000290000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000290000/10000"

		return ns, nil
	}
}

// CloudCredentialOperatorNSGetter creates the namespace in which all credentialsrequests end up
func CloudCredentialOperatorNSGetter() (string, reconciling.NamespaceCreator) {
	return "openshift-cloud-credential-operator", func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Annotations["openshift.io/node-selector"] = ""
		ns.Annotations["openshift.io/sa.scc.mcs"] = "s0:c1,c0"
		ns.Annotations["openshift.io/sa.scc.supplemental-groups"] = "1000000000/10000"
		ns.Annotations["openshift.io/sa.scc.uid-range"] = "1000000000/10000"
		ns.Labels["controller-tools.k8s.io"] = "1.0"
		ns.Labels["openshift.io/run-level"] = "1"

		return ns, nil
	}
}
