package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	usersshkeys "github.com/kubermatic/kubermatic/api/pkg/controller/usersshkeysagent"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerRunOptions struct {
	log kubermaticlog.Options
}

func main() {
	runOp := controllerRunOptions{}
	flag.BoolVar(&runOp.log.Debug, "log-debug", true, "Enables debug logging")
	flag.StringVar(&runOp.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())

	flag.Parse()

	if err := runOp.log.Validate(); err != nil {
		fmt.Printf("error occurred while validating zap logger options: %v\n", err)
		os.Exit(1)
	}

	rawLog := kubermaticlog.New(runOp.log.Debug, kubermaticlog.Format(runOp.log.Format))
	log := rawLog.Sugar()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalw("Failed getting user cluster controller config", zap.Error(err))
	}

	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	mgr, err := manager.New(cfg, manager.Options{Namespace: metav1.NamespaceSystem})
	if err != nil {
		log.Fatalw("Failed creating user ssh key controller", zap.Error(err))
	}

	paths, err := availableUsersPaths()
	if err != nil {
		log.Fatalw("Failed to get users directories", zap.Error(err))
	}
	if err := usersshkeys.Add(mgr, log, paths); err != nil {
		log.Fatalw("Failed registering user ssh key controller", zap.Error(err))
	}

	if err := mgr.Start(done); err != nil {
		log.Fatalw("error occurred while running the controller manager", zap.Error(err))
	}
}

func availableUsersPaths() ([]string, error) {
	var paths []string
	for _, user := range []string{"root", "core", "ubuntu", "centos"} {
		path := fmt.Sprintf("/%v", user)
		if user != "root" {
			path = fmt.Sprintf("/home%v", path)
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("failed describing file info: %v", err)
		}

		uid := fileInfo.Sys().(*syscall.Stat_t).Uid
		gid := fileInfo.Sys().(*syscall.Stat_t).Gid

		sshPath := fmt.Sprintf("%v/.ssh", path)
		if err := createIfNotExist(sshPath, int(uid), int(gid)); err != nil {
			return nil, err
		}

		authorizedKeysPath := fmt.Sprintf("%v/authorized_keys", sshPath)
		if err := createIfNotExist(authorizedKeysPath, int(uid), int(gid)); err != nil {
			return nil, err
		}

		paths = append(paths, authorizedKeysPath)
	}

	return paths, nil
}

func createIfNotExist(path string, uid, gid int) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0600); err != nil {
				return fmt.Errorf("failed creating .ssh dir in %s: %v", path, err)
			}

			if err := os.Chown(path, int(uid), int(gid)); err != nil {
				return fmt.Errorf("failed changing the numeric uid and gid of %s: %v", path, err)
			}
		}

		return fmt.Errorf("failed describing file info: %v", err)
	}

	return nil
}
