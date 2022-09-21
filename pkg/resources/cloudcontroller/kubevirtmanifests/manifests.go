/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirtmanifests

import (
	"embed"
	"path"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed preset
	standardPresetsFs    embed.FS
	standardPresetFolder = "preset"

	//go:embed instancetype
	standardInstancetypeFs     embed.FS
	standardInstancetypeFolder = "instancetype"

	//go:embed preference
	standardPreferenceFs     embed.FS
	standardPreferenceFolder = "preference"
)

type ManifestFS struct {
	Fs     embed.FS
	Folder string
}

type ManifestFSGetter interface {
	GetManifestFS() *ManifestFS
}

type StandardPresetGetter struct{}

func (s *StandardPresetGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:     standardPresetsFs,
		Folder: standardPresetFolder,
	}
}

type StandardInstancetypeGetter struct{}

func (s *StandardInstancetypeGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:     standardInstancetypeFs,
		Folder: standardInstancetypeFolder,
	}
}

type StandardPreferenceGetter struct{}

func (s *StandardPreferenceGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:     standardPreferenceFs,
		Folder: standardPreferenceFolder,
	}
}

// KubernetesFromYaml returns a list of Kubernetes runtime objects from their yaml templates.
// It returns the objects for all files included in the ManifestFS folder, skipping (with error log) the yaml files
// that would not correct yaml files.
func KubernetesFromYaml(client ctrlruntimeclient.Client, manifestFsGetter ManifestFSGetter) []runtime.Object {
	decode := serializer.NewCodecFactory(client.Scheme()).UniversalDeserializer().Decode

	manifestFs := manifestFsGetter.GetManifestFS()
	files, _ := manifestFs.Fs.ReadDir(manifestFs.Folder)
	objects := make([]runtime.Object, 0, len(files))

	for _, f := range files {
		manifest, err := manifestFs.Fs.ReadFile(path.Join(manifestFs.Folder, f.Name()))
		if err != nil {
			kubermaticlog.Logger.Errorf("Could not read the content of the manifest file %v [skipping it] - %v", path.Join(manifestFs.Folder, f.Name()), err)
			continue
		}
		obj, _, err := decode(manifest, nil, nil)
		// Skip and log but continue with others
		if err != nil {
			kubermaticlog.Logger.Errorf("Skipping manifest %v as an error occurred reading it [skipping it]- %v", path.Join(manifestFs.Folder, f.Name()), err)
			continue
		}
		objects = append(objects, obj)
	}

	return objects
}
