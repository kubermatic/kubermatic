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
	"os"
	"sort"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"gopkg.in/fsnotify.v1"

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kkp-usersshkeys-controller"

	// kkpManagedMarker is a comment prefix the agent writes before each KKP-managed key.
	// This allows the agent to distinguish its own keys from external keys (e.g. from
	// cloud-init/machine-deployment) so that removed KKP keys can be cleaned up without
	// affecting externally-managed keys.
	kkpManagedMarker = "# kkp-managed"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	log                *zap.SugaredLogger
	authorizedKeysPath []string
	events             chan event.GenericEvent
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	authorizedKeysPaths []string,
) error {
	reconciler := &Reconciler{
		Client:             mgr.GetClient(),
		log:                log,
		authorizedKeysPath: authorizedKeysPaths,
		events:             make(chan event.GenericEvent),
	}

	if err := reconciler.watchAuthorizedKeys(authorizedKeysPaths); err != nil {
		return fmt.Errorf("failed to watch authorized_keys files: %w", err)
	}

	userSSHKeySecret := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: metav1.NamespaceSystem,
					Name:      resources.UserSSHKeys,
				},
			},
		}
	})

	namePredicate := predicateutil.ByName(resources.UserSSHKeys)
	namespacePredicate := predicateutil.ByNamespace(metav1.NamespaceSystem)

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		For(&corev1.Secret{}, builder.WithPredicates(namePredicate, namespacePredicate)).
		WatchesRawSource(source.Channel(reconciler.events, userSSHKeySecret)).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Processing")

	secret, err := r.fetchUserSSHKeySecret(ctx, request.Namespace)
	if err != nil || secret == nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch user ssh keys: %w", err)
	}

	if err := r.updateAuthorizedKeys(secret.Data); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile user ssh keys: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) watchAuthorizedKeys(paths []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed creating a new file watcher: %w", err)
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
			return fmt.Errorf("failed adding a new path to the files watcher: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) fetchUserSSHKeySecret(ctx context.Context, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: resources.UserSSHKeys, Namespace: namespace}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Debugw("Secret is not found", "secret", secret.Name)
			return nil, nil
		}
		r.log.Errorw("Cannot get secret", "secret", resources.UserSSHKeys)
		return nil, err
	}

	return secret, nil
}

func (r *Reconciler) updateAuthorizedKeys(sshKeys map[string][]byte) error {
	kkpKeys := sortedKKPKeys(sshKeys)

	// deduplicate keys by dropping external keys that match a KKP key so we don't
	// end up with two copies (one marked, one unmarked) after an
	// upgrade or when cloud-init injects the same key via MD
	kkpSet := make(map[string]struct{}, len(kkpKeys))
	for _, nk := range kkpKeys {
		kkpSet[nk.Key] = struct{}{}
	}

	for _, path := range r.authorizedKeysPath {
		if err := updateOwnAndPermissions(path); err != nil {
			return fmt.Errorf("failed updating permissions %s: %w", path, err)
		}

		actualContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed reading file in path %s: %w", path, err)
		}

		// keep keys that are not marked as kkp-managed
		externalKeys := extractExternalKeys(actualContent)

		filtered := make([]string, 0, len(externalKeys))
		for _, k := range externalKeys {
			if _, dup := kkpSet[strings.TrimSpace(k)]; !dup {
				filtered = append(filtered, k)
			}
		}

		merged := mergeAuthorizedKeys(kkpKeys, filtered)

		if !bytes.Equal(actualContent, merged) {
			err = os.WriteFile(path, merged, 0600)
			if err != nil {
				return fmt.Errorf("failed to write file in path %q: %w", path, err)
			}

			r.log.Infow("File has been updated successfully", "file", path)
		}
	}

	return nil
}

// NamedKey pairs a KKP UserSSHKey name with its public key value.
type NamedKey struct {
	Name string
	Key  string
}

