package handler

import (
	"reflect"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/api/provider"
)

func Test_getClusterUpgrades(t *testing.T) {
	type args struct {
		kps map[string]provider.KubernetesProvider
	}
	tests := []struct {
		name string
		args args
		want endpoint.Endpoint
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getClusterUpgrades(tt.args.kps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getClusterUpgrades() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_performClusterUpgrade(t *testing.T) {
	type args struct {
		kps map[string]provider.KubernetesProvider
	}
	tests := []struct {
		name string
		args args
		want endpoint.Endpoint
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := performClusterUpgrade(tt.args.kps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("performClusterUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}
