package version

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/kubermatic/api"
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

	updates := []api.MasterUpdate{}
	err = yaml.Unmarshal(bytes, &updates)
	if err != nil {
		return nil, err
	}

	return updates, nil
}
