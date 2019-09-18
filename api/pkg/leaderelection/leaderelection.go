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
	"k8s.io/client-go/tools/record"
)

const (
	namespace     = "kube-system"
	leaseDuration = 15 * time.Second
	renewDeadline = 10 * time.Second
	retryPeriod   = 2 * time.Second
)

// New returns a new leader elector which uses the "hostname + name" as lock identity
func New(name string, leaderElectionClient kubernetes.Interface, recorder record.EventRecorder, callbacks leaderelection.LeaderCallbacks) (*leaderelection.LeaderElector, error) {
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

func RunAsLeader(ctx context.Context, log *zap.SugaredLogger, cfg *rest.Config, recorder record.EventRecorder, leaderName string, callback func(context.Context) error) error {
	leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(cfg, leaderName))
	if err != nil {
		return err
	}

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
		},
	}

	elector, err := New(leaderName, leaderElectionClient, recorder, callbacks)
	if err != nil {
		return fmt.Errorf("failed to create a leaderelection: %v", err)
	}

	elector.Run(leaderElectionCtx)
	return nil
}
