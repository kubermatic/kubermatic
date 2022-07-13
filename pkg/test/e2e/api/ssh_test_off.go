//go:build e2e

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

package api

import (
	"context"
	"reflect"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateSSHKey(t *testing.T) {
	tests := []struct {
		name      string
		keyName   string
		publicKey string
	}{
		{
			name:      "create user SSH key",
			keyName:   "test",
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com ",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)
			project, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

			sshKey, err := testClient.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("failed to get create SSH key: %v", err)
			}
			sshKeys, err := testClient.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("failed to list SSH keys: %v", err)
			}
			if len(sshKeys) != 1 {
				t.Fatalf("expected one SSH key, got %d", len(sshKeys))
			}
			if !reflect.DeepEqual(sshKeys[0], sshKey) {
				t.Fatalf("expected %v, got %v", sshKey, sshKeys[0])
			}

			// user can't create SSH key with the same name
			_, err = testClient.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err == nil {
				t.Fatalf("expected error, shouldn't create SSH key with existing name")
			}
		})
	}
}

func TestDeleteSSHKey(t *testing.T) {
	tests := []struct {
		name      string
		keyName   string
		publicKey string
	}{
		{
			name:      "create and delete user SSH key",
			keyName:   "test",
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com ",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)
			project, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

			sshKey, err := testClient.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("failed to get create SSH key: %v", err)
			}

			if err := testClient.DeleteUserSSHKey(project.ID, sshKey.ID); err != nil {
				t.Fatalf("failed to delete SSH key: %v", err)
			}
			sshKeys, err := testClient.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("failed to list SSH keys: %v", err)
			}
			if len(sshKeys) != 0 {
				t.Fatal("found SSH key")
			}
		})
	}
}
