package manager

import (
	"github.com/kubermatic/api"
)

// AddonManager represents a addon manager
type AddonManager interface {
	ListReleases() error
	Install(addon *api.ClusterAddon) (*api.ClusterAddon, error)
	Delete(addon *api.ClusterAddon) error
	Update(addon *api.ClusterAddon) error
	Rollback(addon *api.ClusterAddon) error
}
