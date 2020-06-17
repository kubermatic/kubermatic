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

package rbac

import (
	"errors"
	"fmt"
	"strings"
	"time"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const (
	metricNamespace = "kubermatic"
	destinationSeed = "seed"

	// MasterProviderPrefix denotes prefix a master cluster has in its name
	MasterProviderPrefix = "master"
	// SeedProviderPrefix denotes prefix a seed cluster has in its name
	SeedProviderPrefix = "seed"
)

type projectController struct {
	projectQueue workqueue.RateLimitingInterface
	metrics      *Metrics

	projectLister            kubermaticv1lister.ProjectLister
	userLister               kubermaticv1lister.UserLister
	userProjectBindingLister kubermaticv1lister.UserProjectBindingLister

	seedClusterProviders  []*ClusterProvider
	masterClusterProvider *ClusterProvider

	projectResources []projectResource
}

// newProjectRBACController creates a new controller that is responsible for
// managing RBAC roles for project's

// The controller will also set proper ownership chain through OwnerReferences
// so that whenever a project is deleted dependants object will be garbage collected.
func newProjectRBACController(metrics *Metrics, allClusterProviders []*ClusterProvider, resources []projectResource) (*projectController, error) {
	c := &projectController{
		projectQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_for_project"),
		metrics:          metrics,
		projectResources: resources,
	}

	for _, clusterProvider := range allClusterProviders {
		if strings.HasPrefix(clusterProvider.providerName, MasterProviderPrefix) {
			c.masterClusterProvider = clusterProvider
		}
		if strings.HasPrefix(clusterProvider.providerName, SeedProviderPrefix) {
			c.seedClusterProviders = append(c.seedClusterProviders, clusterProvider)
		}
	}

	if c.masterClusterProvider == nil {
		return nil, errors.New("cannot create controller because master cluster provider has not been found")
	}

	projectInformer := c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().Projects()
	userInformer := c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().Users()

	projectInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProject(obj.(*kubermaticv1.Project))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueProject(cur.(*kubermaticv1.Project))
		},
		DeleteFunc: func(obj interface{}) {
			project, ok := obj.(*kubermaticv1.Project)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				project, ok = tombstone.Obj.(*kubermaticv1.Project)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Project %#v", obj))
					return
				}
			}
			c.enqueueProject(project)
		},
	})

	c.projectLister = projectInformer.Lister()
	c.userLister = userInformer.Lister()
	c.userProjectBindingLister = c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings().Lister()

	return c, nil
}

// run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *projectController) run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runProjectWorker, time.Second, stopCh)
		c.metrics.Workers.Inc()
	}

	klog.Info("RBAC generator for project controller started")
	<-stopCh
}

func (c *projectController) runProjectWorker() {
	for c.processProjectNextItem() {
	}
}

func (c *projectController) processProjectNextItem() bool {
	key, quit := c.projectQueue.Get()
	if quit {
		return false
	}
	defer c.projectQueue.Done(key)

	err := c.sync(key.(string))

	handleErr(err, key, c.projectQueue)
	return true
}

func (c *projectController) enqueueProject(project *kubermaticv1.Project) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(project)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", project, err))
		return
	}

	c.projectQueue.Add(key)
}

// handleErr checks if an error happened and makes sure we will retry later.
func handleErr(err error, key interface{}, queue workqueue.RateLimitingInterface) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		queue.Forget(key)
		return
	}

	klog.Errorf("Error syncing %v: %v", key, err)

	// Re-enqueue an item, based on the rate limiter on the
	// queue and the re-enqueueProject history, the key will be processed later again.
	queue.AddRateLimited(key)
}
