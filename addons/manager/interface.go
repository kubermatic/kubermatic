package manager

import (
	"github.com/kubermatic/api/extensions"
)

// AddonManager represents a addon manager
type AddonManager interface {
	ListReleases() error
	Install(addon *extensions.ClusterAddon) (*extensions.ClusterAddon, error)
	Delete(addon *extensions.ClusterAddon) error
	Update(addon *extensions.ClusterAddon) error
	Rollback(addon *extensions.ClusterAddon) error
}
