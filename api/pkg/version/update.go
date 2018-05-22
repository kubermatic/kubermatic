package version

import (
	"bufio"
	"io/ioutil"
	"os"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	yaml "gopkg.in/yaml.v2"
)

// LoadUpdates loads the update definition file and returns the defined MasterUpdate
func LoadUpdates(path string) ([]apiv1.MasterUpdate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Updates []apiv1.MasterUpdate
	}{
		Updates: []apiv1.MasterUpdate{},
	}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Updates, nil
}
