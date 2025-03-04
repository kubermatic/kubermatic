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

package cli

import (
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

func Hello(log *zap.SugaredLogger, app string, version *kubermatic.Versions) {
	if version == nil {
		v := kubermatic.GetVersions()
		version = &v
	}

	log.
		With("version", version.GitVersion).
		With("edition", version.KubermaticEdition.ShortString()).
		Infof("Starting KKP %s (%s)...", app, version.KubermaticEdition)
}
