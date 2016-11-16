/*
Copyright 2016 The Kubernetes Authors.

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

package master

import (
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/apis/autoscaling"
	autoscalingapiv1 "k8s.io/kubernetes/pkg/apis/autoscaling/v1"
	"k8s.io/kubernetes/pkg/genericapiserver"
	horizontalpodautoscaleretcd "k8s.io/kubernetes/pkg/registry/horizontalpodautoscaler/etcd"
)

type AutoscalingRESTStorageProvider struct{}

var _ RESTStorageProvider = &AutoscalingRESTStorageProvider{}

func (p AutoscalingRESTStorageProvider) NewRESTStorage(apiResourceConfigSource genericapiserver.APIResourceConfigSource, restOptionsGetter RESTOptionsGetter) (genericapiserver.APIGroupInfo, bool) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(autoscaling.GroupName)

	if apiResourceConfigSource.AnyResourcesForVersionEnabled(autoscalingapiv1.SchemeGroupVersion) {
		apiGroupInfo.VersionedResourcesStorageMap[autoscalingapiv1.SchemeGroupVersion.Version] = p.v1Storage(apiResourceConfigSource, restOptionsGetter)
		apiGroupInfo.GroupMeta.GroupVersion = autoscalingapiv1.SchemeGroupVersion
	}

	return apiGroupInfo, true
}

func (p AutoscalingRESTStorageProvider) v1Storage(apiResourceConfigSource genericapiserver.APIResourceConfigSource, restOptionsGetter RESTOptionsGetter) map[string]rest.Storage {
	version := autoscalingapiv1.SchemeGroupVersion

	storage := map[string]rest.Storage{}
	if apiResourceConfigSource.ResourceEnabled(version.WithResource("horizontalpodautoscalers")) {
		hpaStorage, hpaStatusStorage := horizontalpodautoscaleretcd.NewREST(restOptionsGetter(autoscaling.Resource("horizontalpodautoscalers")))
		storage["horizontalpodautoscalers"] = hpaStorage
		storage["horizontalpodautoscalers/status"] = hpaStatusStorage
	}
	return storage
}
