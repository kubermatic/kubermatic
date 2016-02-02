package node

import (
	"github.com/kubermatic/api/provider"
)

var _ provider.Node = (*node)(nil)

type node struct {
	FakeID       string `json:"id"`
	FakePublicIP string `json:"public"`
}

func (n *node) ID() string {
	return n.FakeID
}

func (n *node) PublicIP() string {
	return n.FakePublicIP
}
