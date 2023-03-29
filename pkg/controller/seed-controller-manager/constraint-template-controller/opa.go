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

package constrainttemplatecontroller

import (
	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	openpolicyagent "k8c.io/api/v3/pkg/apis/open-policy-agent"
)

func convertConstraintTemplateSpec(kkpSpec *kubermaticv1.ConstraintTemplateSpec) (*constrainttemplatev1.ConstraintTemplateSpec, error) {
	spec := constrainttemplatev1.ConstraintTemplateSpec{
		CRD:     convertConstraintTemplateCRD(kkpSpec.CRD),
		Targets: []constrainttemplatev1.Target{},
	}

	for _, target := range kkpSpec.Targets {
		spec.Targets = append(spec.Targets, constrainttemplatev1.Target{
			Target: target.Target,
			Rego:   target.Rego,
			Libs:   target.Libs,
		})
	}

	return &spec, nil
}

func convertConstraintTemplateCRD(kkpSpec openpolicyagent.CRD) constrainttemplatev1.CRD {
	crd := constrainttemplatev1.CRD{
		Spec: constrainttemplatev1.CRDSpec{
			Names: constrainttemplatev1.Names{
				Kind:       kkpSpec.Spec.Names.Kind,
				ShortNames: kkpSpec.Spec.Names.ShortNames,
			},
		},
	}

	if kkp := kkpSpec.Spec.Validation; kkp != nil {
		crd.Spec.Validation = &constrainttemplatev1.Validation{
			OpenAPIV3Schema: kkp.OpenAPIV3Schema,
			LegacySchema:    kkp.LegacySchema,
		}
	}

	return crd
}
