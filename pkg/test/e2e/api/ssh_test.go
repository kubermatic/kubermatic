// +build e2e

package api

import (
	"reflect"
	"testing"

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
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID, 1)
			defer teardown(t)

			sshKey, err := apiRunner.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("can not get create SSH key due error: %v", err)
			}
			sshKeys, err := apiRunner.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("can not list SSH keys due error: %v", err)
			}
			if len(sshKeys) != 1 {
				t.Fatalf("expected one SSH key, got %d", len(sshKeys))
			}
			if !reflect.DeepEqual(sshKeys[0], sshKey) {
				t.Fatalf("expected %v, got %v", sshKey, sshKeys[0])
			}
			// user can't create SSH key with the same name
			_, err = apiRunner.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
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
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID, 1)
			defer teardown(t)

			sshKey, err := apiRunner.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("can not get create SSH key due error: %v", err)
			}

			if err := apiRunner.DeleteUserSSHKey(project.ID, sshKey.ID); err != nil {
				t.Fatalf("can not delete SSH key due error: %v", err)
			}
			sshKeys, err := apiRunner.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("can not list SSH keys due error: %v", err)
			}
			if len(sshKeys) != 0 {
				t.Fatalf("found SSH key")
			}

		})
	}
}
