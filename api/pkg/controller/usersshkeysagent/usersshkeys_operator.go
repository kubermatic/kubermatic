package usersshkeysagent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"

	"gopkg.in/fsnotify.v1"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/types"
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
)

const (
	operatorName = "usersshkeys-notifier"
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
		return err
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
	log := r.log.With("request", request)
	log.Debug("Processing")

	secret, err := r.fetchUserSSHKeySecret(ctx, request.NamespacedName.Namespace)
	if err != nil || secret == nil {
		return reconcile.Result{}, err
	}

	if err := r.updateAuthorizedKeys(secret.Data); err != nil {
		log.Errorw("Failed reconciling user ssh key secret", zap.Error(err))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) watchAuthorizedKeys(ctx context.Context, paths []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
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
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				r.events <- event.GenericEvent{}
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

func (r *Reconciler) updateAuthorizedKeys(sshKeys map[string][]byte) error {
	expectedUserSSHKeys := bytes.Buffer{}
	for _, secretData := range sshKeys {
		secretData = append(secretData, []byte("\n")...)
		if _, err := expectedUserSSHKeys.Write(secretData); err != nil {
			return err
		}
	}

	for _, path := range r.authorizedKeysPath {
		if _, err := os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err := ioutil.WriteFile(path, expectedUserSSHKeys.Bytes(), 0600); err != nil {
				return err
			}

			return nil
		}

		actualUserSSHKeys, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if !bytes.Equal(actualUserSSHKeys, expectedUserSSHKeys.Bytes()) {
			if err := ioutil.WriteFile(path, expectedUserSSHKeys.Bytes(), 0600); err != nil {
				return err
			}
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
