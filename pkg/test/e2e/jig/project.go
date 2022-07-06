/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package jig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectJig struct {
	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger

	// user-controller parameters
	humanReadableName string

	// data about the generated project
	projectName string
}

func NewProjectJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger) *ProjectJig {
	return &ProjectJig{
		client: client,
		log:    log,
	}
}

func (j *ProjectJig) WithHumanReadableName(name string) *ProjectJig {
	j.humanReadableName = name
	return j
}

func (j *ProjectJig) ProjectName() string {
	return j.projectName
}

func (j *ProjectJig) Project(ctx context.Context) (*kubermaticv1.Project, error) {
	projectProvider, err := kubernetes.NewPrivilegedProjectProvider(j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create project provider: %w", err)
	}

	return projectProvider.GetUnsecured(ctx, j.projectName, &provider.ProjectGetOptions{
		IncludeUninitialized: true,
	})
}

func (j *ProjectJig) Create(ctx context.Context, waitForActive bool) (*kubermaticv1.Project, error) {
	projectProvider, err := kubernetes.NewProjectProvider(nil, j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create project provider: %w", err)
	}

	j.log.Info("Creating project...", "humanname", j.humanReadableName)
	project, err := projectProvider.New(ctx, j.projectName, nil)
	if err != nil {
		return nil, err
	}

	if waitForActive {
		j.log.Info("Waiting for project to become active...")

		projectProvider, err := kubernetes.NewPrivilegedProjectProvider(j.client)
		if err != nil {
			return nil, fmt.Errorf("failed to create project provider: %w", err)
		}

		err = wait.PollLog(j.log, 2*time.Second, 30*time.Second, func() (transient error, terminal error) {
			project, err = projectProvider.GetUnsecured(ctx, j.projectName, &provider.ProjectGetOptions{
				IncludeUninitialized: true,
			})

			if err != nil {
				return err, nil
			}

			if project.Status.Phase != kubermaticv1.ProjectActive {
				return errors.New("project is not active"), nil
			}

			return nil, nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to wait for project to become active: %w", err)
		}
	}

	j.log.Info("Project created successfully.", "name", project.Name)
	j.projectName = project.Name

	return project, nil
}

func (j *ProjectJig) Delete(ctx context.Context, synchronous bool) error {
	if j.projectName == "" {
		return errors.New("no project created or already deleted.")
	}

	log := j.log.With("project", j.projectName)
	log.Info("Deleting project...")

	projectProvider, err := kubernetes.NewPrivilegedProjectProvider(j.client)
	if err != nil {
		return fmt.Errorf("failed to create project provider: %w", err)
	}

	if err := projectProvider.DeleteUnsecured(ctx, j.projectName); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if synchronous {
		log.Info("Waiting for project to be gone...")

		err = wait.PollLog(log, 5*time.Second, 10*time.Minute, func() (transient error, terminal error) {
			_, err := projectProvider.GetUnsecured(ctx, j.projectName, &provider.ProjectGetOptions{
				IncludeUninitialized: true,
			})

			if err == nil {
				return errors.New("project still exists"), nil
			}
			if !apierrors.IsNotFound(err) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil {
			return fmt.Errorf("failed to wait for project to be gone: %w", err)
		}
	}

	j.projectName = ""

	return nil
}
