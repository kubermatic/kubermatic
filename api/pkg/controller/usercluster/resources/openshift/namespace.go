package openshift

import (
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
