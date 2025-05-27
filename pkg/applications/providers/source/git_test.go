//go:build integration

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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/phayes/freeport"
	"github.com/sosedoff/gitkit"
	"golang.org/x/crypto/ssh"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDownloadGitSource(t *testing.T) {
	gitServerDir, err := os.MkdirTemp("", "git-repo")
	defer os.RemoveAll(gitServerDir)
	fatalOnErr(err, "failed to create temporary directory for git server", t)

	repoInfo := createGitRepository(t, gitServerDir)
	privateKey, publicKey := generateSSHKey(t)

	sshRemote := "ssh://git@" + newGitSSHServer(gitServerDir, publicKey, t) + repoInfo.Name
	httpRemote := newGitHTTPServer(gitServerDir, "", "", t) + repoInfo.Name
	httpWithAuthRemote := newGitHTTPServer(gitServerDir, "user", "pass", t) + repoInfo.Name

	secretName := "git-cred"
	passwordCredentials := appskubermaticv1.GitCredentials{
		Method:   appskubermaticv1.GitAuthMethodPassword,
		Username: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secretName}, Key: "username"},
		Password: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secretName}, Key: "password"},
	}
	sshCredentials := &appskubermaticv1.GitCredentials{
		Method: appskubermaticv1.GitAuthMethodSSHKey,
		SSHKey: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secretName}, Key: "ssh-key"}}

	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic", Name: secretName},
		Data:       map[string][]byte{"username": []byte("user"), "password": []byte("pass"), "ssh-key": []byte(privateKey)},
	}

	testCases := []struct {
		name             string
		client           ctrlruntimeclient.Client
		source           *appskubermaticv1.GitSource
		gitServerType    string
		expectedCommit   string
		shallowCheckFunc func(repository *git.Repository, t *testing.T)
	}{
		{
			name: "scenario 1: clone HTTP from branch",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name},
				Path:        "",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.MasterBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 2: clone HTTP from branch with path /",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name},
				Path:        "/",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.MasterBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 3: clone HTTP from branch with path subdir",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.FeatureBranch.Name},
				Path:        "subdir",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.FeatureBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 4: clone HTTP from tag",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 5: clone HTTP from Tag with path /",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "/",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 6: clone HTTP from Tag with path subdir",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "subdir",
				Credentials: nil,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name: "scenario 7: clone HTTP from commit in branch",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Commit: repoInfo.CommitInMaster.Hash, Branch: repoInfo.MasterBranch.Name},
				Path:        "",
				Credentials: nil,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name: "scenario 8: clone HTTP from commit in branch with path /",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name, Commit: repoInfo.CommitInMaster.Hash},
				Path:        "/",
				Credentials: nil,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name: "scenario 9: clone HTTP from commit in branch with path subdir",
			source: &appskubermaticv1.GitSource{
				Remote:      httpRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name, Commit: repoInfo.CommitInMaster.Hash},
				Path:        "subdir",
				Credentials: nil,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name:   "scenario 10: clone HTTP with auth from branch",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      httpWithAuthRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name},
				Path:        "",
				Credentials: &passwordCredentials,
			},
			expectedCommit:   repoInfo.MasterBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 11: clone HTTP with auth from Tag",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      httpWithAuthRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "",
				Credentials: &passwordCredentials,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 12: clone HTTP with auth from commit in branch",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      httpWithAuthRemote,
				Ref:         appskubermaticv1.GitReference{Commit: repoInfo.CommitInMaster.Name, Branch: repoInfo.MasterBranch.Name},
				Path:        "",
				Credentials: &passwordCredentials,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name:   "scenario 13: clone SSH from branch",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.FeatureBranch.Name},
				Path:        "",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.FeatureBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 14: clone SSH from branch with path /",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.FeatureBranch.Name},
				Path:        "/",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.FeatureBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 15: clone SSH from branch with path subdir",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.FeatureBranch.Name},
				Path:        "subdir",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.FeatureBranch.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 16: clone SSH from tag",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 17: clone SSH from tag with path /",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "/",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 18: clone SSH from tag with path subdir",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Tag: repoInfo.Tag.Name},
				Path:        "subdir",
				Credentials: sshCredentials,
			},
			expectedCommit:   repoInfo.Tag.Hash,
			shallowCheckFunc: checkOnlyOneCommit,
		},
		{
			name:   "scenario 19: clone SSH from commit in branch",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name, Commit: repoInfo.CommitInMaster.Name},
				Path:        "",
				Credentials: sshCredentials,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name:   "scenario 20: clone SSH from commit in branch with path /",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name, Commit: repoInfo.CommitInMaster.Name},
				Path:        "/",
				Credentials: sshCredentials,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
		{
			name:   "scenario 21: clone SSH from commit in branch with path subdir",
			client: fake.NewClientBuilder().WithObjects(credentialsSecret).Build(),
			source: &appskubermaticv1.GitSource{
				Remote:      sshRemote,
				Ref:         appskubermaticv1.GitReference{Branch: repoInfo.MasterBranch.Name, Commit: repoInfo.CommitInMaster.Name},
				Path:        "subdir",
				Credentials: sshCredentials,
			},
			expectedCommit: repoInfo.CommitInMaster.Hash,
			shallowCheckFunc: func(repository *git.Repository, t *testing.T) {
				checkOnlyDesiredReferences(repository, map[string]struct{}{repoInfo.MasterBranch.Name: {}}, t)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dest, err := os.MkdirTemp("", "testGitSource-downloadDest")
			fatalOnErr(err, "failed to create temporary directory where sources will be downloaded", t)
			defer os.RemoveAll(dest)

			source := GitSource{
				Ctx:             context.Background(),
				SeedClient:      tc.client,
				Source:          tc.source,
				SecretNamespace: "kubermatic",
			}

			downloadedSource, err := source.DownloadSource(dest)
			fatalOnErr(err, "failed to download sources: %s", t)

			// check path exist in downloaded sources
			expectedPath := path.Join(dest, source.Source.Path)
			if downloadedSource != expectedPath {
				t.Fatalf("error: path returned by source.DownloadSource() should be '%s', got '%s'", expectedPath, downloadedSource)
			}
			_, err = os.Stat(downloadedSource)
			fatalOnErr(err, "error path returned by source.DownloadSource() does not exit: %s", t)

			// check the expected "ref" has been checkout
			repository, err := git.PlainOpen(dest)
			fatalOnErr(err, "failed to open git repo", t)

			head, err := repository.Head()
			fatalOnErr(err, "failed to get Head", t)

			currentCommit := head.Hash().String()
			if currentCommit != tc.expectedCommit {
				t.Errorf("download invalid. Expect repository to point on  '%s', but was on '%s'", tc.expectedCommit, currentCommit)
			}

			// check shallow clone
			tc.shallowCheckFunc(repository, t)
		})
	}
}

