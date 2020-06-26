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

package controller

import (
	"fmt"
	"time"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

const (
	UserClusterAPIServerServiceName         = "apiserver-external"
	UserClusterAPIServerServiceSuffixLength = 6

	// Amount of time to wait until at least one pod is running
	DefaultPodPortForwardWaitTimeout = 60 * time.Second
)

func GenerateName(base, buildID string) string {
	return fmt.Sprintf("%s-%s-%s", base, buildID, utilrand.String(UserClusterAPIServerServiceSuffixLength))
}
