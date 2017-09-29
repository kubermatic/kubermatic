package update

import (
	"testing"

	"k8s.io/api/apps/v1beta1"
)

func Test_healthyDep(t *testing.T) {
	type args struct {
		dep *v1beta1.Deployment
	}

	tests := []struct {
		name string
		args args
		want bool
	}{{

		name: "equal",
		args: args{
			dep: &v1beta1.Deployment{
				Spec: v1beta1.DeploymentSpec{
					Replicas: &[]int32{1}[0],
				},
				Status: v1beta1.DeploymentStatus{
					UpdatedReplicas: 1,
				},
			},
		},
		want: true,
	},
		{

			name: "zero",
			args: args{
				dep: &v1beta1.Deployment{
					Spec: v1beta1.DeploymentSpec{
						Replicas: &[]int32{1}[0],
					},
					Status: v1beta1.DeploymentStatus{
						UpdatedReplicas: 0,
					},
				},
			},
			want: false,
		},
		{

			name: "80%",
			args: args{
				dep: &v1beta1.Deployment{
					Spec: v1beta1.DeploymentSpec{
						Replicas: &[]int32{10}[0],
					},
					Status: v1beta1.DeploymentStatus{
						UpdatedReplicas: 8,
					},
				},
			},
			want: false,
		},
		{

			name: "90%",
			args: args{
				dep: &v1beta1.Deployment{
					Spec: v1beta1.DeploymentSpec{
						Replicas: &[]int32{10}[0],
					},
					Status: v1beta1.DeploymentStatus{
						UpdatedReplicas: 9,
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := healthyDep(tt.args.dep); got != tt.want {
				t.Errorf("healthyDep() = %v, want %v", got, tt.want)
			}
		})
	}
}
