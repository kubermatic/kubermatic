// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	sets "k8s.io/apimachinery/pkg/util/sets"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticAPIConfiguration) DeepCopyInto(out *KubermaticAPIConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticAPIConfiguration.
func (in *KubermaticAPIConfiguration) DeepCopy() *KubermaticAPIConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticAPIConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticAddonConfiguration) DeepCopyInto(out *KubermaticAddonConfiguration) {
	*out = *in
	if in.Default != nil {
		in, out := &in.Default, &out.Default
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticAddonConfiguration.
func (in *KubermaticAddonConfiguration) DeepCopy() *KubermaticAddonConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticAddonConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticAddonsConfiguration) DeepCopyInto(out *KubermaticAddonsConfiguration) {
	*out = *in
	in.Kubernetes.DeepCopyInto(&out.Kubernetes)
	in.Openshift.DeepCopyInto(&out.Openshift)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticAddonsConfiguration.
func (in *KubermaticAddonsConfiguration) DeepCopy() *KubermaticAddonsConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticAddonsConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticAuthConfiguration) DeepCopyInto(out *KubermaticAuthConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticAuthConfiguration.
func (in *KubermaticAuthConfiguration) DeepCopy() *KubermaticAuthConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticAuthConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticConfiguration) DeepCopyInto(out *KubermaticConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticConfiguration.
func (in *KubermaticConfiguration) DeepCopy() *KubermaticConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubermaticConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticConfigurationList) DeepCopyInto(out *KubermaticConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubermaticConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticConfigurationList.
func (in *KubermaticConfigurationList) DeepCopy() *KubermaticConfigurationList {
	if in == nil {
		return nil
	}
	out := new(KubermaticConfigurationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubermaticConfigurationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticConfigurationSpec) DeepCopyInto(out *KubermaticConfigurationSpec) {
	*out = *in
	out.Auth = in.Auth
	if in.FeatureGates != nil {
		in, out := &in.FeatureGates, &out.FeatureGates
		*out = make(sets.String, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.UI = in.UI
	out.API = in.API
	in.SeedController.DeepCopyInto(&out.SeedController)
	out.MasterController = in.MasterController
	if in.MasterFiles != nil {
		in, out := &in.MasterFiles, &out.MasterFiles
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticConfigurationSpec.
func (in *KubermaticConfigurationSpec) DeepCopy() *KubermaticConfigurationSpec {
	if in == nil {
		return nil
	}
	out := new(KubermaticConfigurationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticMasterControllerConfiguration) DeepCopyInto(out *KubermaticMasterControllerConfiguration) {
	*out = *in
	out.ProjectsMigrator = in.ProjectsMigrator
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticMasterControllerConfiguration.
func (in *KubermaticMasterControllerConfiguration) DeepCopy() *KubermaticMasterControllerConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticMasterControllerConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticProjectsMigratorConfiguration) DeepCopyInto(out *KubermaticProjectsMigratorConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticProjectsMigratorConfiguration.
func (in *KubermaticProjectsMigratorConfiguration) DeepCopy() *KubermaticProjectsMigratorConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticProjectsMigratorConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticSeedControllerConfiguration) DeepCopyInto(out *KubermaticSeedControllerConfiguration) {
	*out = *in
	in.Addons.DeepCopyInto(&out.Addons)
	out.Monitoring = in.Monitoring
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticSeedControllerConfiguration.
func (in *KubermaticSeedControllerConfiguration) DeepCopy() *KubermaticSeedControllerConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticSeedControllerConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermaticUIConfiguration) DeepCopyInto(out *KubermaticUIConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermaticUIConfiguration.
func (in *KubermaticUIConfiguration) DeepCopy() *KubermaticUIConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermaticUIConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubermsaticSeedMonitoringConfiguration) DeepCopyInto(out *KubermsaticSeedMonitoringConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubermsaticSeedMonitoringConfiguration.
func (in *KubermsaticSeedMonitoringConfiguration) DeepCopy() *KubermsaticSeedMonitoringConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubermsaticSeedMonitoringConfiguration)
	in.DeepCopyInto(out)
	return out
}
