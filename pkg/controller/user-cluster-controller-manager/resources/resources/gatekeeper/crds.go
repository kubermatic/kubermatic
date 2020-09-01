package gatekeeper

import (
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

const (
	configAPIGroup               = "config.gatekeeper.sh"
	configAPIVersion             = "v1alpha1"
	constraintTemplateAPIGroup   = "templates.gatekeeper.sh"
	constraintTemplateAPIVersion = "v1beta1"
)

// ConfigCRDCreator returns the gatekeeper config CRD definition
func ConfigCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.GatekeeperConfigCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Annotations = map[string]string{"controller-gen.kubebuilder.io/version": "v0.2.4"}
			crd.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			crd.Spec.Group = configAPIGroup
			crd.Spec.Version = configAPIVersion
			crd.Spec.Versions = []apiextensionsv1beta1.CustomResourceDefinitionVersion{
				{Name: configAPIVersion, Served: true, Storage: true},
			}
			crd.Spec.Scope = apiextensionsv1beta1.NamespaceScoped
			crd.Spec.Names.Kind = "Config"
			crd.Spec.Names.ListKind = "ConfigList"
			crd.Spec.Names.Plural = "configs"
			crd.Spec.Names.Singular = "config"

			return crd, nil
		}
	}
}

// ConstraintTemplateCRDCreator returns the gatekeeper constraintTemplate CRD definition
func ConstraintTemplateCRDCreator() reconciling.NamedCustomResourceDefinitionCreatorGetter {
	return func() (string, reconciling.CustomResourceDefinitionCreator) {
		return resources.GatekeeperConstraintTemplateCRDName, func(crd *apiextensionsv1beta1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
			crd.Annotations = map[string]string{"controller-gen.kubebuilder.io/version": "v0.2.4"}
			crd.Labels = map[string]string{"gatekeeper.sh/system": "yes", "controller-tools.k8s.io": "1.0"}
			crd.Spec.Group = constraintTemplateAPIGroup
			crd.Spec.Version = constraintTemplateAPIVersion
			crd.Spec.Versions = []apiextensionsv1beta1.CustomResourceDefinitionVersion{
				{Name: constraintTemplateAPIVersion, Served: true, Storage: true},
				{Name: "v1alpha1", Served: true, Storage: false},
			}
			crd.Spec.Scope = apiextensionsv1beta1.ClusterScoped
			crd.Spec.Names.Kind = "ConstraintTemplate"
			crd.Spec.Names.Plural = "constrainttemplates"
			crd.Spec.Subresources = &apiextensionsv1beta1.CustomResourceSubresources{Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{}}

			return crd, nil
		}
	}
}
