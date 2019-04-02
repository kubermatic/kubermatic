package cluster

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"k8s.io/client-go/kubernetes"
)

func (cc *Controller) reconcileUserClusterResources(cluster *kubermaticv1.Cluster, client kubernetes.Interface) (*kubermaticv1.Cluster, error) {
	var err error
	if err = cc.launchingCreateOpenVPNClientCertificates(cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
