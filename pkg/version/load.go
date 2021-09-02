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

package version

import (
	"bufio"
	"io/ioutil"
	"os"

	"sigs.k8s.io/yaml"
)

// LoadUpdates loads the update definition file and returns the defined MasterUpdate
func LoadUpdates(path string) ([]*Update, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Updates []*Update `json:"updates"`
	}{}

	err = yaml.UnmarshalStrict(bytes, &s)
	if err != nil {
		return nil, err
	}
	for _, update := range s.Updates {
		// AutomaticNodeUpdate implies automatic update, because nodes
		// must not have a newer version than the control plane
		if update.AutomaticNodeUpdate {
			update.Automatic = true
		}
	}

	return s.Updates, nil
}

// LoadVersions loads Versions from a given path
func LoadVersions(path string) ([]*Version, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Versions []*Version `json:"versions"`
	}{}

	err = yaml.UnmarshalStrict(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Versions, nil
}

func LoadProviderIncompatibilities(path string) ([]*ProviderIncompatibility, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		ProviderIncompatibilities []*ProviderIncompatibility `json:"providerIncompatibilities"`
	}{}

	err = yaml.UnmarshalStrict(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.ProviderIncompatibilities, nil
}
