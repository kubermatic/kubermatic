package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/oklog/run"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/usersshkeys"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/signals"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type controllerRunOptions struct {
	namespace string
	log       kubermaticlog.Options
}

func main() {
	runOp := controllerRunOptions{}
	flag.BoolVar(&runOp.log.Debug, "log-debug", true, "Enables debug logging")
	flag.StringVar(&runOp.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&runOp.namespace, "namespace", metav1.NamespaceSystem, "Namespace in which the cluster is running in")

	flag.Parse()

	if err := runOp.log.Validate(); err != nil {
		fmt.Printf("error occurred while validating zap logger options: %v\n", err)
		os.Exit(1)
	}

	rawLog := kubermaticlog.New(runOp.log.Debug, kubermaticlog.Format(runOp.log.Format))
	log := rawLog.Sugar()

	if runOp.namespace == "" {
		log.Fatal("-namespace must be set")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalw("Failed getting user cluster controller config", zap.Error(err))
	}

	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatalw("Failed creating user ssh key controller", zap.Error(err))
	}

	users, err := availableUsers()
	if err != nil {
		log.Fatalw("Failed to get users directories", zap.Error(err))
	}
	if err := usersshkeys.Add(mgr, log, users); err != nil {
		log.Fatalw("Failed registering user ssh key controller", zap.Error(err))
	}
	var g run.Group

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("user requested to stop the application")
			case <-done:
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group starts the controller manager
	{
		g.Add(func() error {
			// Start the Cmd
			return mgr.Start(done)
		}, func(err error) {
			log.Infow("stopping user ssh controller", zap.Error(err))
		})
	}

	if err := g.Run(); err != nil {
		log.Fatalw("Failed running user cluster controller", zap.Error(err))
	}
}

func availableUsers() ([]string, error) {
	var paths []string
	for _, user := range []string{"root", "core", "ubuntu", "centos"} {
		path := fmt.Sprintf("%v%v/authorized_keys", resources.AuthorizedKeysPath, user)
		if file, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		} else {
			if file.IsDir() {
				continue
			}

			paths = append(paths, path)
		}
	}

	return paths, nil
}
