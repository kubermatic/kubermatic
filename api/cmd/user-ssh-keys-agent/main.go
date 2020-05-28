package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	cmdutil "github.com/kubermatic/kubermatic/api/cmd/util"
	usersshkeys "github.com/kubermatic/kubermatic/api/pkg/controller/usersshkeysagent"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	cmdutil.Hello(log, "User SSH-Key Agent", logOpts.Debug)

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
	users, err := availableHomeUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get users in the home dir: %v", err)
	}

	for _, user := range users {
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
		if err := createDirIfNotExist(sshPath, int(uid), int(gid)); err != nil {
			return nil, err
		}

		authorizedKeysPath := fmt.Sprintf("%v/authorized_keys", sshPath)
		if err := createFileIfNotExist(authorizedKeysPath, int(uid), int(gid)); err != nil {
			return nil, err
		}

		paths = append(paths, authorizedKeysPath)
	}

	return paths, nil
}

func createDirIfNotExist(path string, uid, gid int) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed describing file info: %v", err)
	}

	if err := os.Mkdir(path, 0700); err != nil {
		return fmt.Errorf("failed creating .ssh dir in %s: %v", path, err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %v", path, err)
	}

	return nil
}

func createFileIfNotExist(path string, uid, gid int) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed describing file info: %v", err)
	}

	if _, err := os.Create(path); err != nil {
		return fmt.Errorf("failed creating authorized_keys file in %s: %v", path, err)
	}

	if err := os.Chmod(path, os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed changing file mode for authorized_keys file in %s: %v", path, err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %v", path, err)
	}

	return nil
}

func availableHomeUsers() ([]string, error) {
	files, err := ioutil.ReadDir("/home")
	if err != nil {
		return nil, err
	}

	var users = []string{"root"}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		users = append(users, file.Name())
	}

	return users, nil
}
