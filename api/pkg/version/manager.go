package version

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"
)

var (
	errVersionNotFound  = errors.New("version not found")
	errNoDefaultVersion = errors.New("no default version configured")
)

// Manager is a object to handle versions & updates from a predefined config
type Manager struct {
	versions []*MasterVersion
	updates  []*MasterUpdate
}

// MasterVersion is the object representing a Kubernetes Master version.
type MasterVersion struct {
	Version             *semver.Version `json:"version"`
	Default             bool            `json:"default"`
	AllowedNodeVersions []string        `json:"allowedNodeVersions"`
}

// MasterUpdate represents an update option for K8s master components
type MasterUpdate struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Automatic bool   `json:"automatic"`
}

// New returns a instance of Manager
func New(versions []*MasterVersion, updates []*MasterUpdate) *Manager {
	return &Manager{
		updates:  updates,
		versions: versions,
	}
}

// NewFromFiles returns a instance of manager with the versions & updates loaded from the given paths
func NewFromFiles(versionsFilename, updatesFilename string) (*Manager, error) {
	updates, err := LoadUpdates(updatesFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to load updates from %s: %v", updatesFilename, err)
	}
	versions, err := LoadVersions(versionsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to load versions from %s: %v", versionsFilename, err)
	}

	return New(versions, updates), nil
}

// GetDefault returns the default version
func (m *Manager) GetDefault() (*MasterVersion, error) {
	for _, v := range m.versions {
		if v.Default {
			return v, nil
		}
	}
	return nil, errNoDefaultVersion
}

// GetVersion returns the MasterVersions for s
func (m *Manager) GetVersion(s string) (*MasterVersion, error) {
	sv, err := semver.NewVersion(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", s, err)
	}

	for _, v := range m.versions {
		if v.Version.Equal(sv) {
			return v, nil
		}
	}
	return nil, errVersionNotFound
}

// GetMasterVersions returns all MasterVersions which don't result in automatic updates
func (m *Manager) GetMasterVersions() ([]*MasterVersion, error) {
	var masterVersions []*MasterVersion
	for _, v := range m.versions {
		autoUpdate, err := m.AutomaticUpdate(v.Version.String())
		if err != nil {
			glog.V(0).Infof("failed to get AutomaticUpdate for version %s: %v", v.Version.String(), err)
			continue
		}
		if autoUpdate != nil {
			continue
		}
		masterVersions = append(masterVersions, v)
	}
	return masterVersions, nil
}

// AutomaticUpdate returns a version if an automatic update can be found for version sfrom
func (m *Manager) AutomaticUpdate(sfrom string) (*MasterVersion, error) {
	from, err := semver.NewVersion(sfrom)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", sfrom, err)
	}

	var toVersions []string
	for _, u := range m.updates {
		if !u.Automatic {
			continue
		}

		uFrom, err := semver.NewConstraint(u.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from constraint %s: %v", u.From, err)
		}
		if !uFrom.Check(from) {
			continue
		}

		// Automatic updates must not be a constraint. They must be version.
		if _, err = semver.NewVersion(u.To); err != nil {
			return nil, fmt.Errorf("failed to parse to version %s: %v", u.To, err)
		}
		toVersions = append(toVersions, u.To)
	}

	if len(toVersions) == 0 {
		return nil, nil
	}

	if len(toVersions) > 1 {
		return nil, fmt.Errorf("more than one automatic update found for version. Not allowed. Automatic updates to: %v", toVersions)
	}

	mVersion, err := m.GetVersion(toVersions[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get MasterVersion for %s: %v", toVersions[0], err)
	}
	return mVersion, nil
}

// GetPossibleUpdates returns possible updates for the version sfrom
func (m *Manager) GetPossibleUpdates(sfrom string) ([]*MasterVersion, error) {
	from, err := semver.NewVersion(sfrom)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", sfrom, err)
	}

	var toConstraints []*semver.Constraints
	for _, u := range m.updates {
		uFrom, err := semver.NewConstraint(u.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from constraint %s: %v", u.From, err)
		}
		if !uFrom.Check(from) {
			continue
		}

		uTo, err := semver.NewConstraint(u.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to constraint %s: %v", u.To, err)
		}
		toConstraints = append(toConstraints, uTo)
	}

	var possibleVersions []*MasterVersion
	for _, c := range toConstraints {
		for _, v := range m.versions {
			if c.Check(v.Version) && !from.Equal(v.Version) {
				possibleVersions = append(possibleVersions, v)
			}
		}
	}

	return possibleVersions, nil
}
