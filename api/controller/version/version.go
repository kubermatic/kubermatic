package version

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	yaml "gopkg.in/yaml.v2"
)

// LoadVersions loads MasterVersions from a given path
func LoadVersions(path string) (map[string]*api.MasterVersion, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Versions []api.MasterVersion `yaml:"versions"`
	}{
		[]api.MasterVersion{},
	}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	verMap := make(map[string]*api.MasterVersion)

	for i, ver := range s.Versions {
		verMap[ver.ID] = &s.Versions[i]
	}

	return verMap, nil
}

// DefaultMasterVersion determines the current default MasterVersion from a set of versions
func DefaultMasterVersion(versions map[string]*api.MasterVersion) (*api.MasterVersion, error) {
	for _, ver := range versions {
		if ver.Default {
			return ver, nil
		}
	}

	return nil, errors.New("default version not found")
}

// BestAutomaticUpdate determines the best automatic update available from the current version
func BestAutomaticUpdate(from string, updates []api.MasterUpdate) (*api.MasterUpdate, error) {
	type ToUpdate struct {
		to     *semver.Version
		update *api.MasterUpdate
	}
	tos := []*ToUpdate{}
	semverFrom, err := semver.NewVersion(from)
	if err != nil {
		return nil, fmt.Errorf("failed to create semver version: %v", err)
	}
	for i, u := range updates {
		if !u.Automatic {
			continue
		}
		uFrom, err := semver.NewConstraint(u.From)
		if err != nil {
			glog.Warningf("ignoring update %q -> %q with invalid target version", u.From, u.To)
			continue
		}
		if !uFrom.Check(semverFrom) {
			continue
		}

		semverTo, err := semver.NewVersion(u.To)
		if err != nil {
			glog.Warningf("ignoring update %q -> %q with invalid source version", u.From, u.To)
			continue
		}
		if semverFrom.LessThan(semverTo) {
			tos = append(tos, &ToUpdate{
				to:     semverTo,
				update: &updates[i],
			})
		}
	}

	if len(tos) == 0 {
		return nil, nil
	}

	best := tos[0]
	for _, dest := range tos[1:] {
		if best.to.LessThan(dest.to) {
			best = dest
		}
	}

	return best.update, nil
}
