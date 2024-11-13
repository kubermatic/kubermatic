package common

import "testing"

func TestValidatePublicIp(t *testing.T) {
	type args struct {
		ipAddress string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test case 1: private ip address",
			args: args{ipAddress: "10.100.197.9"},
			want: false,
		},
		{
			name: "Test case 2: private ip address",
			args: args{ipAddress: "172.16.1.9"},
			want: false,
		},
		{
			name: "Test case 3: private ip address",
			args: args{ipAddress: "192.168.1.1"},
			want: false,
		},
		{
			name: "Test case 4: public ip address",
			args: args{ipAddress: "167.233.10.245"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePublicIp(tt.args.ipAddress); got != tt.want {
				t.Errorf("ValidatePublicIp() = %v, want %v", got, tt.want)
			}
		})
	}
}
