package openshift

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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
			originalContent["status"] = infrastructureStatus{Platform: platform}
			u.SetUnstructuredContent(originalContent)

			return u, nil
		}
	}
}
