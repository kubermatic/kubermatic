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

package rancher

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"

	rancherclient "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/rancher/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName     = "kubernatic_rancher_controller"
	RancherUsername    = "admin"
	RancherAdminSecret = "rancher-admin-secret"
	// keep the linter happy
	// trueStr                   = "true"
	rancherRandPasswordLength = 16
)

// KubeconfigProvider provides functionality to get a clusters admin kubeconfig
type KubeconfigProvider interface {
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
}

type fileHandlingDone func()

type Reconciler struct {
	ctrlruntimeclient.Client

	log                *zap.SugaredLogger
	kubeconfigProvider KubeconfigProvider
	versions           kubermatic.Versions
}

var (
	rancherClient *rancherclient.Client
)

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	kubeconfigProvider KubeconfigProvider,
	versions kubermatic.Versions,
) error {

	log = log.Named(ControllerName)
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		log:                log,
		kubeconfigProvider: kubeconfigProvider,
		versions:           versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler: reconciler,
	})

	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}
	predicates := predicateutil.ByName(resources.RancherStatefulSetName)

	if err := c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForObject{}, predicates); err != nil {
		return fmt.Errorf("failed to create watch for statefulsets: %v", err)
	}
	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := r.log.With("request", request)
	log.Debug("Processing")

	statefulSet := &appsv1.StatefulSet{}
	if err := r.Get(ctx, request.NamespacedName, statefulSet); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debugw("couldn't find statefulSet", zap.Error(err))
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get rancher statefulset: %v", err)
	}

	result, err := r.reconcile(ctx, log, statefulSet)
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		clusterName := strings.ReplaceAll(statefulSet.Namespace, "cluster-", "")
		log.Errorf("failed to reconcile %s: %v", clusterName, zap.Error(err))
		return *result, fmt.Errorf("failed to reconcile %s: %v", clusterName, err)
	}
	return *result, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, statefulSet *appsv1.StatefulSet) (*reconcile.Result, error) {
	if statefulSet.DeletionTimestamp != nil {
		return nil, nil
	}
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: strings.ReplaceAll(statefulSet.Namespace, "cluster-", "")}, cluster); err != nil {
		log.Debugw("can't find cluster", zap.Error(err))
		return nil, nil
	}
	// rancher integration is disabled
	if flag := cluster.Spec.Features[kubermaticv1.ClusterFeatureRancherIntegration]; !flag {
		return nil, nil
	}
	// cluster is already imported
	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionRancherInitialized, corev1.ConditionTrue) &&
		cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionRancherClusterImported, corev1.ConditionTrue) {
		return nil, nil
	}
	// wait for statefulSet to be ready
	if statefulSet.Status.ReadyReplicas == 0 {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if statefulSet.Annotations == nil {
		statefulSet.Annotations = make(map[string]string)
	}
	// initialize rancher statefulSet
	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionRancherInitialized, corev1.ConditionTrue) {
		if err := r.initRancherServer(ctx, log, statefulSet); err != nil {
			log.Errorw("failed to initialize Rancher Server", zap.Error(err))
			return &reconcile.Result{}, err
		}
		kubermaticv1helper.SetClusterCondition(cluster, r.versions, kubermaticv1.ClusterConditionRancherInitialized, corev1.ConditionTrue, "", "Rancher server initialized successfully")
		if err := r.Update(ctx, cluster); err != nil {
			return &reconcile.Result{}, err
		}
		log.Infow("rancher server statefulSet initialized successfully")

	}
	// Setup the user cluster
	rancherRegToken, err := r.rancherClusterWithRegistrationToken(ctx, cluster.Status.NamespaceName, cluster.Spec.HumanReadableName)
	if err != nil {
		log.Errorw("failed to create rancher cluster:", zap.Error(err))
		return nil, err
	}
	if rancherRegToken != nil {
		if err := r.applyRancherRegstrationCommand(ctx, log, cluster, rancherRegToken); err != nil {
			log.Errorw("failed to apply rancher regstration command", zap.Error(err))
			return nil, err
		}
		kubermaticv1helper.SetClusterCondition(cluster, r.versions, kubermaticv1.ClusterConditionRancherClusterImported, corev1.ConditionTrue, "", "Rancher cluster imported successfully")
		if err := r.Update(ctx, cluster); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r *Reconciler) initRancherServer(ctx context.Context, log *zap.SugaredLogger, statefulSet *appsv1.StatefulSet) error {
	client, err := r.getRancherClient(ctx, statefulSet.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get rancher client: %v", err)
	}
	users, err := client.ListUsers(map[string]string{"username": RancherUsername})
	if err != nil {
		return fmt.Errorf("failed to get user list: %v", err)
	}
	initSecret, err := r.getRancherInitSecret(ctx, statefulSet.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get rancher admin secret [%s] : %v", RancherAdminSecret, err)
	}
	for _, user := range users.Data {
		if user.Username == RancherUsername {
			err = client.SetUserPassword(&user, &rancherclient.SetPasswordInput{NewPassword: string(initSecret.Data["password"])})
			if err != nil {
				return fmt.Errorf("failed to set rancher user password: %v", err)
			}
			break
		}
	}
	return nil
}

