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
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

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

type ConvertKubeconfigOptions struct {
	InPlace   bool
	Namespace string
}

func ConvertKubeconfigCommand(logger *logrus.Logger) *cobra.Command {
	opt := ConvertKubeconfigOptions{
		Namespace: metav1.NamespaceSystem,
	}

	cmd := &cobra.Command{
		Use:          "convert-kubeconfig KUBECONFIG",
		Short:        "Convert a kubeconfig to use static credentials for use in Seeds",
		Long:         "Takes a kubeconfig and creates a ServiceAccount with cluster-admin permissions in all clusters, then updates the kubeconfig to use the ServiceAccount's token",
		RunE:         ConvertKubeconfigFunc(logger, &opt),
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVarP(&opt.InPlace, "in-place", "i", false, "update the given kubeconfig file instead of outputting to stdout")
	cmd.PersistentFlags().StringVarP(&opt.Namespace, "namespace", "n", opt.Namespace, "namespace to create ServiceAccount and ClusterRoleBinding in")

	return cmd
}

func ConvertKubeconfigFunc(logger *logrus.Logger, options *ConvertKubeconfigOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		var err error

		if len(args) == 0 {
			return errors.New("no kubeconfig file given")
		}
		filename := args[0]

		kubeconfig, err := readKubeconfig(filename)
		if err != nil {
			return fmt.Errorf("failed to read kubeconfig: %w", err)
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
				return fmt.Errorf("failed to create client config: %w", err)
			}

			token, err := reconcileCluster(context.Background(), clientConfig, options.Namespace, clog)
			if err != nil {
				return fmt.Errorf("failed to reconcile: %w", err)
			}

			if err := updateKubeconfig(kubeconfig, clusterName, contextName, token); err != nil {
				return fmt.Errorf("failed to update kubeconfig: %w", err)
			}

			clog.Info("Done converting cluster")
		}

		output := "-"
		if options.InPlace {
			output = filename
		}

		if err := writeKubeconfig(kubeconfig, output); err != nil {
			return fmt.Errorf("failed to save kubeconfig: %w", err)
		}

		logger.Info("All Done")

		return nil
	})
}

func readKubeconfig(filename string) (*clientcmdapi.Config, error) {
	var err error

	var input *os.File
	if filename == "-" {
		input = os.Stdin
	} else {
		input, err = os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer input.Close()
	}

	content, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	config, err := clientcmd.Load(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
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
		return fmt.Errorf("failed to serialize kubeconfig: %w", err)
	}

	var output *os.File
	if filename == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to open file for writing: %w", err)
		}
		defer output.Close()
	}

	if _, err := output.Write(encoded); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
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
		return "", fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	cache := mgr.GetCache()
	go func() {
		_ = mgr.Start(ctx)
	}()

	if !cache.WaitForCacheSync(ctx) {
		return "", errors.New("failed to start cache")
	}

	client := mgr.GetClient()

	log.Info("Reconciling ServiceAccount...")
	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreatorGetter,
	}, namespace, client); err != nil {
		return "", fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	log.Info("Reconciling ClusterRoleBinding...")
	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleCreatorGetterFactory(namespace),
	}, "", client); err != nil {
		return "", fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	log.Info("Retrieving ServiceAccount token...")
	sa := corev1.ServiceAccount{}
	if err := client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, &sa); err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount: %w", err)
	}

	if len(sa.Secrets) == 0 {
		return "", errors.New("ServiceAccount has no token Secret assigned")
	}

	secretName := sa.Secrets[0].Name
	secret := corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount token Secret: %w", err)
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
