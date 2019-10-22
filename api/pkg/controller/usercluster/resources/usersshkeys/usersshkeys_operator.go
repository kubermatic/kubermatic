package usersshkeys

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

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
		Client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New(operatorName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watcher for secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	secret := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: resources.UserSSHKeys, Namespace: request.NamespacedName.Namespace}, secret); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debugw("Secret is not found", "secret", secret.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.reconcileUserSSHKeys(secret); err != nil {
		log.Errorw("Failed reconciling user ssh key secret", zap.Error(err))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileUserSSHKeys(secret *corev1.Secret) error {
	existingKeys, err := r.getDefaultAuthorizedKeys(r.authorizedKeysPath)
	if err != nil {
		return err
	}

	for _, s := range secret.Data {
		for k, _ := range existingKeys {
			existingKeys[k] = append(existingKeys[k], string(s))
		}
	}

	return r.updateAuthorizedKeysFile(existingKeys)
}

// TODO(MQ): Sync the secrets once the authorized_keys file has been updated.
func (r *Reconciler) updateSecret(secret *corev1.Secret, existingKeys map[string]struct{}) error {
	panic("implement me")
}

func (r *Reconciler) updateAuthorizedKeysFile(data map[string][]string) error {
	for path, keys := range data {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		var sshKeys string

		r.log.Debugw("Number of keys per path", len(keys))
		for _, key := range keys {
			sshKeys = sshKeys + fmt.Sprintf("%v\n", key)
		}
		if _, err := file.WriteString(sshKeys); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) getDefaultAuthorizedKeys(paths []string) (map[string][]string, error) {
	var keys = make(map[string][]string, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		// At the moment we need to read the first line of the authorized_keys file. The reason for that is
		// the ssh key which is mounted by default per machine. Since this ssh key is not saved in the
		// usersshkeys secret, thus we need to read from file, to write it again in the updated authorized_keys
		// TODO(MQ): Find a better way to sync the default ssh key. Hint: save it in the usersshkeys secret like the other keys?
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			keys[path] = append(keys[path], scanner.Text())
			break
		}

	}
	return keys, nil
}
