package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubermaticerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	corev1interface "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceMetricsInfo is a struct that holds the node metrics
type ResourceMetricsInfo struct {
	Name      string
	Metrics   corev1.ResourceList
	Available corev1.ResourceList
}

// OIDCConfiguration is a struct that holds
// OIDC provider configuration data, read from command line arguments
type OIDCConfiguration struct {
	// URL holds OIDC Issuer URL address
	URL string
	// ClientID holds OIDC ClientID
	ClientID string
	// ClientSecret holds OIDC ClientSecret
	ClientSecret string
	// CookieHashKey is required, used to authenticate the cookie value using HMAC
	// It is recommended to use a key with 32 or 64 bytes.
	CookieHashKey string
	// CookieSecureMode if true then cookie received only with HTTPS otherwise with HTTP.
	CookieSecureMode bool
	// OfflineAccessAsScope if true then "offline_access" scope will be used
	// otherwise 'access_type=offline" query param will be passed
	OfflineAccessAsScope bool
}

// UpdateManager specifies a set of methods to handle cluster versions & updates
type UpdateManager interface {
	GetVersions(string) ([]*version.Version, error)
	GetDefault() (*version.Version, error)
	GetPossibleUpdates(from, clusterType string) ([]*version.Version, error)
}

// ServerMetrics defines metrics used by the API.
type ServerMetrics struct {
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestsDuration       *prometheus.HistogramVec
	InitNodeDeploymentFailures *prometheus.CounterVec
}

// IsBringYourOwnProvider determines whether the spec holds BringYourOwn provider
func IsBringYourOwnProvider(spec kubermaticv1.CloudSpec) (bool, error) {
	providerName, err := provider.ClusterCloudProviderName(spec)
	if err != nil {
		return false, err
	}
	return providerName == provider.BringYourOwnCloudProvider, nil
}

type CredentialsData struct {
	Ctx               context.Context
	KubermaticCluster *kubermaticv1.Cluster
	Client            ctrlruntimeclient.Client
}

func (d CredentialsData) Cluster() *kubermaticv1.Cluster {
	return d.KubermaticCluster
}

func (d CredentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(d.Ctx, d.Client)(configVar, key)
}

// GetReadyPod returns a pod matching provided label selector if it is posting ready status, error otherwise.
// Namespace can be ensured by creating proper PodInterface client.
func GetReadyPod(client corev1interface.PodInterface, labelSelector string) (*corev1.Pod, error) {
	pods, err := client.List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %v", err)
	}

	readyPods := getReadyPods(pods)
	if len(readyPods.Items) < 1 {
		return nil, kubermaticerrors.New(http.StatusBadRequest, "pod is not ready")
	}

	return &readyPods.Items[0], nil
}

// While it is tempting to write our own roundTripper to do all the reading/writing
// in memory intead of opening a TCP port it has some drawbacks:
// * net/http.ReadResponse does not work with websockets, because its body is hardcoded to be an
//   io.ReadCloster and not an io.ReadWriteCloser:
//   * https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/transfer.go#L550
//   * https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/httputil/reverseproxy.go#L518
// * RoundTripping is a bit more complicated than just read and write, mimicking that badly is likely
//   to be more expensive than doing the extra round via the TCP socket:
//   https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/transport.go#L454
func GetPortForwarder(
	coreClient corev1interface.CoreV1Interface,
	cfg *rest.Config,
	namespace string,
	labelSelector string,
	containerPort int) (*portforward.PortForwarder, chan struct{}, error) {
	pod, err := GetReadyPod(coreClient.Pods(namespace), labelSelector)
	if err != nil {
		return nil, nil, err
	}

	dialer, err := getDialerForPod(pod, coreClient.RESTClient(), cfg)
	if err != nil {
		return nil, nil, err
	}

	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	errorBuffer := bytes.NewBuffer(make([]byte, 1024))
	portforwarder, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{"0:" + strconv.Itoa(containerPort)}, stopChan, readyChan, bytes.NewBuffer(make([]byte, 1024)), errorBuffer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create portforwarder: %v", err)
	}

	// Portforwarding is blocking, so we can't do it here
	return portforwarder, stopChan, nil
}

func getReadyPods(pods *corev1.PodList) *corev1.PodList {
	res := &corev1.PodList{}
	for _, pod := range pods.Items {
		if isPodReady(pod) {
			res.Items = append(res.Items, pod)
		}
	}
	return res
}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getDialerForPod(
	pod *corev1.Pod,
	restClient rest.Interface,
	cfg *rest.Config) (httpstream.Dialer, error) {

	// The logic here is copied straight from kubectl at
	// https://github.com/kubernetes/kubernetes/blob/b88662505d288297750becf968bf307dacf872fa/staging/src/k8s.io/kubectl/pkg/cmd/portforward/portforward.go#L334
	req := restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get spdy roundTripper: %v", err)
	}

	return spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL()), nil
}

// WaitForPortForwarder waits until started port forwarder is ready, or emits an error to provided errChan
func WaitForPortForwarder(p *portforward.PortForwarder, errChan <-chan error) error {
	timeout := time.After(10 * time.Second)
	select {
	case <-timeout:
		return errors.New("timeout waiting for backend connection")
	case err := <-errChan:
		return fmt.Errorf("failed to get connection to backend: %v", err)
	case <-p.Ready:
		return nil
	}
}

// WriteHTTPError writes an http error out. If debug is enabled, it also gets logged.
func WriteHTTPError(log *zap.SugaredLogger, w http.ResponseWriter, err error) {
	log.Debugw("Encountered error", zap.Error(err))
	var httpErr kubermaticerrors.HTTPError

	if asserted, ok := err.(kubermaticerrors.HTTPError); ok {
		httpErr = asserted
	} else {
		httpErr = kubermaticerrors.New(http.StatusInternalServerError, err.Error())
	}

	w.WriteHeader(httpErr.StatusCode())
	if _, wErr := w.Write([]byte(httpErr.Error())); wErr != nil {
		log.Errorw("Failed to write body", zap.Error(err))
	}
}

func ForwardPort(log *zap.SugaredLogger, forwarder *portforward.PortForwarder) error {
	// This is blocking so we have to do it in a distinct goroutine
	errorChan := make(chan error)
	go func() {
		log.Debug("Starting to forward port")
		if err := forwarder.ForwardPorts(); err != nil {
			errorChan <- err
		}
	}()

	if err := WaitForPortForwarder(forwarder, errorChan); err != nil {
		return err
	}

	return nil
}

func GetOwnersForProject(userInfo *provider.UserInfo, project *kubermaticv1.Project, memberProvider provider.ProjectMemberProvider, userProvider provider.UserProvider) ([]apiv1.User, error) {
	allProjectMembers, err := memberProvider.List(userInfo, project, &provider.ProjectMemberListOptions{SkipPrivilegeVerification: true})
	if err != nil {
		return nil, err
	}
	projectOwners := []apiv1.User{}
	for _, projectMember := range allProjectMembers {
		if rbac.ExtractGroupPrefix(projectMember.Spec.Group) == rbac.OwnerGroupNamePrefix {
			user, err := userProvider.UserByEmail(projectMember.Spec.UserEmail)
			if err != nil {
				continue
			}
			projectOwners = append(projectOwners, apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					Name: user.Spec.Name,
				},
				Email: user.Spec.Email,
			})
		}
	}
	return projectOwners, nil
}

func GetProject(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string) (*kubermaticv1.Project, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	if adminUserInfo.IsAdmin {
		// get any project for admin
		return privilegedProjectProvider.GetUnsecured(projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
}
