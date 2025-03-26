/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package source

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gogittransport "github.com/go-git/go-git/v5/plumbing/transport"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gogitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GitSource download the application's source from a git repository.
type GitSource struct {
	Ctx context.Context

	// SeedClient to seed cluster.
	SeedClient ctrlruntimeclient.Client

	Source *appskubermaticv1.GitSource

	// Namespace where credential secrets are stored.
	SecretNamespace string
}

// DownloadSource clone the repository into destination and return the full path to the application's sources.
func (g GitSource) DownloadSource(destination string) (string, error) {
	auth, err := g.authFromCredentials()
	if err != nil {
		return "", err
	}

	checkout := g.getCheckoutStrategy()
	if err := checkout(g.Ctx, destination, g.Source, auth); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("failed to clone repository: file protocol not supported, please check git remote url")
		}
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return path.Join(destination, g.Source.Path), nil
}

// authFromCredentials returns a new transport.AuthMethod according to credentials defined in the GitSource.
// If no credentials are defined in the GitSource then nil is returned.
func (g GitSource) authFromCredentials() (gogittransport.AuthMethod, error) {
	var auth gogittransport.AuthMethod
	credentials := g.Source.Credentials
	if credentials != nil {
		switch credentials.Method {
		case appskubermaticv1.GitAuthMethodPassword:
			username, err := util.GetCredentialFromSecret(g.Ctx, g.SeedClient, g.SecretNamespace, credentials.Username.Name, credentials.Username.Key)
			if err != nil {
				return nil, err
			}

			password, err := util.GetCredentialFromSecret(g.Ctx, g.SeedClient, g.SecretNamespace, credentials.Password.Name, credentials.Password.Key)
			if err != nil {
				return nil, err
			}

			auth = &gogithttp.BasicAuth{Username: username, Password: password}

		case appskubermaticv1.GitAuthMethodToken:
			token, err := util.GetCredentialFromSecret(g.Ctx, g.SeedClient, g.SecretNamespace, credentials.Token.Name, credentials.Token.Key)
			if err != nil {
				return nil, err
			}

			auth = &gogithttp.TokenAuth{Token: token}
		case appskubermaticv1.GitAuthMethodSSHKey:
			privateKey, err := util.GetCredentialFromSecret(g.Ctx, g.SeedClient, g.SecretNamespace, credentials.SSHKey.Name, credentials.SSHKey.Key)
			if err != nil {
				return nil, err
			}

			authssh, err := gogitssh.NewPublicKeys("git", []byte(privateKey), "")
			if err != nil {
				return nil, fmt.Errorf("failed to parse private ssh key: %w", err)
			}
			authssh.HostKeyCallback = ssh.InsecureIgnoreHostKey()

			auth = authssh
		default: // this should not happen.
			return nil, fmt.Errorf("unknown Git authentication method '%s'", credentials.Method)
		}
	}
	return auth, nil
}

// getCheckoutStrategy returns the checkoutFunc according to the GitSource.
func (g GitSource) getCheckoutStrategy() checkoutFunc {
	switch {
	case len(g.Source.Ref.Commit) > 0:
		return checkoutFromCommit

	case len(g.Source.Ref.Branch) > 0:
		return checkoutFromBranch

	case len(g.Source.Ref.Tag) > 0:
		return checkoutFromTag

	default: // this should not happen.
		return func(ctx context.Context, destination string, gitSource *appskubermaticv1.GitSource, auth gogittransport.AuthMethod) error {
			return fmt.Errorf("could not determine which reference to checkout")
		}
	}
}

// checkoutFunc define a function to clone and checkout code from repository defined in gitSource into destination using auth as credentials.
type checkoutFunc func(ctx context.Context, destination string, gitSource *appskubermaticv1.GitSource, auth gogittransport.AuthMethod) error

// checkoutFromCommit clone the repository and checkout the desired commit. The commit must belongs to this branch.
// A shallow clone is performed.
func checkoutFromCommit(ctx context.Context, destination string, gitSource *appskubermaticv1.GitSource, auth gogittransport.AuthMethod) error {
	var repo *git.Repository
	var err error

	repo, err = git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:      gitSource.Remote,
		Auth:     auth,
		Progress: nil,
		// Clone only one branch. The commit must belong to this branch.
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(gitSource.Ref.Branch),
		NoCheckout:    true,
		Tags:          git.NoTags,
	})
	if err != nil {
		return err
	}

	workTree, err := repo.Worktree()
	if err != nil { // This should not happen.
		return fmt.Errorf("failed to get worktree's repository: %w", err)
	}
	commit, err := repo.CommitObject(plumbing.NewHash(gitSource.Ref.Commit))
	if err != nil {
		return fmt.Errorf("can not find commit: %w", err)
	}
	return workTree.Checkout(&git.CheckoutOptions{Hash: commit.Hash})
}

// checkoutFromBranch clone the repository and checkout the desired branch. A shallow clone is performed.
func checkoutFromBranch(ctx context.Context, destination string, gitSource *appskubermaticv1.GitSource, auth gogittransport.AuthMethod) error {
	_, err := git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:           gitSource.Remote,
		Auth:          auth,
		Progress:      nil,
		ReferenceName: plumbing.NewBranchReferenceName(gitSource.Ref.Branch),
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
	})
	return err
}

// checkoutFromTag clone the repository and checkout the desired Tag. A shallow clone is performed.
func checkoutFromTag(ctx context.Context, destination string, gitSource *appskubermaticv1.GitSource, auth gogittransport.AuthMethod) error {
	_, err := git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:           gitSource.Remote,
		Auth:          auth,
		Progress:      nil,
		ReferenceName: plumbing.NewTagReferenceName(gitSource.Ref.Tag),
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
	})
	return err
}
