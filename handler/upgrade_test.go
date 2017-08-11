package handler

import (
	"context"
	"reflect"
	"testing"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
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
