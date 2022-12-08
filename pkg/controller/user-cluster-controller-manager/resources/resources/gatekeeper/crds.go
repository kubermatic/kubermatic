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

package gatekeeper

import (
	_ "embed"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	//go:embed static/config-customresourcedefinition.yaml
	configYAML string

	//go:embed static/constrainttemplate-customresourcedefinition.yaml
	constraintTemplateYAML string

	//go:embed static/constrainttemplatepodstatus-customresourcedefinition.yaml
	constraintTemplatePodStatusYAML string

	//go:embed static/constraintpodstatus-customresourcedefinition.yaml
	constraintPodStatusYAML string

	//go:embed static/mutatorpodstatus-customresourcedefinition.yaml
	mutatorPodStatusYAML string

	//go:embed static/assign-customresourcedefinition.yaml
	assignYAML string

	//go:embed static/assignmetadata-customresourcedefinition.yaml
	assignMetadataYAML string

	//go:embed static/modifyset-customresourcedefinition.yaml
	modifySetYAML string

	//go:embed static/provider-customresourcedefinition.yaml
	providerYAML string
)

// ConfigCRDReconciler returns the gatekeeper config CRD definition.
func ConfigCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperConfigCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(configYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

// ConstraintTemplateCRDReconciler returns the gatekeeper constraintTemplate CRD definition.
func ConstraintTemplateCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperConstraintTemplateCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(constraintTemplateYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

// ConstraintPodStatusCRDReconciler returns the gatekeeper ConstraintPodStatus CRD definition.
func ConstraintPodStatusCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperConstraintPodStatusCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(constraintPodStatusYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

// ConstraintTemplatePodStatusCRDReconciler returns the gatekeeper ConstraintTemplatePodStatus CRD definition.
func ConstraintTemplatePodStatusCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperConstraintTemplatePodStatusCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(constraintTemplatePodStatusYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

func MutatorPodStatusCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperMutatorPodStatusCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(mutatorPodStatusYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

func AssignCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperAssignCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(assignYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

func AssignMetadataCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperAssignMetadataCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(assignMetadataYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

func ModifySetCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperModifySetCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(modifySetYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}

func ProviderCRDReconciler() reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return resources.GatekeeperProviderCRDName, func(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			var fileCRD *apiextensionsv1.CustomResourceDefinition
			err := yaml.UnmarshalStrict([]byte(providerYAML), &fileCRD)
			if err != nil {
				return nil, err
			}

			crd.Labels = fileCRD.Labels
			crd.Annotations = fileCRD.Annotations
			crd.Spec = fileCRD.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return crd, nil
		}
	}
}
