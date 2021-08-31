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
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
)

type IncompatibilityCondition string

const (
	AlwaysCondition                IncompatibilityCondition = "always"
	ExternalCloudProviderCondition IncompatibilityCondition = "externalCloudProvider"
)

type OperationType string

const (
	CreateOperation OperationType = "CREATE"
	UpdateOperation OperationType = "UPDATE"
)

var (
	errVersionNotFound  = errors.New("version not found")
	errNoDefaultVersion = errors.New("no default version configured")
)

// Manager is a object to handle versions & updates from a predefined config
type Manager struct {
	versions                  []*Version
	updates                   []*Update
	providerIncompatibilities []*ProviderIncompatibility
}

type ProviderIncompatibility struct {
	Provider  kubermaticv1.ProviderType `json:"provider"`
	Version   string                    `json:"version"`
	Condition IncompatibilityCondition  `json:"condition"`
	Operation OperationType             `json:"operation"`
	Type      string                    `json:"type,omitempty"`
}

// Version is the object representing a Kubernetes version.
type Version struct {
	Version *semver.Version `json:"version"`
	Default bool            `json:"default,omitempty"`
	Type    string          `json:"type,omitempty"`
}

// Update represents an update option for a cluster
type Update struct {
	From                string `json:"from"`
	To                  string `json:"to"`
	Automatic           bool   `json:"automatic,omitempty"`
	AutomaticNodeUpdate bool   `json:"automaticNodeUpdate,omitempty"`
	Type                string `json:"type,omitempty"`
}

// New returns a instance of Manager
func New(versions []*Version, updates []*Update, providerIncompatibilities []*ProviderIncompatibility) *Manager {
	return &Manager{
		updates:                   updates,
		versions:                  versions,
		providerIncompatibilities: providerIncompatibilities,
	}
}

// NewFromFiles returns a instance of manager with the versions & updates loaded from the given paths
func NewFromFiles(versionsFilename, updatesFilename, providerIncompatibilitiesFilename string) (*Manager, error) {
	updates, err := LoadUpdates(updatesFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to load updates from %s: %v", updatesFilename, err)
	}
	for _, update := range updates {
		// set default type if empty
		if len(update.Type) == 0 {
			update.Type = v1.KubernetesClusterType
		}
	}

	versions, err := LoadVersions(versionsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to load versions from %s: %v", versionsFilename, err)
	}
	for _, version := range versions {
		if len(version.Type) == 0 {
			version.Type = v1.KubernetesClusterType
		}
	}

	providerIncompatibilities, err := LoadProviderIncompatibilities(providerIncompatibilitiesFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to load provider providerIncompatibilities from %s: %v", providerIncompatibilitiesFilename, err)
	}
	for _, i := range providerIncompatibilities {
		if len(i.Type) == 0 {
			i.Type = v1.KubernetesClusterType
		}
	}

	return New(versions, updates, providerIncompatibilities), nil
}

// GetDefault returns the default version
func (m *Manager) GetDefault() (*Version, error) {
	for _, v := range m.versions {
		if v.Default {
			return v, nil
		}
	}
	return nil, errNoDefaultVersion
}

// GetVersion returns the Versions for s
func (m *Manager) GetVersion(s, t string) (*Version, error) {
	sv, err := semver.NewVersion(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", s, err)
	}

	for _, v := range m.versions {
		if v.Version.Equal(sv) && v.Type == t {
			return v, nil
		}
	}
	return nil, errVersionNotFound
}

// GetVersions returns all Versions which don't result in automatic updates
func (m *Manager) GetVersions(clusterType string) ([]*Version, error) {
	var masterVersions []*Version
	for _, v := range m.versions {
		if v.Type == clusterType {
			autoUpdate, err := m.AutomaticControlplaneUpdate(v.Version.String(), clusterType)
			if err != nil {
				kubermaticlog.Logger.Errorf("Failed to get AutomaticUpdate for version %s: %v", v.Version.String(), err)
				continue
			}
			if autoUpdate != nil {
				continue
			}
			masterVersions = append(masterVersions, v)
		}
	}
	return masterVersions, nil
}

