/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	seedFlag = cli.BoolFlag{
		Name:  "seed",
		Usage: "Print Seed defaults",
	}

	kubermaticConfigFlag = cli.BoolFlag{
		Name:  "config",
		Usage: "Print KubermaticConfiguration defaults",
	}
)

func PrintCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:   "print",
		Usage:  "Prints default values for CRDs",
		Action: PrintAction(logger),
		Flags: []cli.Flag{
			kubermaticConfigFlag,
			seedFlag,
		},
	}
}

func PrintAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {

		nopLogger := zap.NewNop().Sugar()
		var defaulted interface{}
		var err error

		if ctx.Bool(seedFlag.Name) {

			config := &kubermaticv1.Seed{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubermatic.k8s.io/v1",
					Kind:       "Seed",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
			}

			defaulted, err = common.DefaultSeed(config, nopLogger)

			if err != nil {
				return cli.NewExitError(fmt.Errorf("failed to create default Seed: %v", err), 1)
			}
		}

		if ctx.Bool(kubermaticConfigFlag.Name) || defaulted == nil {

			config := &operatorv1alpha1.KubermaticConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "operator.kubermatic.io/v1alpha1",
					Kind:       "KubermaticConfiguration",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
			}

			defaulted, err = common.DefaultConfiguration(config, nopLogger)

			if err != nil {
				return cli.NewExitError(fmt.Errorf("failed to create default KubermaticConfiguration: %v", err), 1)
			}

		}

		if err = yamlutil.Encode(defaulted, os.Stdout); err != nil {
			return cli.NewExitError(fmt.Errorf("failed to create YAML: %v", err), 1)
		}

		return nil
	}))
}
