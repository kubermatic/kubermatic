package seedcontrollerlifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "seedcontroller_lifecycle_manager"

	// We must only enqueue this one key
	queueKey = ControllerName
)

type Reconciler struct {
	ctx                  context.Context
	log                  *zap.SugaredLogger
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	controllerFactory    func() (manager.Runnable, error)
	enqueue              func()
	activeController     *controllersInstance
}

// controllersInstance represents an instance of a set of running controllers
type controllersInstance struct {
	log        *zap.SugaredLogger
	configHash string
	controller manager.Runnable
	running    bool
	stopChan   chan struct{}
	enqueue    func()
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactory func() (manager.Runnable, error),
) error {

	reconciler := &Reconciler{
		ctx:                  ctx,
		log:                  log.Named(ControllerName),
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		controllerFactory:    controllerFactory,
	}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for _, t := range []runtime.Object{&kubermaticv1.Seed{}, &corev1.Secret{}} {
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueConst(queueKey)); err != nil {
			return fmt.Errorf("failed to create watch for type %T: %v", t, err)
		}
	}

	sourceChannel := make(chan event.GenericEvent)
	reconciler.enqueue = func() {
		sourceChannel <- event.GenericEvent{
			// TODO: Is it needed to fill this?
			Object: &kubermaticv1.Seed{},
		}
	}
	if err := c.Watch(&source.Channel{Source: sourceChannel}, controllerutil.EnqueueConst(queueKey)); err != nil {
		return fmt.Errorf("failed to create watch for channelSource: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile()
	if err != nil {
		r.log.With("error", err).Error("reconciliation failed")
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile() error {
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}

	seedKubeconfigMap := map[string]rest.Config{}
	for seedName := range seeds {
		cfg, err := r.seedKubeconfigGetter(seedName)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig for seed %q: %v", seedName, err)
		}
		seedKubeconfigMap[seedName] = *cfg
	}

	configHash, err := r.seedKubeconfigHashStable(seedKubeconfigMap)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for seed kubeconfigs: %v", err)
	}
	log := r.log.With("config_hash", configHash)

	if r.activeController != nil && r.activeController.configHash == configHash {
		if !r.activeController.running {
			log.Info("found matching controllers but were not running, starting them")
			r.activeController.controller.Start(r.activeController.stopChan)
		}
		return nil
	}

	controllerInstance, err := r.controllerFactory()
	if err != nil {
		return fmt.Errorf("failed to construct controllers: %v", err)
	}

	if r.activeController != nil && r.activeController.running {
		log.Info("Stopping old version of controllers")
		close(r.activeController.stopChan)
	}

	r.activeController = &controllersInstance{
		configHash: configHash,
		controller: controllerInstance,
		stopChan:   make(chan struct{}),
	}
	go func() {
		if err := r.activeController.controller.Start(r.activeController.stopChan); err != nil {
			log.Errorw("controllers stopped with error", zap.Error(err))
		}
		log.Debug("controllers stopped")
		// Make sure we check on this
		r.activeController.running = false
		r.enqueue()
	}()

	return nil
}

// seedKubeconfigStableHash returns a hash over the seedNames + associated kubeconfigs
// it is used to determine config changes, so it must not rely on ordering of items it gets
func (r *Reconciler) seedKubeconfigHashStable(seedKubeConfigs map[string]rest.Config) (string, error) {

	var sortedNames []string
	for name := range seedKubeConfigs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var rawConfigData []byte
	for _, seed := range sortedNames {
		// Function types can not be marshalled, so we have to extract the marshallable parts...
		cfg := seedKubeConfigs[seed]
		identifier := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
			seed, cfg.Host, cfg.APIPath, cfg.Username, cfg.Password, cfg.BearerToken, cfg.BearerTokenFile,
			string(cfg.CertData), string(cfg.KeyData), string(cfg.CAData))
		jsonEncodedRestConfig, err := json.Marshal([]byte(identifier))
		if err != nil {
			return "", fmt.Errorf("failed to marshal restconfig: %v", err)
		}
		rawConfigData = append(rawConfigData, jsonEncodedRestConfig...)
	}

	checksumRaw := sha256.Sum256(rawConfigData)
	return string(checksumRaw[:]), nil
}
