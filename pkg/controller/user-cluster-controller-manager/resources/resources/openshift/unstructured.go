package openshift

import (
	"fmt"

	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type infrastructureStatus struct {
	Platform string `json:"platform"`
}

// InfrastructureCreatorGetter returns the Infrastructure object. It is needed by the
// cloud-credential-operator.
func InfrastructureCreatorGetter(platform string) reconciling.NamedUnstructuredCreatorGetter {
	return func() (string, string, string, reconciling.UnstructuredCreator) {
		return "cluster", "Infrastructure", "config.openshift.io/v1", func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {

			originalContent := u.Object
			if originalContent == nil {
				originalContent = map[string]interface{}{}
			}
			// Spec must not be empty, but avoid overwriting anything
			if _, ok := originalContent["spec"]; !ok {
				originalContent["spec"] = struct{}{}
			}
			originalContent["status"] = infrastructureStatus{Platform: translateKubernetesCloudProviderName(platform)}
			u.SetUnstructuredContent(originalContent)

			return u, nil
		}
	}
}

// From https://github.com/openshift/cloud-credential-operator/blob/ec6f38d73a7921e79d0ca7555da3a864e808e681/vendor/github.com/openshift/api/config/v1/types_infrastructure.go#L64-L87
func translateKubernetesCloudProviderName(kubernetesCloudProviderName string) string {
	switch kubernetesCloudProviderName {
	case "aws":
		return "AWS"
	case "azure":
		return "Azure"
	case "gcp":
		return "GCP"
	case "openstack":
		return "OpenStack"
	case "vsphere":
		return "VSphere"
	default:
		return "None"
	}
}

// ClusterVersionCreatorGetter returns the ClusterVersionCreator
func ClusterVersionCreatorGetter(clusterNamespaceName string) reconciling.NamedUnstructuredCreatorGetter {
	return func() (string, string, string, reconciling.UnstructuredCreator) {
		return "version", "ClusterVersion", "config.openshift.io/v1", func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {

			if u.Object == nil {
				u.Object = map[string]interface{}{}
			}
			u.Object["spec"] = struct {
				// Used by the AWS CloudCredentialActuator to tag resources:
				// https://github.com/openshift/cloud-credential-operator/blob/ec6f38d73a7921e79d0ca7555da3a864e808e681/pkg/aws/actuator/actuator.go#L192
				// We could the `cluster-` prefix but it allows us to identify our tags and wont harm.
				ClusterID string `json:"clusterID"`
			}{
				ClusterID: clusterNamespaceName,
			}
			return u, nil
		}
	}
}

// ConsoleOAuthClientName is the name of the OAuthClient object created for the openshift console
const ConsoleOAuthClientName = "console"

func ConsoleOAuthClientCreator(consoleCallbackURI string) reconciling.NamedUnstructuredCreatorGetter {
	return func() (string, string, string, reconciling.UnstructuredCreator) {
		return ConsoleOAuthClientName, "OAuthClient", "oauth.openshift.io/v1", func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {

			if u.Object == nil {
				u.Object = map[string]interface{}{}
			}
			if _, exists := u.Object["secret"]; !exists {
				secret, err := generateNewSecret()
				if err != nil {
					return nil, fmt.Errorf("failed to generate secret: %v", err)
				}
				u.Object["secret"] = secret
			}

			u.Object["grantMethod"] = "auto"
			u.Object["redirectURIs"] = []string{consoleCallbackURI}

			return u, nil
		}
	}
}