// GetVersionsV2 returns all Versions which don't result in automatic updates
func (m *Manager) GetVersionsV2(clusterType string, provider kubermaticv1.ProviderType, conditions ...IncompatibilityCondition) ([]*Version, error) {
	var masterVersions []*Version
	for _, v := range m.versions {
		if v.Type == clusterType {
			autoUpdate, err := m.AutomaticControlplaneUpdate(v.Version.String(), clusterType)
			if err != nil {
				kubermaticlog.Logger.Errorf("Failed to get AutomaticUpdate for version %s: %v", v.Version.String(), err)
				continue
			}
			if autoUpdate != nil {
				continue
			}
			compatible, err := m.checkProviderCompatibility(v.Version, provider, clusterType, CreateOperation, conditions...)
			if err != nil {
				return nil, err
			}
			if compatible {
				masterVersions = append(masterVersions, v)
			}
		}
	}
	return masterVersions, nil
}

// AutomaticNodeUpdate returns an automatic node update or nil
func (m *Manager) AutomaticNodeUpdate(fromVersionRaw, clusterType, controlPlaneVersion string) (*Version, error) {
	version, err := m.automaticUpdate(fromVersionRaw, clusterType, true)
	if err != nil || version == nil {
		return version, err
	}
	controlPlaneSemver, err := semver.NewVersion(controlPlaneVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse controlplane version: %v", err)
	}

	if err := nodeupdate.EnsureVersionCompatible(controlPlaneSemver, version.Version); err != nil {
		return nil, err
	}

	return version, nil
}

// AutomaticControlplaneUpdate returns a version if an automatic update can be found for the version
// passed in
func (m *Manager) AutomaticControlplaneUpdate(fromVersionRaw, clusterType string) (*Version, error) {
	return m.automaticUpdate(fromVersionRaw, clusterType, false)
}

func (m *Manager) automaticUpdate(fromVersionRaw, clusterType string, isForNode bool) (*Version, error) {
	from, err := semver.NewVersion(fromVersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", fromVersionRaw, err)
	}

	isAutomatic := func(u *Update) bool {
		if isForNode {
			return u.AutomaticNodeUpdate
		}
		return u.Automatic
	}

	var toVersions []string
	for _, u := range m.updates {
		if u.Type != clusterType {
			continue
		}

		if !isAutomatic(u) {
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

	version, err := m.GetVersion(toVersions[0], clusterType)
	if err != nil {
		return nil, fmt.Errorf("failed to get Version for %s: %v", toVersions[0], err)
	}
	return version, nil
}

// GetPossibleUpdates returns possible updates for the version passed in
func (m *Manager) GetPossibleUpdates(fromVersionRaw, clusterType string, provider kubermaticv1.ProviderType, conditions ...IncompatibilityCondition) ([]*Version, error) {
	from, err := semver.NewVersion(fromVersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", fromVersionRaw, err)
	}
	var possibleVersions []*Version

	var toConstraints []*semver.Constraints
	for _, u := range m.updates {
		if u.Type == clusterType {
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
	}

	for _, c := range toConstraints {
		for _, v := range m.versions {
			if c.Check(v.Version) && !from.Equal(v.Version) && v.Type == clusterType {
				compatible, err := m.checkProviderCompatibility(v.Version, provider, clusterType, UpdateOperation, conditions...)
				if err != nil {
					return nil, err
				}
				if compatible {
					possibleVersions = append(possibleVersions, v)
				}
			}
		}
	}

	return possibleVersions, nil
}

func (m *Manager) checkProviderCompatibility(version *semver.Version, provider kubermaticv1.ProviderType, clusterType string, operation OperationType, conditions ...IncompatibilityCondition) (bool, error) {
	var compatible = true
	var err error
	for _, pi := range m.providerIncompatibilities {
		if pi.Provider == provider && pi.Type == clusterType && operation == pi.Operation {
			if pi.Condition == AlwaysCondition {
				compatible, err = createAndCheckConstraint(version, pi.Version)
				if err != nil {
					return false, fmt.Errorf("check incompatibility failed")
				}
			} else {
				for _, ic := range conditions {
					if pi.Condition == ic || ic == AlwaysCondition || pi.Condition == AlwaysCondition {
						compatible, err = createAndCheckConstraint(version, pi.Version)
						if err != nil {
							return false, fmt.Errorf("check incompatibility failed")
						}
						if !compatible {
							return false, nil
						}
					}
				}
			}
			if !compatible {
				return false, nil
			}
		}
	}
	return compatible, nil
}

func createAndCheckConstraint(baseVersion *semver.Version, version string) (bool, error) {
	c, err := semver.NewConstraint(version)
	if err != nil {
		return false, fmt.Errorf("failed to parse to constraint %s: %v", c, err)
	}

	return !c.Check(baseVersion), nil
}