// checkOnlyDesiredReferences fails the test if repository has different references than desiredRefSet.
// it can be used to validate a shallow clone of branch.
func checkOnlyDesiredReferences(repository *git.Repository, desiredRefSet map[string]struct{}, t *testing.T) {
	iter, _ := repository.References()

	refSet := map[string]struct{}{}
	if err := iter.ForEach(func(reference *plumbing.Reference) error {
		shortName := reference.Name().Short()
		if shortName != "HEAD" {
			// avoid duplicate like master and origin/master
			name := strings.Split(shortName, "/")
			refSet[name[len(name)-1]] = struct{}{}
		}
		return nil
	}); err != nil {
		t.Fatalf("failed to iterate on references")
	}

	if !reflect.DeepEqual(desiredRefSet, refSet) {
		t.Fatalf("repository does not contain expected references. expect %v got %v", desiredRefSet, refSet)
	}
}

// checkOnlyOneCommit fails the test if the repository contains more than one commit.
func checkOnlyOneCommit(repository *git.Repository, t *testing.T) {
	itercomit, err := repository.CommitObjects()
	if err != nil {
		t.Fatalf("failed to list commits: %s", err)
	}
	nbCommit := 0
	if err := itercomit.ForEach(func(commit *object.Commit) error {
		nbCommit++
		return nil
	}); err != nil {
		t.Fatalf("failed to iterate over the commits")
	}
	if nbCommit != 1 {
		t.Fatalf("only one commit should have been clone. got %d commits", nbCommit)
	}
}

// generateSSHKey a private and public ssh key and returned it in that order.
func generateSSHKey(t *testing.T) (string, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	fatalOnErr(err, "failed to genetate client private key", t)

	privateKeyPEM := strings.Trim(string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})), " \n")

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	fatalOnErr(err, "failed to generate client public key", t)
	pubKey := strings.Trim(string(ssh.MarshalAuthorizedKey(pub)), " \n")

	return privateKeyPEM, pubKey
}

func newGitHTTPServer(repoPath string, user string, password string, t *testing.T) string {
	service := gitkit.New(gitkit.Config{
		Dir:        repoPath,
		AutoCreate: false,
		AutoHooks:  false,
		Auth:       len(user) > 0 && len(password) > 0, // Auth is enabled only if user and password are defined.
	})

	service.AuthFunc = func(cred gitkit.Credential, req *gitkit.Request) (bool, error) {
		return cred.Username == user && cred.Password == password, nil
	}

	server := httptest.NewServer(service)
	t.Cleanup(func() { server.Close() })

	return server.URL
}

