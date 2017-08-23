package handler

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/blang/semver"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/kubernetes"
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
		kps      map[string]provider.KubernetesProvider
		versions map[string]*api.MasterVersion
		updates  []api.MasterUpdate
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
				versions: generateMasterVersions([]string{"0.0.1", "1.6.0", "1.7.0"}),

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
						name: "base request",
						ctx:  context.Background(),
						req:  generateUpgradeReq("1.6.0", "234jkh24234g", "base", "anom"),
						want: want{
							val: nil,
							err: NewUnknownUpgradePath("0.0.1", "1.6.0"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := performClusterUpgrade(tt.args.kps, tt.args.versions, tt.args.updates)
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

func generateClusterReq(cluster, dc, user string) clusterReq {
	return clusterReq{
		dcReq: dcReq{
			dc: dc,
			userReq: userReq{
				user: provider.User{
					Name: user,
				},
			},
		},
		cluster: cluster,
	}
}

func generateUpgradeReq(to, cluster, dc, user string) upgradeReq {
	return upgradeReq{
		to:         to,
		clusterReq: generateClusterReq(cluster, dc, user),
	}
}

type update struct {
	from string
	to   string
}

func generateMasterUpdates(updates []update) []api.MasterUpdate {
	us := make([]api.MasterUpdate, len(updates))

	for i, u := range updates {
		us[i] = api.MasterUpdate{
			From:    u.from,
			To:      u.to,
			Enabled: true,
			Visible: true,
			Promote: true,
		}
	}

	return us
}

func generateMasterVersions(versions []string) map[string]*api.MasterVersion {
	vs := make(map[string]*api.MasterVersion)

	for _, v := range versions {
		vs[v] = &api.MasterVersion{
			Name: v,
			ID:   v,
		}
	}

	return vs
}

func generateSemVerSlice(versions []string) semver.Versions {
	vs := make(semver.Versions, 0)

	for _, v := range versions {
		ver, err := semver.Parse(v)
		if err != nil {
			continue
		}
		vs = append(vs, ver)
	}

	sort.Sort(vs)
	return vs
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
		updates  []api.MasterUpdate
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
							err: NewWrongRequest(nil, clusterReq{}),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewWrongRequest("blah", clusterReq{}),
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
							err: NewWrongRequest(nil, clusterReq{}),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewWrongRequest("blah", clusterReq{}),
						},
					},
					{
						name: "base request - empty response",
						ctx:  context.Background(),
						req:  generateClusterReq("234jkh24234g", "base", "anom"),
						want: want{
							val: semver.Versions{},
							err: nil,
						},
					},
				},
			},
		},
		{
			name: "upgradable",
			args: args{
				kps:      generateBaseKubernetesProvider(),
				versions: generateMasterVersions([]string{"0.0.1", "1.6.0", "1.7.0"}),
				updates: generateMasterUpdates([]update{
					update{from: "0.0.1", to: "1.6.0"},
					update{from: "1.6.0", to: "1.7.0"},
				}),
				args: []endpointArgs{
					{
						name: "no request",
						ctx:  context.Background(),
						req:  nil,
						want: want{
							val: nil,
							err: NewWrongRequest(nil, clusterReq{}),
						},
					},
					{
						name: "wrong request",
						ctx:  context.Background(),
						req:  "blah",
						want: want{
							val: nil,
							err: NewWrongRequest("blah", clusterReq{}),
						},
					},
					{
						name: "base request - empty response",
						ctx:  context.Background(),
						req:  generateClusterReq("234jkh24234g", "base", "anom"),
						want: want{
							val: generateSemVerSlice([]string{"1.6.0", "1.7.0"}),
							err: nil,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := getClusterUpgrades(tt.args.kps, tt.args.versions, tt.args.updates)
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
