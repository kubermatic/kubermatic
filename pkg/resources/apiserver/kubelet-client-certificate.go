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

package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

type kubeletClientCertificateCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
}

// KubeletClientCertificateCreator returns a function to create/update a secret with the client certificate for the apiserver -> kubelet connection.
func KubeletClientCertificateCreator(data kubeletClientCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	return certificates.GetClientCertificateCreator(
		resources.KubeletClientCertificatesSecretName,
		"kube-apiserver-kubelet-client",
		[]string{"system:masters"},
		resources.KubeletClientCertSecretKey,
		resources.KubeletClientKeySecretKey,
		data.GetRootCA,
	)
}
