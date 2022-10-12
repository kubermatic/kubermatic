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

package middleware

import (
	kubermaticcontext "k8c.io/kubermatic/v2/pkg/util/context"
)

const (
	// RawTokenContextKey key under which the current token (OpenID ID Token) is kept in the ctx.
	RawTokenContextKey kubermaticcontext.Key = "raw-auth-token"

	// ClusterProviderContextKey key under which the current ClusterProvider is kept in the ctx.
	ClusterProviderContextKey kubermaticcontext.Key = "cluster-provider"

	// PrivilegedClusterProviderContextKey key under which the current PrivilegedClusterProvider is kept in the ctx.
	PrivilegedClusterProviderContextKey kubermaticcontext.Key = "privileged-cluster-provider"

	UserCRContextKey = kubermaticcontext.UserCRContextKey
)