func (r *Reconciler) rancherClusterWithRegistrationToken(ctx context.Context, clusterNamespace, clusterName string) (*rancherclient.ClusterRegistrationToken, error) {
	client, err := r.getRancherClient(ctx, clusterNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get rancher client: %v", err)
	}
	clusterList, err := client.ListClusters(map[string]string{"name": clusterName})
	if err != nil {
		return nil, fmt.Errorf("failed to list rancher clusters: %v", err)
	}
	rancherCluster := &rancherclient.Cluster{Name: clusterName}
	if len(clusterList.Data) != 0 {
		rancherCluster = &clusterList.Data[0]
	} else {
		rancherCluster, err = client.CreateImportedCluster(rancherCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create rancher imported cluster: %v", err)
		}
	}
	if !isRancherClusterProvisioned(rancherCluster) {
		token := &rancherclient.ClusterRegistrationToken{ClusterID: rancherCluster.ID}
		token, err = client.CreateClusterRegistrationToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to create rancher cluster registration token: %v", err)
		}
		return token, nil
	}

	return nil, nil
}

func (r *Reconciler) getRancherClient(ctx context.Context, namespace string) (*rancherclient.Client, error) {
	if rancherClient != nil {
		return rancherClient, nil
	}
	address, err := r.getRancherServerURL(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("can't get rancher service URL: %v", err)
	}
	initSecret, err := r.getRancherInitSecret(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get rancher secret: %v", err)
	}
	opts := rancherclient.Options{
		Endpoint:  address,
		AccessKey: RancherUsername,
		SecretKey: string(initSecret.Data["password"]),
		Insecure:  true,
	}
	rancherClient, err = rancherclient.New(opts)
	if err != nil {
		r.log.Debugw("failed to login updated credentials:", zap.Error(err))
		// fall back to the default password
		opts.SecretKey = "admin"
		rancherClient, err = rancherclient.New(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to logging to rancher server: %v", err)
		}
	}
	return rancherClient, nil
}

func (r *Reconciler) getRancherServerURL(ctx context.Context, namespace string) (string, error) {
	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: resources.RancherServerServiceName, Namespace: namespace}, service); err != nil {
		return "", err
	}
	var port int32
	for _, svcPort := range service.Spec.Ports {
		if svcPort.Name == "https" {
			port = svcPort.NodePort
			break
		}
	}
	if port == 0 {
		return "", fmt.Errorf("Can't find rancher server service nodeport")
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: strings.ReplaceAll(service.Namespace, "cluster-", "")}, cluster); err != nil {
		return "", fmt.Errorf("failed to get cluster: %v", err)
	}

	return fmt.Sprintf("https://%s:%d", cluster.Address.ExternalName, port), nil
}

func (r *Reconciler) getRancherInitSecret(ctx context.Context, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: RancherAdminSecret, Namespace: namespace}, secret)
	if err != nil && !kubeapierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get secret: %v", err)
	}
	if secret.Name != "" {
		return secret, nil
	}
	secret.Name = RancherAdminSecret
	secret.Namespace = namespace
	secret.Data = map[string][]byte{
		"password": []byte(randString()),
		"user":     []byte("admin"),
	}
	err = r.Create(ctx, secret)
	return secret, err
}

func (r *Reconciler) applyRancherRegstrationCommand(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, regToken *rancherclient.ClusterRegistrationToken) error {
	client := getHTTPClient(true)
	resp, err := client.Get(regToken.ManifestURL)
	if err != nil {
		return fmt.Errorf("failed to get HTTP client: %v", err)
	}
	defer resp.Body.Close()
	manifest, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read http response: %v", err)
	}
	kubeconfigFilename, kubeconfigDone, err := r.writeAdminKubeconfig(log, cluster)
	if err != nil {
		return fmt.Errorf("failed to write the admin kubeconfig to the local filesystem: %v", err)
	}
	defer kubeconfigDone()
	cmd := getApplyCommand(ctx, kubeconfigFilename)
	buffer := bytes.Buffer{}

	buffer.Write(manifest)
	cmd.Stdin = &buffer

	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run rancher kubectl apply command: %v", err)
	}
	return nil
}

func (r *Reconciler) writeAdminKubeconfig(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := r.kubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	kubeconfigFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-rancher-kubeconfig", cluster.Name))
	if err := ioutil.WriteFile(kubeconfigFilename, kubeconfig, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	log.Debugw("Wrote admin kubeconfig", "file", kubeconfigFilename)

	return kubeconfigFilename, getFileDeleteFinalizer(log, kubeconfigFilename), nil
}

func getFileDeleteFinalizer(log *zap.SugaredLogger, filename string) fileHandlingDone {
	return func() {
		if err := os.RemoveAll(filename); err != nil {
			log.Errorw("Failed to delete file", zap.Error(err), "file", filename)
		}
	}
}

func getApplyCommand(ctx context.Context, kubeconfigFilename string) *exec.Cmd {
	return exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFilename, "apply", "-f", "-")
}

func getHTTPClient(insecure bool) http.Client {
	tr := http.DefaultTransport
	if insecure {
		tr.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return http.Client{
		Transport: tr,
	}
}

func isRancherClusterProvisioned(cluster *rancherclient.Cluster) bool {
	if cluster == nil || cluster.Conditions == nil {
		return false
	}
	for _, condition := range cluster.Conditions {
		if condition.Type == "Provisioned" && condition.Status == "True" {
			return true
		}
	}
	return false
}

func randString() string {
	rand.Seed(time.Now().UnixNano())
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"!@#$%^&*()"

	b := make([]byte, rancherRandPasswordLength)
	for i := 0; i < rancherRandPasswordLength; i++ {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
