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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
)

var (
	errVersionNotFound  = errors.New("version not found")
	errNoDefaultVersion = errors.New("no default version configured")
)

// Manager is a object to handle versions & updates from a predefined config.
type Manager struct {
	versions                  []*Version
	updates                   []*Update
	providerIncompatibilities []*ProviderIncompatibility
}

type ProviderIncompatibility struct {
	Provider  kubermaticv1.ProviderType  `json:"provider"`
	Version   string                     `json:"version"`
	Condition kubermaticv1.ConditionType `json:"condition"`
	Operation kubermaticv1.OperationType `json:"operation"`
}

// Version is the object representing a Kubernetes version.
type Version struct {
	Version *semverlib.Version `json:"version"`
	Default bool               `json:"default,omitempty"`
}

// Update represents an update option for a cluster.
type Update struct {
	From                string `json:"from"`
	To                  string `json:"to"`
	Automatic           bool   `json:"automatic,omitempty"`
	AutomaticNodeUpdate bool   `json:"automaticNodeUpdate,omitempty"`
}

// New returns a instance of Manager.
func New(versions []*Version, updates []*Update, providerIncompatibilities []*ProviderIncompatibility) *Manager {
	return &Manager{
		updates:                   updates,
		versions:                  versions,
		providerIncompatibilities: providerIncompatibilities,
	}
}

// NewFromConfiguration converts the configured versions/updates into the datatypes used by
// this package and returns a version.Manager on success.
func NewFromConfiguration(config *kubermaticv1.KubermaticConfiguration) *Manager {
	updates := []*Update{}
	versions := []*Version{}
	incompatibilities := []*ProviderIncompatibility{}

	k8s := config.Spec.Versions

	for _, k8sVersion := range k8s.Versions {
		versions = append(versions, &Version{
			Version: k8sVersion.Semver(),
			Default: k8s.Default != nil && k8sVersion.Equal(k8s.Default),
		})
	}

	for _, u := range k8s.Updates {
		updates = append(updates, &Update{
			From:                u.From,
			To:                  u.To,
			Automatic:           u.Automatic != nil && *u.Automatic,
			AutomaticNodeUpdate: u.AutomaticNodeUpdate != nil && *u.AutomaticNodeUpdate,
		})
	}

	for _, incomp := range k8s.ProviderIncompatibilities {
		incompatibilities = append(incompatibilities, &ProviderIncompatibility{
			Provider:  kubermaticv1.ProviderType(incomp.Provider),
			Version:   incomp.Version,
			Condition: incomp.Condition,
			Operation: incomp.Operation,
		})
	}

	return New(versions, updates, incompatibilities)
}

// GetDefault returns the default version.
func (m *Manager) GetDefault() (*Version, error) {
	for _, v := range m.versions {
		if v.Default {
			return v, nil
		}
	}
	return nil, errNoDefaultVersion
}

// GetVersion returns the Versions for s.
func (m *Manager) GetVersion(s string) (*Version, error) {
	sv, err := semverlib.NewVersion(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %w", s, err)
	}

	for _, v := range m.versions {
		if v.Version.Equal(sv) {
			return v, nil
		}
	}

	return nil, errVersionNotFound
}

// GetVersions returns all Versions which don't result in automatic updates, regardless of the cloud provider.
func (m *Manager) GetVersions() ([]*Version, error) {
	var masterVersions []*Version
	for _, v := range m.versions {
		autoUpdate, err := m.AutomaticControlplaneUpdate(v.Version.String())
		if err != nil {
			kubermaticlog.Logger.Errorf("Failed to get AutomaticUpdate for version %s: %v", v.Version.String(), err)
			continue
		}
		if autoUpdate != nil {
			continue
		}
		masterVersions = append(masterVersions, v)
	}

	return masterVersions, nil
}

// GetVersionsForProvider returns all Versions which don't result in automatic updates.
func (m *Manager) GetVersionsForProvider(provider kubermaticv1.ProviderType, conditions ...kubermaticv1.ConditionType) ([]*Version, error) {
	versions, err := m.GetVersions()
	if err != nil {
		return nil, err
	}

	filtered := []*Version{}
	for _, v := range versions {
		compatible, err := checkProviderCompatibility(v.Version, provider, kubermaticv1.CreateOperation, m.providerIncompatibilities, conditions...)
		if err != nil {
			return nil, err
		}
		if compatible {
			filtered = append(filtered, &Version{
				Version: v.Version,
				Default: v.Default,
			})
		}
	}

	return filtered, nil
}