// sortedKKPKeys returns the KKP SSH keys sorted alphabetically by key value,
// preserving the Secret map key (UserSSHKey object name) for marker comments.
func sortedKKPKeys(sshKeys map[string][]byte) []NamedKey {
	keys := make([]NamedKey, 0, len(sshKeys))
	for name, v := range sshKeys {
		keys = append(keys, NamedKey{
			Name: name,
			Key:  strings.TrimSpace(string(v)),
		})
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Key < keys[j].Key
	})

	return keys
}

// AuthorizedKeysView is the parsed form of an authorized_keys file: KKP-managed
// keys keyed by their UserSSHKey object name, plus external (unmarked) keys.
type AuthorizedKeysView struct {
	Managed  map[string]string // UserSSHKey name -> public key value
	External []string          // public key lines without a kkp-managed marker
}

// ParseAuthorizedKeys parses an authorized_keys file and splits keys into
// KKP-managed (preceded by a kkpManagedMarker comment) and external.
func ParseAuthorizedKeys(content string) AuthorizedKeysView {
	view := AuthorizedKeysView{Managed: map[string]string{}}

	lines := strings.Split(content, "\n")
	var pendingName string
	skipNext := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, kkpManagedMarker) {
			pendingName = ""
			if rest := strings.TrimPrefix(line, kkpManagedMarker); strings.HasPrefix(rest, ":") {
				pendingName = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
			}
			skipNext = true
			continue
		}

		if skipNext {
			view.Managed[pendingName] = line
			pendingName = ""
			skipNext = false
			continue
		}

		view.External = append(view.External, line)
	}

	return view
}

// extractExternalKeys parses an authorized_keys file and returns key lines
// that are not preceded by the kkpManagedMarker comment.
func extractExternalKeys(content []byte) []string {
	return ParseAuthorizedKeys(string(content)).External
}

// mergeAuthorizedKeys builds the final authorized_keys content. Each KKP key
// is preceded by a kkpManagedMarker comment (including the key name for
// readability). External keys are appended without a marker.
func mergeAuthorizedKeys(kkpKeys []NamedKey, externalKeys []string) []byte {
	var buf bytes.Buffer

	for _, nk := range kkpKeys {
		fmt.Fprintf(&buf, "%s: %s\n%s\n", kkpManagedMarker, nk.Name, nk.Key)
	}

	for _, key := range externalKeys {
		buf.WriteString(key)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}

func updateOwnAndPermissions(path string) error {
	sshPath := strings.TrimSuffix(path, "/authorized_keys")
	if err := os.Chmod(sshPath, os.FileMode(0700)); err != nil {
		return fmt.Errorf("failed to change permission on file: %w", err)
	}

	if err := os.Chmod(path, os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed to change permission on file: %w", err)
	}

	userHome := strings.TrimSuffix(sshPath, "/.ssh")
	fileInfo, err := os.Stat(userHome)
	if err != nil {
		return fmt.Errorf("failed describing the authorized_keys file in path %s: %w", userHome, err)
	}

	uid := fileInfo.Sys().(*syscall.Stat_t).Uid
	gid := fileInfo.Sys().(*syscall.Stat_t).Gid

	if err := os.Chown(path, int(uid), int(gid)); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %w", path, err)
	}

	if err := os.Chown(sshPath, int(uid), int(gid)); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %w", sshPath, err)
	}

	return nil
}

// CacheOptions returns a user-ssh-keys-agent specific cache.Options struct that limits the cache
// to the Secret object that is needed by the controller. This is done so we can limit the RBAC
// assignment for this controller to the bare minimum (the resource name).
func CacheOptions() cache.Options {
	return cache.Options{
		DefaultNamespaces: map[string]cache.Config{
			metav1.NamespaceSystem: {
				FieldSelector: fields.SelectorFromSet(fields.Set{"metadata.name": resources.UserSSHKeys}),
			},
		},
		ByObject: map[ctrlruntimeclient.Object]cache.ByObject{
			&corev1.Secret{}: {
				Field: fields.SelectorFromSet(fields.Set{"metadata.name": resources.UserSSHKeys}),
			},
		},
	}
}
