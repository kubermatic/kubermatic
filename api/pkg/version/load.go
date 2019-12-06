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