// AutomaticNodeUpdate returns an automatic node update or nil.
func (m *Manager) AutomaticNodeUpdate(fromVersionRaw, controlPlaneVersion string) (*Version, error) {
	version, err := m.automaticUpdate(fromVersionRaw, true)
	if err != nil || version == nil {
		return version, err
	}
	controlPlaneSemver, err := semverlib.NewVersion(controlPlaneVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse controlplane version: %w", err)
	}

	if err := nodeupdate.EnsureVersionCompatible(controlPlaneSemver, version.Version); err != nil {
		return nil, err
	}

	return version, nil
}

// AutomaticControlplaneUpdate returns a version if an automatic update can be found for the version
// passed in.
func (m *Manager) AutomaticControlplaneUpdate(fromVersionRaw string) (*Version, error) {
	return m.automaticUpdate(fromVersionRaw, false)
}

func (m *Manager) automaticUpdate(fromVersionRaw string, isForNode bool) (*Version, error) {
	from, err := semverlib.NewVersion(fromVersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %w", fromVersionRaw, err)
	}

	isAutomatic := func(u *Update) bool {
		if isForNode {
			return u.AutomaticNodeUpdate
		}
		return u.Automatic || u.AutomaticNodeUpdate
	}

	var toVersions []string
	for _, u := range m.updates {
		if !isAutomatic(u) {
			continue
		}

		uFrom, err := semverlib.NewConstraint(u.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from constraint %s: %w", u.From, err)
		}
		if !uFrom.Check(from) {
			continue
		}

		// Automatic updates must not be a constraint. They must be version.
		if _, err = semverlib.NewVersion(u.To); err != nil {
			return nil, fmt.Errorf("failed to parse to version %s: %w", u.To, err)
		}
		toVersions = append(toVersions, u.To)
	}

	if len(toVersions) == 0 {
		return nil, nil
	}

	if len(toVersions) > 1 {
		return nil, fmt.Errorf("more than one automatic update found for version. Not allowed. Automatic updates to: %v", toVersions)
	}

	version, err := m.GetVersion(toVersions[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get Version for %s: %w", toVersions[0], err)
	}
	return version, nil
}

// GetPossibleUpdates returns possible updates for the version passed in.
func (m *Manager) GetPossibleUpdates(fromVersionRaw string, provider kubermaticv1.ProviderType, conditions ...kubermaticv1.ConditionType) ([]*Version, error) {
	from, err := semverlib.NewVersion(fromVersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %w", fromVersionRaw, err)
	}
	var possibleVersions []*Version

	var toConstraints []*semverlib.Constraints
	for _, u := range m.updates {
		uFrom, err := semverlib.NewConstraint(u.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from constraint %s: %w", u.From, err)
		}
		if !uFrom.Check(from) {
			continue
		}

		uTo, err := semverlib.NewConstraint(u.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to constraint %s: %w", u.To, err)
		}
		toConstraints = append(toConstraints, uTo)
	}

	for _, c := range toConstraints {
		for _, v := range m.versions {
			if c.Check(v.Version) && !from.Equal(v.Version) {
				compatible, err := checkProviderCompatibility(v.Version, provider, kubermaticv1.UpdateOperation, m.providerIncompatibilities, conditions...)
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

func (m *Manager) GetIncompatibilities() []*ProviderIncompatibility {
	return m.providerIncompatibilities
}

func (m *Manager) GetKubeOnePossibleUpdates(fromVersionRaw string, provider kubermaticv1.ProviderType, conditions ...kubermaticv1.ConditionType) ([]*Version, error) {
	from, err := semverlib.NewVersion(fromVersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %w", fromVersionRaw, err)
	}
	var possibleVersions []*Version

	var toConstraints []*semverlib.Constraints
	for _, u := range m.updates {
		uFrom, err := semverlib.NewConstraint(u.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from constraint %s: %w", u.From, err)
		}
		if !uFrom.Check(from) {
			continue
		}

		uTo, err := semverlib.NewConstraint(u.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to constraint %s: %w", u.To, err)
		}
		toConstraints = append(toConstraints, uTo)
	}

	for _, c := range toConstraints {
		for _, v := range m.versions {
			if c.Check(v.Version) && !from.Equal(v.Version) && from.LessThan(v.Version) {
				possibleVersions = append(possibleVersions, v)
			}
		}
	}

	return possibleVersions, nil
}
