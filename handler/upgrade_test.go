package handler

import (
	"context"
	"reflect"
	"testing"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

func Test_performClusterUpgrade(t *testing.T) {
	type args struct {
		kps     map[string]provider.KubernetesProvider
		updates []api.MasterUpdate
	}
	type endpointArgs struct {
		name string
		ctx  context.Context
		req  interface{}
	}
	type want struct {
		val interface{}
		err error
	}
	tests := []struct {
		name         string
		args         args
		want         want
		endpointArgs []endpointArgs
	}{
		{
			name: "nop",
			args: args{
				kps:     nil,
				updates: nil,
			},
			endpointArgs: []endpointArgs{
				{
					name: "nop",
					ctx:  context.Background(),
					req:  nil,
				},
			},
			want: want{
				val: nil,
				err: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := performClusterUpgrade(tt.args.kps, tt.args.updates)
			for _, ttEndpoint := range tt.endpointArgs {
				t.Run(ttEndpoint.name, func(t *testing.T) {
					if got, err := fn(ttEndpoint.ctx, ttEndpoint.req); !reflect.DeepEqual(got, tt.want.val) && !reflect.DeepEqual(err, tt.want.err) {
						t.Errorf("performClusterUpgrade() = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}

func Test_getClusterUpgrades(t *testing.T) {
	type args struct {
		kps      map[string]provider.KubernetesProvider
		versions map[string]*api.MasterVersion
	}
	type endpointArgs struct {
		name string
		ctx  context.Context
		req  interface{}
	}
	type want struct {
		val interface{}
		err error
	}
	tests := []struct {
		name         string
		args         args
		want         want
		endpointArgs []endpointArgs
	}{
		{
			name: "nop",
			args: args{
				kps:      nil,
				versions: nil,
			},
			endpointArgs: []endpointArgs{
				{
					name: "nop",
					ctx:  context.Background(),
					req:  nil,
				},
			},
			want: want{
				val: nil,
				err: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := getClusterUpgrades(tt.args.kps, tt.args.versions)
			for _, ttEndpoint := range tt.endpointArgs {
				t.Run(ttEndpoint.name, func(t *testing.T) {
					if got, err := fn(ttEndpoint.ctx, ttEndpoint.req); !reflect.DeepEqual(got, tt.want.val) && !reflect.DeepEqual(err, tt.want.err) {
						t.Errorf("performClusterUpgrade() = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}
