package usersshkeysagent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap"

	"github.com/fsnotify/fsnotify"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	operatorName = "usersshkeys-notifier"
	namespace    = "kube-system"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	log                *zap.SugaredLogger
	authorizedKeysPath []string
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	authorizedKeysPaths []string) error {
	reconciler := &Reconciler{
		Client:             mgr.GetClient(),
		log:                log,
		authorizedKeysPath: authorizedKeysPaths,
	}

	c, err := controller.New(operatorName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	namePredicate := predicateutil.ByName(resources.UserSSHKeys)
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, namePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for secrets: %v", err)
	}

	if err := reconciler.watchAuthorizedKeys(context.TODO(), authorizedKeysPaths); err != nil {
		return fmt.Errorf("failed to watch authorized_keys files: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	secret, err := r.fetchUserSSHKeySecret(ctx, request.NamespacedName.Namespace)
	if err != nil || secret == nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileUserSSHKeys(secret); err != nil {
		log.Errorw("Failed reconciling user ssh key secret", zap.Error(err))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileUserSSHKeys(secret *corev1.Secret) error {
	for _, path := range r.authorizedKeysPath {
		filesKeys, err := r.getAuthorizedKeys(path)
		if err != nil {
			return err
		}

		secretKeys := reverseMapKeyValue(secret.Data)
		if !reflect.DeepEqual(filesKeys, secretKeys) {
			if err := r.updateAuthorizedKeysFile(path, secret.Data); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) updateAuthorizedKeysFile(path string, data map[string][]byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var sshKeys string

	for _, sshKey := range data {
		sshKeys += fmt.Sprintf("%v\n", string(sshKey))
	}
	if _, err := file.WriteString(sshKeys); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) getAuthorizedKeys(path string) (map[string]struct{}, error) {
	var (
		userSSHKeys = make(map[string]struct{})
		file        *os.File
		err         error
	)

	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			file, err = os.Create(path)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if file == nil {
		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		userSSHKeys[scanner.Text()] = struct{}{}
	}
	return userSSHKeys, nil
}

func (r *Reconciler) watchAuthorizedKeys(ctx context.Context, paths []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					secret, err := r.fetchUserSSHKeySecret(ctx, namespace)
					if err != nil || secret == nil {
						return
					}
					if err := r.updateAuthorizedKeysFile(event.Name, secret.Data); err != nil {
						r.log.Errorw("Cannot update authorized_keys file", "path", event.Name, zap.Error(err))
					}

				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	for _, path := range paths {
		err = watcher.Add(path)
		if err != nil {
			return err
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

func reverseMapKeyValue(data map[string][]byte) map[string]struct{} {
	reveresedMap := make(map[string]struct{}, len(data))

	for _, v := range data {
		reveresedMap[string(v)] = struct{}{}
	}

	return reveresedMap
}
