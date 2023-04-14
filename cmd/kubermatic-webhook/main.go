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
	appskubermaticv1 "k8c.io/api/v3/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources/reconciling"
	"k8c.io/kubermatic/v3/pkg/util/cli"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"
	addonmutation "k8c.io/kubermatic/v3/pkg/webhook/addon/mutation"
	applicationdefinitionmutation "k8c.io/kubermatic/v3/pkg/webhook/application/applicationdefinition/mutation"
	applicationdefinitionvalidation "k8c.io/kubermatic/v3/pkg/webhook/application/applicationdefinition/validation"
	clustermutation "k8c.io/kubermatic/v3/pkg/webhook/cluster/mutation"
	clustervalidation "k8c.io/kubermatic/v3/pkg/webhook/cluster/validation"
	clustertemplatevalidation "k8c.io/kubermatic/v3/pkg/webhook/clustertemplate/validation"
	datacenterwebhook "k8c.io/kubermatic/v3/pkg/webhook/datacenter"
	ipampoolvalidation "k8c.io/kubermatic/v3/pkg/webhook/ipampool/validation"
	kubermaticconfigurationvalidation "k8c.io/kubermatic/v3/pkg/webhook/kubermaticconfiguration/validation"
	mlaadminsettingmutation "k8c.io/kubermatic/v3/pkg/webhook/mlaadminsetting/mutation"
	uservalidation "k8c.io/kubermatic/v3/pkg/webhook/user/validation"
	usersshkeymutation "k8c.io/kubermatic/v3/pkg/webhook/usersshkey/mutation"
	usersshkeyvalidation "k8c.io/kubermatic/v3/pkg/webhook/usersshkey/validation"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	rootCtx := signals.SetupSignalHandler()

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
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))
	reconciling.Configure(log)

	// say hello
	versions := kubermatic.NewDefaultVersions(edition.CommunityEdition)
	cli.Hello(log, "Webhook", options.log.Debug, versions)

	// /////////////////////////////////////////
	// get kubeconfig

	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get kubeconfig", zap.Error(err))
	}

	// /////////////////////////////////////////
	// create manager

	mgr, err := manager.New(cfg, manager.Options{
		BaseContext: func() context.Context {
			return rootCtx
		},
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

	// create config getter
	configGetter, err := createConfigGetter(mgr, &options)
	if err != nil {
		log.Fatalw("Unable to create the configuration getter", zap.Error(err))
	}

	datacenterGetter, err := kubernetes.DatacenterGetterFactory(mgr.GetClient())
	if err != nil {
		log.Fatalw("Unable to create the datacenter getter", zap.Error(err))
	}

	datacentersGetter, err := kubernetes.DatacentersGetterFactory(mgr.GetClient())
	if err != nil {
		log.Fatalw("Unable to create the datacenters getter", zap.Error(err))
	}

	caPool := options.caBundle.CertPool()

	// /////////////////////////////////////////
	// add pprof runnable, which will start a websever if configured

	if err := mgr.Add(&options.pprof); err != nil {
		log.Fatalw("Failed to add the pprof handler", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup Datacenter webhook

	datacenterValidator := datacenterwebhook.NewValidator(mgr.GetClient(), datacentersGetter)
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.Datacenter{}).WithValidator(datacenterValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup datacenter validation webhook", zap.Error(err))
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
	clusterValidator := clustervalidation.NewValidator(mgr.GetClient(), configGetter, datacenterGetter, options.featureGates, caPool)
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.Cluster{}).WithValidator(clusterValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup cluster validation webhook", zap.Error(err))
	}

	// mutation cannot, because we require separate defaulting for CREATE and UPDATE operations
	clustermutation.NewAdmissionHandler(mgr.GetClient(), configGetter, datacenterGetter, caPool).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup ClusterTemplate webhooks

	clusterTemplateValidator := clustertemplatevalidation.NewValidator(mgr.GetClient(), configGetter, datacenterGetter, options.featureGates, caPool)
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.ClusterTemplate{}).WithValidator(clusterTemplateValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup cluster validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup Addon webhook

	addonmutation.NewAdmissionHandler(mgr.GetClient()).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup MLAAdminSetting webhooks

	mlaadminsettingmutation.NewAdmissionHandler(mgr.GetClient()).SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup User webhooks

	userValidator := uservalidation.NewValidator()
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.User{}).WithValidator(userValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup user validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup UserSSHKey webhooks

	usersshkeymutation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	userSSHKeyValidator := usersshkeyvalidation.NewValidator()
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.UserSSHKey{}).WithValidator(userSSHKeyValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup user SSH key validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// setup ApplicationDefinition webhook

	// Setup the mutation admission handler for ApplicationDefinition CRDs
	applicationdefinitionmutation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// Setup the validation admission handler for ApplicationDefinition CRDs
	applicationdefinitionvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// /////////////////////////////////////////
	// setup IPAMPool webhook

	ipamPoolValidator := ipampoolvalidation.NewValidator(mgr.GetClient())
	if err := builder.WebhookManagedBy(mgr).For(&kubermaticv1.IPAMPool{}).WithValidator(ipamPoolValidator).Complete(); err != nil {
		log.Fatalw("Failed to setup IPAMPool validation webhook", zap.Error(err))
	}

	// /////////////////////////////////////////
	// Here we go!

	log.Info("Starting the webhook...")
	if err := mgr.Start(rootCtx); err != nil {
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
	if err := appskubermaticv1.AddToScheme(dst); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", appskubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := apiextensionsv1.AddToScheme(dst); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1.SchemeGroupVersion), zap.Error(err))
	}
}

func createConfigGetter(mgr manager.Manager, options *appOptions) (provider.KubermaticConfigurationGetter, error) {
	if options.kubermaticConfiguration != nil {
		return kubernetes.StaticKubermaticConfigurationGetterFactory(options.kubermaticConfiguration)
	}

	return kubernetes.DynamicKubermaticConfigurationGetterFactory(mgr.GetClient(), options.namespace)
}
