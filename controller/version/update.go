package version

import (
	"bufio"
	"io/ioutil"
	"os"

	"github.com/kubermatic/api"
	yaml "gopkg.in/yaml.v2"
)

func LoadUpdates(path string) ([]api.MasterUpdate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Updates []api.MasterUpdate
	}{
		Updates: []api.MasterUpdate{},
	}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Updates, nil
}
