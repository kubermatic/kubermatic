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

package usersshkeysagent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"gopkg.in/fsnotify.v1"

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	operatorName = "usersshkeys-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	log                *zap.SugaredLogger
	authorizedKeysPath []string
	events             chan event.GenericEvent
	secretName         string
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	authorizedKeysPaths []string,
	secretName string) error {
	reconciler := &Reconciler{
		Client:             mgr.GetClient(),
		log:                log,
		authorizedKeysPath: authorizedKeysPaths,
		events:             make(chan event.GenericEvent),
		secretName:         secretName,
	}

	c, err := controller.New(operatorName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return fmt.Errorf("failed creating a new runtime controller: %v", err)
	}

	namePredicate := predicateutil.ByName(reconciler.secretName)
	namespacePredicate := predicateutil.ByNamespace(metav1.NamespaceSystem)
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, namePredicate, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for secrets: %v", err)
	}

	if err := reconciler.watchAuthorizedKeys(context.TODO(), authorizedKeysPaths); err != nil {
		return fmt.Errorf("failed to watch authorized_keys files: %v", err)
	}

	userSSHKeySecret := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: metav1.NamespaceSystem,
					Name:      reconciler.secretName,
				},
			},
		}
	})

	if err := c.Watch(&source.Channel{Source: reconciler.events}, userSSHKeySecret); err != nil {
		return fmt.Errorf("failed to create watch for channelSource: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Processing")

	secret, err := r.fetchUserSSHKeySecret(ctx, request.NamespacedName.Namespace)
	if err != nil || secret == nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch user ssh keys: %v", err)
	}

	if err := r.updateAuthorizedKeys(secret.Data); err != nil {
		r.log.Errorw("Failed reconciling user ssh key secret", zap.Error(err))
		return reconcile.Result{}, fmt.Errorf("failed to reconcile user ssh keys: %v", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) watchAuthorizedKeys(ctx context.Context, paths []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed creating a new file watcher: %v", err)
	}

	go func() {
		for {
			select {
			case e, ok := <-watcher.Events:
				if !ok {
					return
				}
				if e.Op&fsnotify.Write == fsnotify.Write {
					r.events <- event.GenericEvent{}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				r.log.Errorw("Error occurred during watching authorized_keys file", zap.Error(err))
				r.events <- event.GenericEvent{}
			}
		}
	}()

	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("failed adding a new path to the files watcher: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) fetchUserSSHKeySecret(ctx context.Context, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: r.secretName, Namespace: namespace}, secret); err != nil {
		if kubeapierrors.IsNotFound(err) {
			r.log.Debugw("Secret is not found", "secret", secret.Name)
			return nil, nil
		}
		r.log.Errorw("Cannot get secret", "secret", r.secretName)
		return nil, err
	}

	return secret, nil
}

func (r *Reconciler) updateAuthorizedKeys(sshKeys map[string][]byte) error {
	expectedUserSSHKeys, err := createBuffer(sshKeys)
	if err != nil {
		return fmt.Errorf("failed creating user ssh keys buffer: %v", err)
	}

	for _, path := range r.authorizedKeysPath {
		if err := updateOwnAndPermissions(path); err != nil {
			return fmt.Errorf("failed updating permissions %s: %v", path, err)
		}

		actualUserSSHKeys, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed reading file in path %s: %v", path, err)
		}

		if !bytes.Equal(actualUserSSHKeys, expectedUserSSHKeys.Bytes()) {
			if err := ioutil.WriteFile(path, expectedUserSSHKeys.Bytes(), 0600); err != nil {
				return fmt.Errorf("failed to overwrite file in path %s: %v", path, err)
			}
			r.log.Infow("File has been updated successfully", "file", path)
		}
	}

	return nil
}

func createBuffer(data map[string][]byte) (*bytes.Buffer, error) {
	var (
		keys   = make([]string, 0, len(data))
		buffer = &bytes.Buffer{}
	)

	for key := range data {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for k := range keys {
		key := keys[k]
		data[key] = append(data[key], []byte("\n")...)
		if _, err := buffer.Write(data[key]); err != nil {
			return nil, fmt.Errorf("failed writing user ssh keys to buffer: %v", err)
		}
	}

	return buffer, nil
}

func updateOwnAndPermissions(path string) error {
	sshPath := strings.TrimSuffix(path, "/authorized_keys")
	if err := os.Chmod(sshPath, os.FileMode(0700)); err != nil {
		return fmt.Errorf("failed to change permission on file: %v", err)
	}

	if err := os.Chmod(path, os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed to change permission on file: %v", err)
	}

	userHome := strings.TrimSuffix(sshPath, "/.ssh")
	fileInfo, err := os.Stat(userHome)
	if err != nil {
		return fmt.Errorf("failed describing the authorized_keys file in path %s: %v", userHome, err)
	}

	uid := fileInfo.Sys().(*syscall.Stat_t).Uid
	gid := fileInfo.Sys().(*syscall.Stat_t).Gid

	if err := os.Chown(path, int(uid), int(gid)); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %v", path, err)
	}

	if err := os.Chown(sshPath, int(uid), int(gid)); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %v", sshPath, err)
	}

	return nil
}
