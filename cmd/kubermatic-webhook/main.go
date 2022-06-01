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
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	addonmutation "k8c.io/kubermatic/v2/pkg/webhook/addon/mutation"
	applicationdefinitionvalidation "k8c.io/kubermatic/v2/pkg/webhook/application/applicationdefinition/validation"
	clustermutation "k8c.io/kubermatic/v2/pkg/webhook/cluster/mutation"
	clustervalidation "k8c.io/kubermatic/v2/pkg/webhook/cluster/validation"
	kubermaticconfigurationvalidation "k8c.io/kubermatic/v2/pkg/webhook/kubermaticconfiguration/validation"
	mlaadminsettingmutation "k8c.io/kubermatic/v2/pkg/webhook/mlaadminsetting/mutation"
	oscvalidation "k8c.io/kubermatic/v2/pkg/webhook/operatingsystemmanager/operatingsystemconfig/validation"
	ospvalidation "k8c.io/kubermatic/v2/pkg/webhook/operatingsystemmanager/operatingsystemprofile/validation"
	seedwebhook "k8c.io/kubermatic/v2/pkg/webhook/seed"
	uservalidation "k8c.io/kubermatic/v2/pkg/webhook/user/validation"
	usersshkeymutation "k8c.io/kubermatic/v2/pkg/webhook/usersshkey/mutation"
	usersshkeyvalidation "k8c.io/kubermatic/v2/pkg/webhook/usersshkey/validation"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	rootCtx := context.Background()

	// /////////////////////////////////////////
	// setup flags

	options, err := initApplicationOptions()
	if err != nil {
		fmt.Printf("Invalid flags: %v\n", err)
		os.Exit(1)
	}

	// /////////////////////////////////////////
	// setup logging

	rawLog := kubermaticlog.New(options.log.Debug, options.log.Format)
	log := rawLog.Sugar()

	// set the logger used by controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog))

	// say hello
	versions := kubermatic.NewDefaultVersions()
	cli.Hello(log, "Webhook", options.log.Debug, &versions)

	// /////////////////////////////////////////
	// get kubeconfig

	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get kubeconfig", zap.Error(err))
	}

	// /////////////////////////////////////////
	// create manager

	mgr, err := manager.New(cfg, manager.Options{
		Namespace: options.namespace,
	})
	if err != nil {
		log.Fatalw("Failed to create the manager", zap.Error(err))
	}

	// apply the CLI flags for configuring the  webhook server to the manager
	if err := options.webhook.Configure(mgr.GetWebhookServer()); err != nil {
		log.Fatalw("Failed to configure webhook server", zap.Error(err))
	}

	// add APIs we use
	addAPIs(mgr.GetScheme(), log)

	// create config and seed getters; note that if the webhook runs on a pure Seed
	// cluster, the seedsGetter will only ever see 1 Seed resource; this is fine, as
	// Seeds on seed clusters are managed by the KKP operator and admins should only
	// ever manage Seeds on the master cluster anyway, i.e. all changes to Seeds on
	// seed clusters were already validated on the master cluster
	configGetter, seedGetter, seedsGetter, seedClientGetter := createGetters(rootCtx, log, mgr, &options)

	caPool := options.caBundle.CertPool()

	// /////////////////////////////////////////
	// add pprof runnable, which will start a websever if configured

	if err := mgr.Add(&options.pprof); err != nil {
		log.Fatalw("Failed to add the pprof handler", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup Seed webhook

	seedValidator, err := seedwebhook.NewValidator(seedsGetter, seedClientGetter, options.featureGates)
	if err != nil {
		log.Fatalw("Failed to create seed validator", zap.Error(err))
	}

	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.Seed{}).WithValidator(seedValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup seed validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup KubermaticConfiguration webhooks

	configValidator := kubermaticconfigurationvalidation.NewValidator()
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.KubermaticConfiguration{}).WithValidator(configValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup KubermaticConfiguration validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup Cluster webhooks

	// validation webhook can already use ctrl-runtime boilerplate
	clusterValidator := clustervalidation.NewValidator(mgr.GetClient(), seedGetter, configGetter, options.featureGates, caPool)
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.Cluster{}).WithValidator(clusterValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup cluster validation webhook", zap.Error(err))
	}

	// mutation cannot, because we require separate defaulting for CREATE and UPDATE operations
	clustermutation.NewAdmissionHandler(mgr.GetClient(), configGetter, seedGetter, caPool).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup Addon webhook

	addonmutation.NewAdmissionHandler(seedGetter, seedClientGetter).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup MLAAdminSetting webhooks

	mlaadminsettingmutation.NewAdmissionHandler(seedGetter, seedClientGetter).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup User webhooks

	userValidator := uservalidation.NewValidator(mgr.GetClient())
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.User{}).WithValidator(userValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup user validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup UserSSHKey webhooks

	usersshkeymutation.NewAdmissionHandler(mgr.GetClient()).SetupWebhookWithManager(mgr)

	userSSHKeyValidator := usersshkeyvalidation.NewValidator(mgr.GetClient())
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.UserSSHKey{}).WithValidator(userSSHKeyValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup user SSH key validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup OSM webhooks

	// Setup the validation admission handler for OperatingSystemConfig CRDs
	oscvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// Setup the validation admission handler for OperatingSystemProfile CRDs
	ospvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// Setup the validation admission handler for ApplicationDefinition CRDs
	applicationdefinitionvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// Here we go!

	log.Info("Starting the webhook...")
	if err := mgr.Start(ctrlruntime.SetupSignalHandler()); err != nil {
		log.Fatalw("The controller manager has failed", zap.Error(err))
	}
}

func addAPIs(dst *runtime.Scheme, log *zap.SugaredLogger) {
	if err := kubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := clusterv1alpha1.AddToScheme(dst); err != nil {
		log.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := osmv1alpha1.AddToScheme(dst); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", osmv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := appskubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", appskubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
}

func createGetters(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, options *appOptions) (
	configGetter provider.KubermaticConfigurationGetter,
	seedGetter provider.SeedGetter,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
) {
	client := mgr.GetClient()

	var err error

	if options.kubermaticConfiguration != nil {
		configGetter, err = provider.StaticKubermaticConfigurationGetterFactory(options.kubermaticConfiguration)
	} else {
		configGetter, err = provider.DynamicKubermaticConfigurationGetterFactory(client, options.namespace)
	}
	if err != nil {
		log.Fatalw("Unable to create the configuration getter", zap.Error(err))
	}

	// if master and seed clusters are shared, the webhook can be configured
	// with a seed name; if no name is configured, the seed getter will simply
	// return nil.
	// The kubermatic-operator takes care of setting the -seed-name flag properly.
	if options.seedName != "" {
		seedGetter, err = seedGetterFactory(ctx, client, options)
	} else {
		seedGetter = func() (*kubermaticv1.Seed, error) {
			return nil, nil
		}
	}
	if err != nil {
		log.Fatalw("Unable to create the seed getter", zap.Error(err))
	}

	seedsGetter, err = seedsGetterFactory(ctx, client, options.namespace)
	if err != nil {
		log.Fatalw("Failed to create seeds getter", zap.Error(err))
	}

	seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(ctx, client)
	if err != nil {
		log.Fatalw("Failed to create seed kubeconfig getter", zap.Error(err))
	}

	seedClientGetter = provider.SeedClientGetterFactory(seedKubeconfigGetter)

	return
}
