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

package leaderelection

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	namespace     = "kube-system"
	leaseDuration = 15 * time.Second
	renewDeadline = 10 * time.Second
	retryPeriod   = 2 * time.Second
)

// New returns a new leader elector which uses the "hostname + name" as lock identity
func New(name string, leaderElectionClient kubernetes.Interface, recorder resourcelock.EventRecorder, callbacks leaderelection.LeaderCallbacks) (*leaderelection.LeaderElector, error) {
	// Identity used to distinguish between multiple controller manager instances
	id, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error getting hostname: %s", err.Error())
	}

	// Lock required for leader election
	rl := resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Client: leaderElectionClient.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id + "-" + name,
			EventRecorder: recorder,
		},
	}

	return leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          &rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDeadline,
		RetryPeriod:   retryPeriod,
		Callbacks:     callbacks,
	})
}

func RunAsLeader(ctx context.Context, log *zap.SugaredLogger, cfg *rest.Config, recorder resourcelock.EventRecorder, leaderName string, callback func(context.Context) error) error {
	leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(cfg, leaderName))
	if err != nil {
		return err
	}
	log = log.With("leader-name", leaderName)

	leaderElectionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	callbacks := kubeleaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			log.Info("acquired the leader lease")
			if err := callback(ctx); err != nil {
				log.Error(err)
				cancel()
			}
		},
		OnStoppedLeading: func() {
			// Gets called when we could not renew the lease or the parent context was closed
			log.Info("lost leader lease")
			cancel()
			// We will not do anything anymore at this point, so we must sure we exist here so we get restarted
			// and it becomes visible that there is an issue. If we have any kind of bug in the cmds signal handling
			// we may just get stuck here in a defunct state.
			log.Fatal("Leader lease lost, exiting.")
		},
	}

	elector, err := New(leaderName, leaderElectionClient, recorder, callbacks)
	if err != nil {
		return fmt.Errorf("failed to create a leaderelection: %v", err)
	}

	elector.Run(leaderElectionCtx)
	return nil
}
