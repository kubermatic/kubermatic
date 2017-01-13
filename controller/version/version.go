package version

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kubermatic/api"

	yaml "gopkg.in/yaml.v2"
)

//load yaml
func loadVersions(path string) (map[string]*api.MasterVersion, error) {

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

//upgrade path

//get latest version
func LatestVersion() (*api.MasterVersion, error) {
	vers, err := loadVersions("path")
	if err != nil {
		return nil, err
	}

	for _, ver := range vers {
		if ver.Latest {
			return ver, nil
		}
	}

	err = fmt.Errorf("Could not locate latest version")
	return nil, err

}
