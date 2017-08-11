package handler

import (
	"context"
	"reflect"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/kubernetes"
)

func Test_performClusterUpgrade(t *testing.T) {
	type want struct {
		val interface{}
		err error
	}

	type endpointArgs struct {
		name string
		ctx  context.Context
		req  interface{}
		want want
	}

	type args struct {
		kps     map[string]provider.KubernetesProvider
		updates []api.MasterUpdate
		args    []endpointArgs
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "no config",
			args: args{
				kps:     nil,
				updates: nil,
				args: []endpointArgs{
					{
						name: "no request",
						ctx:  context.Background(),
						req:  nil,
						want: want{
							val: nil,
							err: NewNotImplemented(),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewNotImplemented(),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := performClusterUpgrade(tt.args.kps, tt.args.updates)
			for _, ttE := range tt.args.args {
				t.Run(ttE.name, func(t *testing.T) {
					got, err := fn(ttE.ctx, ttE.req)
					if ok := reflect.DeepEqual(got, ttE.want.val); !ok {
						t.Errorf("expected: %+v - got: %+v", ttE.want.val, got)
					}
					if ok := reflect.DeepEqual(err, ttE.want.err); !ok {
						t.Errorf("expected: %+v - got: %+v", ttE.want.err, err)
					}
				})
			}
		})
	}
}

func generateFakeDCs() map[string]provider.DatacenterMeta {
	return nil
}

func generateBaseKubernetesProvider() map[string]provider.KubernetesProvider {
	return map[string]provider.KubernetesProvider{
		"base": kubernetes.NewKubernetesFakeProvider(
			"base",
			nil,
			generateFakeDCs(),
		),
	}
}

func generateClusterUpgradeReq(cluster, dc, user string) upgradeReq {
	return upgradeReq{
		clusterReq: clusterReq{
			dcReq: dcReq{
				dc: dc,
				userReq: userReq{
					user: provider.User{
						Name: user,
					},
				},
			},
			cluster: cluster,
		},
	}
}

func Test_getClusterUpgrades(t *testing.T) {
	type want struct {
		val interface{}
		err error
	}

	type endpointArgs struct {
		name string
		ctx  context.Context
		req  interface{}
		want want
	}

	type args struct {
		kps      map[string]provider.KubernetesProvider
		versions map[string]*api.MasterVersion
		args     []endpointArgs
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "no config",
			args: args{
				kps:      nil,
				versions: nil,
				args: []endpointArgs{
					{
						name: "no request",
						ctx:  context.Background(),
						req:  nil,
						want: want{
							val: nil,
							err: NewWrongRequest(nil, upgradeReq{}),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewWrongRequest("blah", upgradeReq{}),
						},
					},
				},
			},
		},
		{
			name: "base config",
			args: args{
				kps:      generateBaseKubernetesProvider(),
				versions: nil,
				args: []endpointArgs{
					{
						name: "no request",
						ctx:  context.Background(),
						req:  nil,
						want: want{
							val: nil,
							err: NewWrongRequest(nil, upgradeReq{}),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewWrongRequest("blah", upgradeReq{}),
						},
					},
					{
						name: "base request - empty response",
						ctx:  context.Background(),
						req:  generateClusterUpgradeReq("234jkh24234g", "base", "anom"),
						want: want{
							val: []*version.Version{},
							err: nil,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := getClusterUpgrades(tt.args.kps, tt.args.versions)
			for _, ttE := range tt.args.args {
				t.Run(ttE.name, func(t *testing.T) {
					got, err := fn(ttE.ctx, ttE.req)
					if ok := reflect.DeepEqual(got, ttE.want.val); !ok {
						t.Errorf("expected: %+v - got: %+v", ttE.want.val, got)
					}
					if ok := reflect.DeepEqual(err, ttE.want.err); !ok {
						t.Errorf("expected: %+v - got: %+v", ttE.want.err, err)
					}
				})
			}
		})
	}
}
