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

package openshift

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

// GetAPIServicesForOpenshiftVersion returns all the NamedAPIServiceCreatorGetters for the given Openshift version
// or an error
func GetAPIServicesForOpenshiftVersion(openshiftVersion string, caBundle []byte) ([]reconciling.NamedAPIServiceCreatorGetter, error) {
	switch {
	case strings.HasPrefix(openshiftVersion, "4.1."):
		return []reconciling.NamedAPIServiceCreatorGetter{
			apiServiceFactory(caBundle, "v1.apps.openshift.io", "apps.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.authorization.openshift.io", "authorization.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.build.openshift.io", "build.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.image.openshift.io", "image.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.oauth.openshift.io", "oauth.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.project.openshift.io", "project.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.quota.openshift.io", "quota.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.route.openshift.io", "route.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.security.openshift.io", "security.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.template.openshift.io", "template.openshift.io", 9900, "v1", 15),
			apiServiceFactory(caBundle, "v1.user.openshift.io", "user.openshift.io", 9900, "v1", 15),
		}, nil
	default:
		return nil, fmt.Errorf("apiservices for openshift version %q are unknown", openshiftVersion)
	}
}

func apiServiceFactory(
	caBundle []byte,
	name string,
	group string,
	groupPriorityMinimum int32,
	version string,
	versionPriority int32) reconciling.NamedAPIServiceCreatorGetter {
	return func() (string, reconciling.APIServiceCreator) {
		return name, func(s *apiregistrationv1beta1.APIService) (*apiregistrationv1beta1.APIService, error) {

			if s.Spec.Service == nil {
				s.Spec.Service = &apiregistrationv1beta1.ServiceReference{}
			}
			s.Spec.Service.Namespace = "openshift-apiserver"
			s.Spec.Service.Name = "api"
			s.Spec.Group = group
			s.Spec.Version = version
			s.Spec.InsecureSkipTLSVerify = false
			s.Spec.CABundle = caBundle
			s.Spec.GroupPriorityMinimum = groupPriorityMinimum
			s.Spec.VersionPriority = versionPriority

			return s, nil
		}
	}
}
