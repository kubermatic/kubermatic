package version

import (
	"bufio"
	"io/ioutil"
	"os"
	"errors"

	yaml "gopkg.in/yaml.v2"

	"github.com/kubermatic/api"
)

func LoadVersions(path string) (map[string]*api.MasterVersion, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	vers := []api.MasterVersion{}
	err = yaml.Unmarshal(bytes, &vers)
	if err != nil {
		return nil, err
	}

	verMap := make(map[string]*api.MasterVersion)

	for _, ver := range vers {
		verMap[ver.ID] = &ver
	}

	return verMap, nil
}

func DefaultMasterVersion(versions map[string]*api.MasterVersion) (*api.MasterVersion, error) {
	for _, ver := range versions {
		if ver.Default {
			return ver, nil
		}
	}

	return nil, errors.New("latest version not found")
}
