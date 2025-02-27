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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	usersshkeys "k8c.io/kubermatic/v2/pkg/controller/usersshkeysagent"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))
	reconciling.Configure(log)

	cli.Hello(log, "User SSH-Key Agent", nil)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalw("Failed getting user cluster controller config", zap.Error(err))
	}

	ctx := signals.SetupSignalHandler()

	mgr, err := manager.New(cfg, manager.Options{
		BaseContext: func() context.Context {
			return ctx
		},
		Cache: usersshkeys.CacheOptions(),
	})
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

	if err := mgr.Start(ctx); err != nil {
		log.Fatalw("error occurred while running the controller manager", zap.Error(err))
	}
}

func availableUsersPaths() ([]string, error) {
	var paths []string
	users, err := availableHomeUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get users in the home dir: %w", err)
	}

	for _, user := range users {
		path := fmt.Sprintf("/%v", user)
		if user != "root" {
			path = fmt.Sprintf("/home%v", path)
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("failed describing file info: %w", err)
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

	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed describing file info: %w", err)
	}

	if err := os.Mkdir(path, 0700); err != nil {
		return fmt.Errorf("failed creating .ssh dir in %s: %w", path, err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %w", path, err)
	}

	return nil
}

func createFileIfNotExist(path string, uid, gid int) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed describing file info: %w", err)
	}

	if _, err := os.Create(path); err != nil {
		return fmt.Errorf("failed creating authorized_keys file in %s: %w", path, err)
	}

	if err := os.Chmod(path, os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed changing file mode for authorized_keys file in %s: %w", path, err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed changing the numeric uid and gid of %s: %w", path, err)
	}

	return nil
}

func availableHomeUsers() ([]string, error) {
	files, err := os.ReadDir("/home")
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
