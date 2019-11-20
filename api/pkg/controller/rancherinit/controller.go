package rancherinit

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"github.com/rancher/norman/clientbase"
	normantypes "github.com/rancher/norman/types"
	rancherv3 "github.com/rancher/types/client/management/v3"

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
	ControllerName = "kubernatic_rancherinit_controller"
)

// KubeconfigProvider provides functionality to get a clusters admin kubeconfig
type KubeconfigProvider interface {
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
}

type fileHandlingDone func()

type Reconciler struct {
	log *zap.SugaredLogger
	ctrlruntimeclient.Client
	KubeconfigProvider KubeconfigProvider
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	kubeconfigProvider KubeconfigProvider,
) error {

	log = log.Named(ControllerName)
	reconciler := &Reconciler{
		log:                log,
		Client:             mgr.GetClient(),
		KubeconfigProvider: kubeconfigProvider,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler: reconciler,
	})

	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for statefulsets: %v", err)
	}
	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := r.log.With("request", request)
	log.Debug("Processing")
	if request.Name != resources.RancherStatefulSetName {
		return reconcile.Result{}, nil
	}
	statefulSet := &appsv1.StatefulSet{}
	if err := r.Get(ctx, request.NamespacedName, statefulSet); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Errorw("Couldn't find statefulSet", zap.Error(err))
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// all done.
	if statefulSet.Annotations["kubermatic.io/rancher-server-cluster-imported"] == "true" {
		return reconcile.Result{}, nil
	}

	result, err := r.reconcile(ctx, log, statefulSet)
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, statefulSet *appsv1.StatefulSet) (*reconcile.Result, error) {
	if statefulSet.DeletionTimestamp != nil {
		return nil, nil
	}
	// wait for statefulSet to be ready
	if statefulSet.Status.ReadyReplicas == 0 {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if statefulSet.Annotations == nil {
		statefulSet.Annotations = make(map[string]string)
	}
	if statefulSet.Annotations["kubermatic.io/rancher-server-ready"] != "true" { // if we have ready replicas, we should be ready
		statefulSet.Annotations["kubermatic.io/rancher-server-ready"] = "true"
		if err := r.Update(ctx, statefulSet); err != nil {
			if kubeapierrors.IsConflict(err) {
				return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}
			log.Errorw("failed to update rancher statefulSet", zap.Error(err))
			return nil, err
		}
	}
	// initialize rancher statefulSet
	if statefulSet.Annotations["kubermatic.io/rancher-server-initialized"] != "true" {
		if err := r.initRancherServer(ctx, log, statefulSet); err != nil {
			log.Errorw("failed to initialize Rancher Server", zap.Error(err))
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// Setup the user cluster
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: statefulSet.OwnerReferences[0].Name}, cluster); err != nil {
		log.Errorw("can't find cluster", zap.Error(err))
		return nil, err
	}
	rancherRegToken, err := r.rancherClusterWithRegistrationToken(ctx, cluster.Status.NamespaceName, cluster.Spec.HumanReadableName)
	if err != nil {
		log.Errorw("failed to create rancher cluster", zap.Error(err))
		return nil, err
	}
	if rancherRegToken != nil {
		if err := r.applyRancherRegstrationCommand(ctx, log, cluster, rancherRegToken); err != nil {
			log.Errorw("failed to apply rancher regstration command", zap.Error(err))
			return nil, err
		}
		statefulSet.Annotations["kubermatic.io/rancher-server-cluster-imported"] = "true"
		if err := r.Update(ctx, statefulSet); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (r *Reconciler) initRancherServer(ctx context.Context, log *zap.SugaredLogger, statefulSet *appsv1.StatefulSet) error {
	// first, try the default credentials
	token, err := r.rancherLogin(ctx, statefulSet.Namespace, false)
	if err != nil || token == "" {
		log.Errorw("can't find default token:", zap.Error(err))
		// retry with the initSecret, this happens if we previously failed before changing the admin password
		token, err = r.rancherLogin(ctx, statefulSet.Namespace, true)
		if err != nil {
			return fmt.Errorf("can't login to rancher server: %v", err)
		}
	}
	address, err := r.getRancherServerURL(ctx, statefulSet.Namespace)
	if err != nil {
		return fmt.Errorf("can't get rancher service URL: %v", err)
	}
	rancherClient, err := getRancherrancherv3(token, address)
	if err != nil {
		return err
	}
	initSecret, err := r.getRancherInitSecret(ctx, statefulSet.Namespace)
	if err != nil {
		return err
	}
	if err = updateRancherAdminPassword(rancherClient, string(initSecret.Data["password"])); err != nil {
		return err
	}
	// at this point, the rancher statefulSet is initialized and ready to use
	statefulSet.Annotations["kubermatic.io/rancher-server-initialized"] = "true"
	if err := r.Update(ctx, statefulSet); err != nil {
		return err
	}
	log.Infow("rancher server statefulSet initialized successfully")
	return nil

}

func (r *Reconciler) applyRancherRegstrationCommand(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, regToken *rancherv3.ClusterRegistrationToken) error {
	client := getHTTPClient()
	resp, err := client.Get(regToken.ManifestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	manifest, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
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
		return err
	}
	return nil
}

func (r *Reconciler) rancherClusterWithRegistrationToken(ctx context.Context, clusterNamespace, clusterName string) (*rancherv3.ClusterRegistrationToken, error) {
	token, err := r.rancherLogin(ctx, clusterNamespace, true)
	if err != nil {
		return nil, err
	}

	address, err := r.getRancherServerURL(ctx, clusterNamespace)
	if err != nil {
		return nil, fmt.Errorf("can't get rancher service URL: %v", err)
	}
	client, err := getRancherrancherv3(token, address)
	if err != nil {
		return nil, err
	}

	rancherCluster := &rancherv3.Cluster{Name: clusterName}
	var created bool
	rancherClusterList, err := client.Cluster.List(&normantypes.ListOpts{})
	if err != nil {
		return nil, err
	}
	for _, cluster := range rancherClusterList.Data {
		if rancherCluster.Name == cluster.Name {
			rancherCluster = &cluster
			created = true
			break
		}
	}
	if !created {
		rancherCluster, err = client.Cluster.Create(rancherCluster)
		if err != nil {
			return nil, err
		}
	}
	if !isRancherClusterProvisioned(rancherCluster) {
		return client.ClusterRegistrationToken.Create(&rancherv3.ClusterRegistrationToken{
			ClusterID: rancherCluster.ID,
		})
	}
	return nil, err
}

func updateRancherAdminPassword(client *rancherv3.Client, password string) error {
	users, err := client.User.List(&normantypes.ListOpts{})
	if err != nil {
		return err
	}
	for _, user := range users.Data {
		if user.Username == "admin" {
			_, err = client.User.ActionSetpassword(&user, &rancherv3.SetPasswordInput{NewPassword: password})
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (r *Reconciler) rancherLogin(ctx context.Context, namespace string, useSecret bool) (string, error) {
	address, err := r.getRancherServerURL(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("can't get rancher service URL: %v", err)
	}
	url := fmt.Sprintf("%s/v3-public/localProviders/local?action=login", address)
	// rancher default password
	password := "admin"

	if useSecret { // we don't use the default credentials, we use the initSecret
		initSecret, err := r.getRancherInitSecret(ctx, namespace)
		if err != nil {
			return "", err
		}
		password = string(initSecret.Data["password"])
	}

	msg := map[string]interface{}{
		"description":  "",
		"username":     "admin",
		"password":     password,
		"responseType": "json",
		"ttl":          0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	client := getHTTPClient()
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("login failed with status: %v", resp.StatusCode)
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	token, ok := data["token"].(string)
	if !ok {
		return "", fmt.Errorf("can't find rancher token")
	}
	return token, nil
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
	if err := r.Get(ctx, types.NamespacedName{Name: service.OwnerReferences[0].Name}, cluster); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s:%d", cluster.Address.IP, port), nil
}

func (r *Reconciler) getRancherInitSecret(ctx context.Context, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "rancher-init", Namespace: namespace}, secret)
	if err != nil && !kubeapierrors.IsNotFound(err) {
		return nil, err
	}
	if secret.Name != "" {
		return secret, nil
	}
	secret.Name = "rancher-init"
	secret.Namespace = namespace
	secret.Data = map[string][]byte{
		"password": []byte(randString()),
		"user":     []byte("admin"),
	}
	return secret, r.Create(ctx, secret)
}

func randString() string {
	rand.Seed(time.Now().UnixNano())
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"!@#$%^&*()"

	b := make([]byte, 10)
	for i := 0; i < 10; i++ {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (r *Reconciler) writeAdminKubeconfig(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster)
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

func getHTTPClient() http.Client {
	tr := http.DefaultTransport
	tr.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return http.Client{
		Transport: tr,
	}
}

func getRancherrancherv3(token, address string) (*rancherv3.Client, error) {
	if !strings.HasSuffix(address, "/v3") {
		address = fmt.Sprintf("%s/v3", address)
	}
	auth := strings.Split(token, ":")
	if len(auth) != 2 {
		return nil, fmt.Errorf("invalid auth token")
	}
	options := &clientbase.ClientOpts{
		URL:       address,
		AccessKey: auth[0],
		SecretKey: auth[1],
	}

	rancherClient, err := rancherv3.NewClient(options)
	if err != nil {
		return nil, err
	}
	return rancherClient, nil
}

func isRancherClusterProvisioned(cluster *rancherv3.Cluster) bool {
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
