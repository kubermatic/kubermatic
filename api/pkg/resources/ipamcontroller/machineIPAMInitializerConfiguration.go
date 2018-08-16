package ipamcontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
)

// MachineIPAMInitializerConfiguration returns the Initializer Configuration which makes sure that machine resoures are initialized by the ipam controller
func MachineIPAMInitializerConfiguration(data *resources.TemplateData, existing *admissionv1alpha1.InitializerConfiguration) (*admissionv1alpha1.InitializerConfiguration, error) {
	var iniCfg *admissionv1alpha1.InitializerConfiguration
	if existing != nil {
		iniCfg = existing
	} else {
		iniCfg = &admissionv1alpha1.InitializerConfiguration{}
	}

	iniCfg.Name = resources.MachineIPAMInitializerConfigurationName

	iniCfg.Initializers = []admissionv1alpha1.Initializer{
		{
			Name: resources.MachineIPAMInitializerName,
			Rules: []admissionv1alpha1.Rule{
				{
					APIGroups:   []string{"machine.k8s.io"},
					APIVersions: []string{"*"},
					Resources:   []string{"machines"},
				},
			},
		},
	}

	return iniCfg, nil
}
