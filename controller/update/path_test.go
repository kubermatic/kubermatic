package cluster

import (
	"testing"

	"github.com/kubermatic/api"
)

func TestPathSearch_Search(t *testing.T) {
	search := NewPathSearch(
		[]*api.MasterVersion{
			{
				ID: "1.5.1",
			},
			{
				ID: "1.5.2",
			},
			{
				ID: "1.5.3",
			},
			{
				ID: "1.5.4",
			},
		}, []*api.MasterUpdate{
			{
				From: "1.5.1",
				To:   "1.5.2",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "1.5.*",
				To:   "1.5.4",
			},
		},
	)
}
