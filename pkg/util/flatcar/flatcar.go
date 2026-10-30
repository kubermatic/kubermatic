/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package flatcar

import kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

// Source names the level that decided the flatcar-linux-update-operator setting.
type Source string

const (
	SourceNone       Source = ""
	SourceCluster    Source = "cluster"
	SourceDatacenter Source = "datacenter"
	SourceSeed       Source = "seed"
	SourceGlobal     Source = "global"
)

// Resolution is the resolved setting for a single user cluster.
type Resolution struct {
	Disabled bool
	Source   Source
}

// Resolve walks cluster > datacenter > seed > global and returns the first declared value; an
// explicit false stops the walk.
func Resolve(clusterConf, dcConf, seedConf, globalConf *kubermaticv1.FlatcarUpdateOperatorSettings) Resolution {
	for _, level := range []struct {
		conf   *kubermaticv1.FlatcarUpdateOperatorSettings
		source Source
	}{
		{clusterConf, SourceCluster},
		{dcConf, SourceDatacenter},
		{seedConf, SourceSeed},
		{globalConf, SourceGlobal},
	} {
		if level.conf != nil && level.conf.Disabled != nil {
			return Resolution{Disabled: *level.conf.Disabled, Source: level.source}
		}
	}

	return Resolution{Disabled: false, Source: SourceNone}
}
