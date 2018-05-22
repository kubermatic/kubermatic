package version

import (
	"fmt"

	"github.com/Masterminds/semver"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

type Manager struct {
	versions []*apiv1.MasterVersion
	updates  []*apiv1.MasterUpdate
}

func New(versions []*apiv1.MasterVersion, updates []*apiv1.MasterUpdate) *Manager {
	return &Manager{
		updates:  updates,
		versions: versions,
	}
}

func (m *Manager) AutomaticUpdate(from *semver.Version) (*apiv1.MasterVersion, error) {
	var toConstraints []*semver.Constraints
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

		uTo, err := semver.NewConstraint(u.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to constraint %s: %v", u.To, err)
		}
		toConstraints = append(toConstraints, uTo)
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
