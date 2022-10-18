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

package manifests

import (
	"embed"
	"path"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed presets
	standardPresetsFs embed.FS
	standardPresetDir = "presets"

	//go:embed instancetypes
	standardInstancetypeFs  embed.FS
	standardInstancetypeDir = "instancetypes"

	//go:embed preferences
	standardPreferenceFs  embed.FS
	standardPreferenceDir = "preferences"
)

type ManifestFS struct {
	Fs  embed.FS
	Dir string
}

type ManifestFSGetter interface {
	GetManifestFS() *ManifestFS
}

type StandardPresetGetter struct{}

func (s *StandardPresetGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:  standardPresetsFs,
		Dir: standardPresetDir,
	}
}

type StandardInstancetypeGetter struct{}

func (s *StandardInstancetypeGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:  standardInstancetypeFs,
		Dir: standardInstancetypeDir,
	}
}

type StandardPreferenceGetter struct{}

func (s *StandardPreferenceGetter) GetManifestFS() *ManifestFS {
	return &ManifestFS{
		Fs:  standardPreferenceFs,
		Dir: standardPreferenceDir,
	}
}

// RuntimeFromYaml returns a list of Kubernetes runtime objects from their yaml templates.
// It returns the objects for all files included in the ManifestFS folder, skipping (with error log) the yaml files
// that would not contain correct yaml files.
func RuntimeFromYaml(client ctrlruntimeclient.Client, manifestFsGetter ManifestFSGetter) []runtime.Object {
	decode := serializer.NewCodecFactory(client.Scheme()).UniversalDeserializer().Decode

	manifestFs := manifestFsGetter.GetManifestFS()
	files, _ := manifestFs.Fs.ReadDir(manifestFs.Dir)
	objects := make([]runtime.Object, 0, len(files))

	for _, f := range files {
		manifest, err := manifestFs.Fs.ReadFile(path.Join(manifestFs.Dir, f.Name()))
		if err != nil {
			kubermaticlog.Logger.Errorf("Could not read the content of the manifest file %v [skipping it] - %v", path.Join(manifestFs.Dir, f.Name()), err)
			continue
		}
		obj, _, err := decode(manifest, nil, nil)
		// Skip and log but continue with others
		if err != nil {
			kubermaticlog.Logger.Errorf("Skipping manifest %v as an error occurred reading it [skipping it]- %v", path.Join(manifestFs.Dir, f.Name()), err)
			continue
		}
		objects = append(objects, obj)
	}

	return objects
}
