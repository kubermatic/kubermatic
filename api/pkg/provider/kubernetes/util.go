package kubernetes

import (
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	restclient "k8s.io/client-go/rest"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// createImpersonationClientWrapperFromUserInfo is a helper method that spits back kubermatic client that uses user impersonation
func createImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubermaticImpersonationClient) (kubermaticclientv1.KubermaticV1Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}
