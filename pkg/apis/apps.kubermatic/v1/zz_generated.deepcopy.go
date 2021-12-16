//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright  The Kubermatic Kubernetes Platform contributors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationConstraints) DeepCopyInto(out *ApplicationConstraints) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationConstraints.
func (in *ApplicationConstraints) DeepCopy() *ApplicationConstraints {
	if in == nil {
		return nil
	}
	out := new(ApplicationConstraints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationDefinition) DeepCopyInto(out *ApplicationDefinition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationDefinition.
func (in *ApplicationDefinition) DeepCopy() *ApplicationDefinition {
	if in == nil {
		return nil
	}
	out := new(ApplicationDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationDefinition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationDefinitionList) DeepCopyInto(out *ApplicationDefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApplicationDefinition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationDefinitionList.
func (in *ApplicationDefinitionList) DeepCopy() *ApplicationDefinitionList {
	if in == nil {
		return nil
	}
	out := new(ApplicationDefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationDefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationDefinitionSpec) DeepCopyInto(out *ApplicationDefinitionSpec) {
	*out = *in
	if in.Versions != nil {
		in, out := &in.Versions, &out.Versions
		*out = make([]ApplicationVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationDefinitionSpec.
func (in *ApplicationDefinitionSpec) DeepCopy() *ApplicationDefinitionSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationDefinitionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationInstallation) DeepCopyInto(out *ApplicationInstallation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationInstallation.
func (in *ApplicationInstallation) DeepCopy() *ApplicationInstallation {
	if in == nil {
		return nil
	}
	out := new(ApplicationInstallation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationInstallation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationInstallationCondition) DeepCopyInto(out *ApplicationInstallationCondition) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationInstallationCondition.
func (in *ApplicationInstallationCondition) DeepCopy() *ApplicationInstallationCondition {
	if in == nil {
		return nil
	}
	out := new(ApplicationInstallationCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationInstallationList) DeepCopyInto(out *ApplicationInstallationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApplicationInstallation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationInstallationList.
func (in *ApplicationInstallationList) DeepCopy() *ApplicationInstallationList {
	if in == nil {
		return nil
	}
	out := new(ApplicationInstallationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationInstallationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationInstallationSpec) DeepCopyInto(out *ApplicationInstallationSpec) {
	*out = *in
	out.ApplicationRef = in.ApplicationRef
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make(json.RawMessage, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationInstallationSpec.
func (in *ApplicationInstallationSpec) DeepCopy() *ApplicationInstallationSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationInstallationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationInstallationStatus) DeepCopyInto(out *ApplicationInstallationStatus) {
	*out = *in
	in.LastUpdated.DeepCopyInto(&out.LastUpdated)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ApplicationInstallationCondition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationInstallationStatus.
func (in *ApplicationInstallationStatus) DeepCopy() *ApplicationInstallationStatus {
	if in == nil {
		return nil
	}
	out := new(ApplicationInstallationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationRef) DeepCopyInto(out *ApplicationRef) {
	*out = *in
	out.Version = in.Version
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationRef.
func (in *ApplicationRef) DeepCopy() *ApplicationRef {
	if in == nil {
		return nil
	}
	out := new(ApplicationRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationSource) DeepCopyInto(out *ApplicationSource) {
	*out = *in
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(HelmSource)
		(*in).DeepCopyInto(*out)
	}
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(GitSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationSource.
func (in *ApplicationSource) DeepCopy() *ApplicationSource {
	if in == nil {
		return nil
	}
	out := new(ApplicationSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationTemplate) DeepCopyInto(out *ApplicationTemplate) {
	*out = *in
	in.Source.DeepCopyInto(&out.Source)
	if in.FormSpec != nil {
		in, out := &in.FormSpec, &out.FormSpec
		*out = make([]FormField, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationTemplate.
func (in *ApplicationTemplate) DeepCopy() *ApplicationTemplate {
	if in == nil {
		return nil
	}
	out := new(ApplicationTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationVersion) DeepCopyInto(out *ApplicationVersion) {
	*out = *in
	out.Constraints = in.Constraints
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationVersion.
func (in *ApplicationVersion) DeepCopy() *ApplicationVersion {
	if in == nil {
		return nil
	}
	out := new(ApplicationVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FormField) DeepCopyInto(out *FormField) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FormField.
func (in *FormField) DeepCopy() *FormField {
	if in == nil {
		return nil
	}
	out := new(FormField)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitCredentials) DeepCopyInto(out *GitCredentials) {
	*out = *in
	if in.Username != nil {
		in, out := &in.Username, &out.Username
		*out = new(GlobalSecretKeySelector)
		**out = **in
	}
	if in.Password != nil {
		in, out := &in.Password, &out.Password
		*out = new(GlobalSecretKeySelector)
		**out = **in
	}
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(GlobalSecretKeySelector)
		**out = **in
	}
	if in.SSHKey != nil {
		in, out := &in.SSHKey, &out.SSHKey
		*out = new(GlobalSecretKeySelector)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitCredentials.
func (in *GitCredentials) DeepCopy() *GitCredentials {
	if in == nil {
		return nil
	}
	out := new(GitCredentials)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitSource) DeepCopyInto(out *GitSource) {
	*out = *in
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(GitCredentials)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitSource.
func (in *GitSource) DeepCopy() *GitSource {
	if in == nil {
		return nil
	}
	out := new(GitSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GlobalSecretKeySelector) DeepCopyInto(out *GlobalSecretKeySelector) {
	*out = *in
	out.SecretReference = in.SecretReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalSecretKeySelector.
func (in *GlobalSecretKeySelector) DeepCopy() *GlobalSecretKeySelector {
	if in == nil {
		return nil
	}
	out := new(GlobalSecretKeySelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmCredentials) DeepCopyInto(out *HelmCredentials) {
	*out = *in
	out.Username = in.Username
	out.Password = in.Password
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmCredentials.
func (in *HelmCredentials) DeepCopy() *HelmCredentials {
	if in == nil {
		return nil
	}
	out := new(HelmCredentials)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmSource) DeepCopyInto(out *HelmSource) {
	*out = *in
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(HelmCredentials)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmSource.
func (in *HelmSource) DeepCopy() *HelmSource {
	if in == nil {
		return nil
	}
	out := new(HelmSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Version) DeepCopyInto(out *Version) {
	*out = *in
	out.Version = in.Version
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Version.
func (in *Version) DeepCopy() *Version {
	if in == nil {
		return nil
	}
	out := new(Version)
	in.DeepCopyInto(out)
	return out
}
