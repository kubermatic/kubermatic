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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
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
	if project != "" {
		log.Infow("-project given, will not create a new project", "project", project)
	}

	return &ProjectJig{
		client:            client,
		log:               log,
		humanReadableName: "e2e test project",
		projectName:       project,
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
	if j.projectName == "" {
		return nil, errors.New("no project created yet")
	}

	project := &kubermaticv1.Project{}
	if err := j.client.Get(ctx, types.NamespacedName{Name: j.projectName}, project); err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

func (j *ProjectJig) Create(ctx context.Context, waitForActive bool) (*kubermaticv1.Project, error) {
	if j.projectName != "" {
		return j.Project(ctx)
	}

	projectName := rand.String(10)

	j.log.Infow("Creating project...", "humanname", j.humanReadableName)
	project := &kubermaticv1.Project{}
	project.Name = projectName
	project.Spec.Name = j.humanReadableName

	if err := j.client.Create(ctx, project); err != nil {
		return nil, err
	}

	log := j.log.With("project", project.Name)

	log.Info("Project created successfully.")
	j.projectName = project.Name

	if waitForActive {
		log.Info("Waiting for project to become active...")

		err := wait.PollLog(ctx, j.log, 2*time.Second, 30*time.Second, func(ctx context.Context) (transient error, terminal error) {
			p := &kubermaticv1.Project{}
			if err := j.client.Get(ctx, types.NamespacedName{Name: projectName}, p); err != nil {
				return err, nil
			}

			if p.Status.Phase != kubermaticv1.ProjectActive {
				return errors.New("project is not active"), nil
			}

			project = p

			return nil, nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to wait for project to become active: %w", err)
		}
	}

	return project, nil
}

func (j *ProjectJig) Delete(ctx context.Context, synchronous bool) error {
	if j.projectName == "" {
		return nil
	}

	log := j.log.With("project", j.projectName)
	log.Info("Deleting project...")

	project := &kubermaticv1.Project{}
	project.Name = j.projectName

	if err := j.client.Delete(ctx, project); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if synchronous {
		log.Info("Waiting for project to be gone...")

		err := wait.PollLog(ctx, log, 5*time.Second, 30*time.Minute, func(ctx context.Context) (transient error, terminal error) {
			project := &kubermaticv1.Project{}
			err := j.client.Get(ctx, types.NamespacedName{Name: j.projectName}, project)
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
