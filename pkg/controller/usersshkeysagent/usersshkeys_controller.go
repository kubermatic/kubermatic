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

	"gopkg.in/fsnotify.v1"

	"go.uber.org/zap"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	operatorName = "usersshkeys-controller"
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
	authorizedKeysPaths []string) error {
	reconciler := &Reconciler{
		Client:             mgr.GetClient(),
		log:                log,
		authorizedKeysPath: authorizedKeysPaths,
		events:             make(chan event.GenericEvent),
	}

	c, err := controller.New(operatorName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return fmt.Errorf("failed creating a new runtime controller: %v", err)
	}

	namePredicate := predicateutil.ByName(resources.UserSSHKeys)
	namespacePredicate := predicateutil.ByNamespace(metav1.NamespaceSystem)
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, namePredicate, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for secrets: %v", err)
	}

	if err := reconciler.watchAuthorizedKeys(context.TODO(), authorizedKeysPaths); err != nil {
		return fmt.Errorf("failed to watch authorized_keys files: %v", err)
	}

	userSSHKeySecret := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: metav1.NamespaceSystem,
					Name:      resources.UserSSHKeys,
				},
			},
		}
	})

	if err := c.Watch(&source.Channel{Source: reconciler.events}, userSSHKeySecret); err != nil {
		return fmt.Errorf("failed to create watch for channelSource: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: resources.UserSSHKeys, Namespace: namespace}, secret); err != nil {
		if kubeapierrors.IsNotFound(err) {
			r.log.Debugw("Secret is not found", "secret", secret.Name)
			return nil, nil
		}
		r.log.Errorw("Cannot get secret", "secret", resources.UserSSHKeys)
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

// newEventHandler takes a obj->request mapper function and wraps it into an
// handler.EnqueueRequestsFromMapFunc.
func newEventHandler(rf handler.ToRequestsFunc) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{
		ToRequests: rf,
	}
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
		return fmt.Errorf("faild describing the authorized_keys file in path %s: %v", userHome, err)
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
