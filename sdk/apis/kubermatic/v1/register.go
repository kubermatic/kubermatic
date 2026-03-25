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

package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	if err := AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to add kubermatic scheme: %v", err))
	}
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// GroupName is the group name use in this package.
const (
	GroupName    = "kubermatic.k8c.io"
	GroupVersion = "v1"
)

// SchemeGroupVersion is group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

// Resource takes an unqualified resource and returns a Group qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&UserSSHKey{},
		&UserSSHKeyList{},
		&Cluster{},
		&ClusterList{},
		&EtcdBackupConfig{},
		&EtcdBackupConfigList{},
		&EtcdRestore{},
		&EtcdRestoreList{},
		&User{},
		&UserList{},
		&Project{},
		&ProjectList{},
		&Addon{},
		&AddonList{},
		&UserProjectBinding{},
		&UserProjectBindingList{},
		&Seed{},
		&SeedList{},
		&KubermaticSetting{},
		&KubermaticSettingList{},
		&AddonConfig{},
		&AddonConfigList{},
		&Preset{},
		&PresetList{},
		&AdmissionPlugin{},
		&AdmissionPluginList{},
		&ExternalCluster{},
		&ExternalClusterList{},
		&ConstraintTemplate{},
		&ConstraintTemplateList{},
		&Constraint{},
		&ConstraintList{},
		&Alertmanager{},
		&AlertmanagerList{},
		&ClusterTemplate{},
		&ClusterTemplateList{},
		&ClusterTemplateInstance{},
		&ClusterTemplateInstanceList{},
		&RuleGroup{},
		&RuleGroupList{},
		&AllowedRegistry{},
		&AllowedRegistryList{},
		&MLAAdminSetting{},
		&MLAAdminSettingList{},
		&KubermaticConfiguration{},
		&KubermaticConfigurationList{},
		&IPAMPool{},
		&IPAMPoolList{},
		&IPAMAllocation{},
		&IPAMAllocationList{},
		&ResourceQuota{},
		&ResourceQuotaList{},
		&GroupProjectBinding{},
		&GroupProjectBindingList{},
		&ClusterBackupStorageLocation{},
		&ClusterBackupStorageLocationList{},
		&PolicyTemplate{},
		&PolicyTemplateList{},
		&PolicyBinding{},
		&PolicyBindingList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