func newGitSSHServer(repoPath string, pub string, t *testing.T) string {
	// When starting server it will generate a pair of key for the server in this folder.
	keyDir, err := os.MkdirTemp("", "git-server-ssh")
	fatalOnErr(err, "failed to create temporary directory where git server will store ssh key", t)

	t.Cleanup(func() {
		os.RemoveAll(keyDir)
	})

	server := gitkit.NewSSH(gitkit.Config{
		KeyDir: keyDir,
		Dir:    repoPath,
		Auth:   true,
	})

	server.PublicKeyLookupFunc = func(s string) (*gitkit.PublicKey, error) {
		if s != pub {
			return nil, fmt.Errorf("unauthozied key providided: %s", s)
		}
		return &gitkit.PublicKey{
			Id: "123",
		}, nil
	}

	port, err := freeport.GetFreePort()
	fatalOnErr(err, "failed to get free port for git ssh server", t)

	err = server.Listen(fmt.Sprintf("127.0.0.1:%d", port))
	fatalOnErr(err, "git ssh server can not bind on address", t)
	go func() {
		err := server.Serve()
		fatalOnErr(err, "error git SSH server", t)
	}()
	return server.Address()
}

// createGitRepository create a git repository named repo1 in temp directory and returns information about the repository.
//
// the repository has the following structure:
//   - C5 (feature-1) add subdir2/file5
//     | * C4 (master) add file4
//     | * C3 add file3
//     |/
//   - C2 (tag: v1.0.0) subdir/file2
//   - C1 add file1
//
// The content of the repository for master branch is:
// .
// ├── file1
// ├── file3
// ├── file4
// └── subdir
//
//	└── file2
//
// the content of the repository for feature-1 branch is:
// .
// ├── file1
// ├── subdir
// │   └── file2
// └── subdir2
//
//	└── file5
func createGitRepository(t *testing.T, temp string) repoInfo {
	repoDir := temp + "/repo1"
	err := os.Mkdir(repoDir, 0700)
	fatalOnErr(err, "failed to create repository directory", t)

	repo, err := git.PlainInit(repoDir, false)
	fatalOnErr(err, "failed to init git repository", t)

	// create master with 4 commits
	err = repo.CreateBranch(&config.Branch{Name: "master"})
	fatalOnErr(err, "failed to create master branch", t)

	worktree, err := repo.Worktree()
	fatalOnErr(err, "failed to get worktree", t)

	createFileAndCommit(repoDir, "file1", "content of file1", worktree, t)
	c2Hash := createFileAndCommit(repoDir, "subdir/file2", "content of file2", worktree, t)
	c3Hash := createFileAndCommit(repoDir, "file3", "content of file3", worktree, t)
	masterHash := createFileAndCommit(repoDir, "file4", "content of file4", worktree, t)

	// create breanch feature-1 from commit c2
	err = worktree.Checkout(&git.CheckoutOptions{Hash: c2Hash, Branch: plumbing.NewBranchReferenceName("feature-1"), Create: true})
	fatalOnErr(err, "failed to checkout branch feature-1", t)
	featureHash := createFileAndCommit(repoDir, "subdir2/file5", "content of file5", worktree, t)

	// create tag v1.0.0 on commit c2
	tag, err := repo.CreateTag("v1.0.0", c2Hash, nil)
	fatalOnErr(err, "failed to create tag", t)

	return repoInfo{
		// The git server implementation used for testing expect a bare git repository. So we point on on the .git
		// directory of our non-bare repository.
		Name:           "/repo1/.git",
		MasterBranch:   ref{Name: "master", Hash: masterHash.String()},
		FeatureBranch:  ref{Name: "feature-1", Hash: featureHash.String()},
		CommitInMaster: ref{Name: c3Hash.String(), Hash: c3Hash.String()},
		Tag:            ref{Name: "v1.0.0", Hash: tag.Hash().String()},
	}
}

// createFileAndCommit create a file in the repoDir/fileInRepo with the desired content and commit it to git.
// If fileInRepo contains "parent directory" (e.g. foo/bar/myFile) then  any necessary parents directory will be created
// in the repository.
func createFileAndCommit(repoDir string, fileInRepo string, content string, worktree *git.Worktree, t *testing.T) plumbing.Hash {
	t.Helper()

	fileFullPath := path.Join(repoDir, fileInRepo)
	parentDir := path.Dir(fileFullPath)
	err := os.MkdirAll(parentDir, 0700)
	fatalOnErr(err, "failed to create parentDir "+parentDir, t)

	err = os.WriteFile(fileFullPath, []byte(content+"\n"), 0600)
	fatalOnErr(err, "failed to create file "+fileFullPath, t)

	_, err = worktree.Add(fileInRepo)
	fatalOnErr(err, "failed to add "+fileInRepo+" to repository", t)

	user := object.Signature{
		Name:  "test",
		Email: "test@noreply.com",
		When:  time.Now(),
	}

	hash, err := worktree.Commit("add "+fileInRepo, &git.CommitOptions{
		Author:    &user,
		Committer: &user,
	})
	fatalOnErr(err, "failed to commit "+fileInRepo, t)

	return hash
}

// ref holds the name and the sha1 hash of reference or a commit.
// If ref holds a commit then Name == Hash.
type ref struct {
	Name string
	Hash string
}

type repoInfo struct {
	Name           string
	MasterBranch   ref
	FeatureBranch  ref
	CommitInMaster ref
	Tag            ref
}

// fatalOnErr fails the test if the error message if the error is not nil.
func fatalOnErr(err error, msg string, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}
