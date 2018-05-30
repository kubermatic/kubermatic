package version

import (
	"bufio"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

// LoadUpdates loads the update definition file and returns the defined MasterUpdate
func LoadUpdates(path string) ([]*MasterUpdate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Updates []*MasterUpdate `json:"updates"`
	}{}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Updates, nil
}

// LoadVersions loads MasterVersions from a given path
func LoadVersions(path string) ([]*MasterVersion, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Versions []*MasterVersion `json:"versions"`
	}{}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Versions, nil
}
