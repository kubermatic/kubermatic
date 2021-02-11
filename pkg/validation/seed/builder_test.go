/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package seed

import (
	"context"
	"testing"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TODO(irozzo) Add some tests
func TestBuilder(t *testing.T) {
	tests := []struct {
		name    string
		client  ctrlruntimeclient.Client
		wantErr bool
	}{
		{
			name:    "No client provided",
			client:  nil,
			wantErr: true,
		},
		{
			name:    "Client provided",
			client:  fakectrlruntimeclient.NewClientBuilder().Build(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&ValidationHandlerBuilder{}).
				Client(tt.client).
				Build(context.TODO())
			if (err == nil) == tt.wantErr {
				t.Errorf("Expected validation error = %v, but got: %v", tt.wantErr, err)
			}
		})
	}
}
