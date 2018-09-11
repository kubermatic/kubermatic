package kubernetes

import (
	"errors"

	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

var (
	// ErrAlreadyExist an error indicating that the the resource already exists
	ErrAlreadyExist = errors.New("AlreadyExist")
)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// createImpersonationClientWrapper is a helper method that spits back kubermatic client that uses user impersonation
func createImpersonationClientWrapper(user *kubermaticapiv1.User, projectInternalName string, createImpersonationClient kubermaticImpersonationClient) (kubermaticclientv1.KubermaticV1Interface, error) {
	if user == nil || len(projectInternalName) == 0 {
		return nil, errors.New("a project and/or a user is missing but required")
	}
	groupName, err := user.GroupForProject(projectInternalName)
	if err != nil {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, projectInternalName, err)
	}
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: user.Spec.Email,
		Groups:   []string{groupName},
	}
	return createImpersonationClient(impersonationCfg)
}

// createImpersonationClientWrapperFromUserInfo is a helper method that spits back kubermatic client that uses user impersonation
func createImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubermaticImpersonationClient) (kubermaticclientv1.KubermaticV1Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}
