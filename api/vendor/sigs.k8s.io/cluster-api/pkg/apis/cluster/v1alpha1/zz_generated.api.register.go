/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package v1alpha1

import (
	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster"
)

var (
	clusterClusterStorage = builders.NewApiResource( // Resource status endpoint
		cluster.InternalCluster,
		ClusterSchemeFns{},
		func() runtime.Object { return &Cluster{} },     // Register versioned resource
		func() runtime.Object { return &ClusterList{} }, // Register versioned resource list
		&ClusterStrategy{builders.StorageStrategySingleton},
	)
	clusterMachineStorage = builders.NewApiResource( // Resource status endpoint
		cluster.InternalMachine,
		MachineSchemeFns{},
		func() runtime.Object { return &Machine{} },     // Register versioned resource
		func() runtime.Object { return &MachineList{} }, // Register versioned resource list
		&MachineStrategy{builders.StorageStrategySingleton},
	)
	clusterMachineDeploymentStorage = builders.NewApiResource( // Resource status endpoint
		cluster.InternalMachineDeployment,
		MachineDeploymentSchemeFns{},
		func() runtime.Object { return &MachineDeployment{} },     // Register versioned resource
		func() runtime.Object { return &MachineDeploymentList{} }, // Register versioned resource list
		&MachineDeploymentValidationStrategy{builders.StorageStrategySingleton},
	)
	clusterMachineSetStorage = builders.NewApiResource( // Resource status endpoint
		cluster.InternalMachineSet,
		MachineSetSchemeFns{},
		func() runtime.Object { return &MachineSet{} },     // Register versioned resource
		func() runtime.Object { return &MachineSetList{} }, // Register versioned resource list
		&MachineSetStrategy{builders.StorageStrategySingleton},
	)
	ApiVersion = builders.NewApiVersion("cluster.k8s.io", "v1alpha1").WithResources(
		clusterClusterStorage,
		builders.NewApiResource( // Resource status endpoint
			cluster.InternalClusterStatus,
			ClusterSchemeFns{},
			func() runtime.Object { return &Cluster{} },     // Register versioned resource
			func() runtime.Object { return &ClusterList{} }, // Register versioned resource list
			&ClusterStatusStrategy{builders.StatusStorageStrategySingleton},
		), clusterMachineStorage,
		builders.NewApiResource( // Resource status endpoint
			cluster.InternalMachineStatus,
			MachineSchemeFns{},
			func() runtime.Object { return &Machine{} },     // Register versioned resource
			func() runtime.Object { return &MachineList{} }, // Register versioned resource list
			&MachineStatusStrategy{builders.StatusStorageStrategySingleton},
		), clusterMachineDeploymentStorage,
		builders.NewApiResource( // Resource status endpoint
			cluster.InternalMachineDeploymentStatus,
			MachineDeploymentSchemeFns{},
			func() runtime.Object { return &MachineDeployment{} },     // Register versioned resource
			func() runtime.Object { return &MachineDeploymentList{} }, // Register versioned resource list
			&MachineDeploymentValidationStatusStrategy{builders.StatusStorageStrategySingleton},
		), clusterMachineSetStorage,
		builders.NewApiResource( // Resource status endpoint
			cluster.InternalMachineSetStatus,
			MachineSetSchemeFns{},
			func() runtime.Object { return &MachineSet{} },     // Register versioned resource
			func() runtime.Object { return &MachineSetList{} }, // Register versioned resource list
			&MachineSetStatusStrategy{builders.StatusStorageStrategySingleton},
		))

	// Required by code generated by go2idl
	AddToScheme        = ApiVersion.SchemaBuilder.AddToScheme
	SchemeBuilder      = ApiVersion.SchemaBuilder
	localSchemeBuilder = &SchemeBuilder
	SchemeGroupVersion = ApiVersion.GroupVersion
)

// Required by code generated by go2idl
// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Required by code generated by go2idl
// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

//
// Cluster Functions and Structs
//
// +k8s:deepcopy-gen=false
type ClusterSchemeFns struct {
	builders.DefaultSchemeFns
}

// +k8s:deepcopy-gen=false
type ClusterStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type ClusterStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

//
// Machine Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineSchemeFns struct {
	builders.DefaultSchemeFns
}

// +k8s:deepcopy-gen=false
type MachineStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

//
// MachineDeployment Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineDeploymentSchemeFns struct {
	builders.DefaultSchemeFns
}

// +k8s:deepcopy-gen=false
type MachineDeploymentValidationStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineDeploymentValidationStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineDeployment `json:"items"`
}

//
// MachineSet Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineSetSchemeFns struct {
	builders.DefaultSchemeFns
}

// +k8s:deepcopy-gen=false
type MachineSetStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineSetStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineSet `json:"items"`
}
