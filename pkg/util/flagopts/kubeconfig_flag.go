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

package flagopts

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

// NewKubeconfig detect KUBECONFIG from ENV or default to $HOME/.kube/config
func NewKubeconfig() KubeconfigFlag {
	defaultKubeconfig, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		defaultKubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	return KubeconfigFlag(defaultKubeconfig)
}

// KubeconfigFlag flag, will detect possible defaults
type KubeconfigFlag string

// String is flag.Value implementation method
func (s KubeconfigFlag) String() string {
	return string(s)
}

// Set is flag.Value implementation method
func (s *KubeconfigFlag) Set(val string) error {
	if s == nil {
		s = new(KubeconfigFlag)
	}
	*s = KubeconfigFlag(val)
	return nil
}
