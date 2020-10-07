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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	serviceAccountName = "kubermatic"
)

var (
	inPlaceConversionFlag = cli.BoolFlag{
		Name:  "in-place",
		Usage: "Update the given kubeconfig file instead of outputting to stdout",
	}
	namespaceFlag = cli.StringFlag{
		Name:  "namespace",
		Value: metav1.NamespaceSystem,
		Usage: "Namespace to create ServiceAccount and ClusterRoleBinding in",
	}
)

func ConvertKubeconfigCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:      "convert-kubeconfig",
		Usage:     "Takes a kubeconfig and creates a ServiceAccount with cluster-admin permissions in all clusters, then updates the kubeconfig to use the ServiceAccount's token",
		Action:    ConvertKubeconfigAction(logger),
		ArgsUsage: "KUBECONFIG",
		Flags: []cli.Flag{
			inPlaceConversionFlag,
			namespaceFlag,
		},
	}
}

func ConvertKubeconfigAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		var err error

		filename := ctx.Args().First()
		if filename == "" {
			return errors.New("no kubeconfig file given")
		}

		kubeconfig, err := readKubeconfig(filename)
		if err != nil {
			return fmt.Errorf("failed to read kubeconfig: %v", err)
		}

		namespace := ctx.String(namespaceFlag.Name)
		if namespace == "" {
			namespace = metav1.NamespaceSystem
		}

		for clusterName := range kubeconfig.Clusters {
			clog := logger.WithField("cluster", clusterName)

			// find the first context for this cluster
			contextName := ""
			for ctxName, ctx := range kubeconfig.Contexts {
				if ctx.Cluster == clusterName {
					contextName = ctxName
				}
			}

			if contextName == "" {
				clog.Warn("No context found for cluster, skipping.")
				continue
			}

			clog = clog.WithField("context", contextName)

			clog.Info("Converting cluster")

			clientConfig, err := clientcmd.NewInteractiveClientConfig(*kubeconfig, contextName, nil, nil, nil).ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to create client config: %v", err)
			}

			token, err := reconcileCluster(context.Background(), clientConfig, namespace, clog)
			if err != nil {
				return fmt.Errorf("failed to reconcile: %v", err)
			}

			if err := updateKubeconfig(kubeconfig, clusterName, contextName, token); err != nil {
				return fmt.Errorf("failed to update kubeconfig: %v", err)
			}

			clog.Info("Done converting cluster")
		}

		output := "-"
		if ctx.Bool(inPlaceConversionFlag.Name) {
			output = filename
		}

		if err := writeKubeconfig(kubeconfig, output); err != nil {
			return fmt.Errorf("failed to save kubeconfig: %v", err)
		}

		logger.Info("All Done")

		return nil
	}))
}

func readKubeconfig(filename string) (*clientcmdapi.Config, error) {
	var err error

	var input *os.File
	if filename == "-" {
		input = os.Stdin
	} else {
		input, err = os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %v", err)
		}
		defer input.Close()
	}

	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %v", err)
	}

	config, err := clientcmd.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %v", err)
	}

	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*clientcmdapi.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*clientcmdapi.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*clientcmdapi.Context{}
	}

	return config, nil
}

func writeKubeconfig(kubeconfig *clientcmdapi.Config, filename string) error {
	encoded, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to serialize kubeconfig: %v", err)
	}

	var output *os.File
	if filename == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to open file for writing: %v", err)
		}
		defer output.Close()
	}

	if _, err := output.Write(encoded); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}

func serviceAccountCreatorGetter() (string, reconciling.ServiceAccountCreator) {
	return serviceAccountName, func(existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return existing, nil
	}
}

func clusterRoleCreatorGetterFactory(namespace string) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return "kubermatic:cluster-admin", func(existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			existing.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: namespace,
				},
			}

			existing.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}

			return existing, nil
		}
	}
}

func reconcileCluster(ctx context.Context, config *rest.Config, namespace string, log logrus.FieldLogger) (string, error) {
	mgr, err := manager.New(config, manager.Options{
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cache := mgr.GetCache()
	go func() {
		_ = mgr.Start(mgrCtx.Done())
	}()

	if !cache.WaitForCacheSync(mgrCtx.Done()) {
		return "", errors.New("failed to start cache")
	}

	client := mgr.GetClient()

	log.Info("Reconciling ServiceAccount...")
	if err := reconciling.ReconcileServiceAccounts(mgrCtx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreatorGetter,
	}, namespace, client); err != nil {
		return "", fmt.Errorf("failed to create ServiceAccount: %v", err)
	}

	log.Info("Reconciling ClusterRoleBinding...")
	if err := reconciling.ReconcileClusterRoleBindings(mgrCtx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleCreatorGetterFactory(namespace),
	}, "", client); err != nil {
		return "", fmt.Errorf("failed to create ClusterRoleBinding: %v", err)
	}

	log.Info("Retrieving ServiceAccount token...")
	sa := corev1.ServiceAccount{}
	if err := client.Get(mgrCtx, types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, &sa); err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount: %v", err)
	}

	if len(sa.Secrets) == 0 {
		return "", errors.New("ServiceAccount has no token Secret assigned")
	}

	secretName := sa.Secrets[0].Name
	secret := corev1.Secret{}
	if err := client.Get(mgrCtx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount token Secret: %v", err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", fmt.Errorf("ServiceAccount token Secret %q does not contain token", secretName)
	}

	return string(token), nil
}

func updateKubeconfig(kubeconfig *clientcmdapi.Config, clusterName string, contextName string, token string) error {
	authInfoName := fmt.Sprintf("%s-kubermatic-service-account", clusterName)

	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.Token = token

	kubeconfig.AuthInfos[authInfoName] = authInfo

	for ctxName, context := range kubeconfig.Contexts {
		if ctxName == contextName {
			context.AuthInfo = authInfoName
		}
	}

	return nil
}
