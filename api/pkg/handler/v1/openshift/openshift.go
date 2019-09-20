package openshift

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"go.uber.org/zap"

	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	kubermaticerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/httpstream"
	corev1interface "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Minimal wrapper to implement the http.Handler interface
type dynamicHTTPHandler func(http.ResponseWriter, *http.Request)

// ServeHTTP implements http.Handler
func (dHandler dynamicHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dHandler(w, r)
	return
}

// ConsoleProxyEndpoint is an endpoint that proxies to the Openshift console running
// in the seed. It also performs authentication on the users behalf. Currently, it only supports
// login as cluster-admin user, so this must not be accessible for users that are not cluster admin.
func ConsoleProxyEndpoint(
	log *zap.SugaredLogger,
	extractor transporthttp.RequestFunc,
	projectProvider provider.ProjectProvider,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {

		log := log.With("endpoint", "openshift-console-proxy", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)
		request, err := common.DecodeGetClusterReq(ctx, r)
		if err != nil {
			writeHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, err.Error()))
			return
		}

		// The endpoint the middleware is called with is the innermost one, hence we must
		// define it as closure and pass it to the middleware() call below.
		endpoint := func(ctx context.Context, request interface{}) (interface{}, error) {
			req, ok := request.(common.GetClusterReq)
			if !ok {
				return nil, kubermaticerrors.New(http.StatusBadRequest, "invalid request")
			}
			cluster, err := cluster.GetCluster(ctx, req, projectProvider)
			if err != nil {
				return nil, kubermaticerrors.New(http.StatusInternalServerError, err.Error())
			}
			log = log.With("cluster", cluster.Name)

			rawClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
			if !ok {
				return nil, kubermaticerrors.New(http.StatusInternalServerError, "no clusterProvider in request")
			}
			clusterProvider, ok := rawClusterProvider.(*kubernetesprovider.ClusterProvider)
			if !ok {
				return nil, kubermaticerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
			}
			if strings.HasSuffix(r.URL.Path, "console-login") {
				// TODO: Verify the user may do this
				consoleLogin(ctx, log, w, cluster, clusterProvider.GetSeedClusterAdminRuntimeClient(), r)
				return nil, nil
			}

			// TODO: Cache these, the current approach of creating a roundTripper per request
			// is extremely inefficient. Keep in mind that this has to be threadsafe.
			roundTripper, err := consoleRoundTripper(
				ctx,
				log,
				clusterProvider.GetSeedClusterAdminClient().CoreV1(),
				clusterProvider.SeedAdminConfig(),
				cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get RoundTripper for console: %v", err)
			}
			defer func() {
				if err := roundTripper.target.Close(); err != nil {
					log.Errorw("Failed closing transport", zap.Error(err))
				}
			}()

			// Proxy the request
			proxy := &httputil.ReverseProxy{
				Director:  func(_ *http.Request) { return },
				Transport: roundTripper}
			proxy.ServeHTTP(w, r)

			return nil, nil
		}

		if _, err := middlewares(endpoint)(ctx, request); err != nil {
			writeHTTPError(log, w, err)
			return
		}
	})

}

// consoleLogin loggs an user into the console by doing the oauth login, then returning a redirect.
// This is not done by the user themsvelces, because:
// * The openshift OAuth server is under the same URL as the kubermatic UI but doesn't have a
//   certificate signed by a CA the browser has. This mean that if HSTS is enabled, the brower
//   wont allow the user to visit that URL.
// * It is poor UX to require the User to login twice.
func consoleLogin(
	ctx context.Context,
	log *zap.SugaredLogger,
	w http.ResponseWriter,
	cluster *kubermaticv1.Cluster,
	seedClient ctrlruntimeclient.Client,
	initialRequest *http.Request) {

	log.Debug("Login request received")

	oauthServiceName := types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      openshiftresources.OAuthServiceName,
	}
	oauthService := &corev1.Service{}
	if err := seedClient.Get(ctx, oauthServiceName, oauthService); err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to retrieve oauth service: %v", err))
		return
	}
	if n := len(oauthService.Spec.Ports); n != 1 {
		writeHTTPError(log, w, fmt.Errorf("OAuth service doesn't have exactly one port but %d", n))
		return
	}
	oauthPort := oauthService.Spec.Ports[0].NodePort

	oauthPasswordSecretName := types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      openshiftresources.ConsoleAdminPasswordSecretName,
	}
	oauthPasswordSecret := &corev1.Secret{}
	if err := seedClient.Get(ctx, oauthPasswordSecretName, oauthPasswordSecret); err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to get OAuth credential secret: %v", err))
		return
	}
	oauthPassword := string(oauthPasswordSecret.Data[openshiftresources.ConsoleAdminUserName])
	if oauthPassword == "" {
		writeHTTPError(log, w, errors.New("no OAuth password found"))
		return
	}

	oauthStateValue, err := generateRandomOauthState()
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to get oauth state token: %v", err))
		return
	}

	queryArgs := url.Values{
		"client_id":     []string{"console"},
		"response_type": []string{"code"},
		"scope":         []string{"user:full"},
		"state":         []string{oauthStateValue},
	}
	// TODO: Should we put that into cluster.Address?
	oauthURL, err := url.Parse(fmt.Sprintf("https://%s:%d/oauth/authorize", cluster.Address.ExternalName, oauthPort))
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to parse oauth url: %v", err))
		return
	}
	oauthURL.RawQuery = queryArgs.Encode()

	oauthRequest, err := http.NewRequest(http.MethodGet, oauthURL.String(), nil)
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to construct query for oauthRequest: %v", err))
		return
	}
	oauthRequest.SetBasicAuth(openshiftresources.ConsoleAdminUserName, oauthPassword)

	resp, err := httpRequestOAuthClient().Do(oauthRequest)
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to get oauth code: %v", err))
		return
	}

	redirectURL, err := resp.Location()
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to get redirectURL: %v", err))
		return
	}

	oauthCode := redirectURL.Query().Get("code")
	if oauthCode == "" {
		writeHTTPError(log, w, errors.New("did not get an OAuth code back from Openshift OAuth server"))
	}
	// We don't check this here again. If something is wrong with it, Openshift will complain
	returnedOAuthState := redirectURL.Query().Get("state")
	http.SetCookie(w, &http.Cookie{Name: "state-token", Value: returnedOAuthState})

	redirectQueryArgs := url.Values{
		"state": []string{returnedOAuthState},
		"code":  []string{oauthCode},
	}
	// Leave the Host unset, http.Redirect will fill it with the host from the original request
	redirectTargetURLRaw := strings.Replace(initialRequest.URL.Path, "console-login", "console/auth/callback", 1)
	redirectTargetURL, err := url.Parse(redirectTargetURLRaw)
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to parse target redirect URL: %v", err))
		return
	}
	redirectTargetURL.RawQuery = redirectQueryArgs.Encode()

	http.Redirect(w, initialRequest, redirectURL.String(), http.StatusFound)
}

// generateRandomOauthState generates a random string that is being used when performing the
// oauth request. The Openshift console checks that the query param on the request it received
// matches a cookie:
// https://github.com/openshift/console/blob/5c80c44d31e244b01dd9bbb4c8b1adec18e3a46b/auth/auth.go#L375
func generateRandomOauthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to get entropy: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// writeHTTPError writes an http error out. If debug is enabled, it also gets loogged.
func writeHTTPError(log *zap.SugaredLogger, w http.ResponseWriter, err error) {
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

// httpRequestOAuthClient is used to perform the OAuth request.
// it needs some special settings.
func httpRequestOAuthClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// TODO: Fetch the CA instead and use it for verification
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		// We must not follow the redirect
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// inMemRoundTripper is a roundTripper what reads and writes from/to
// an io.ReadWriteCloser. We use it when proxying to the backing console
// pod, to avoid opening a TCP port in one goroutine, that gets read from
// another goroutine, both with the same lifecycle.
type inMemRoundTripper struct {
	target io.ReadWriteCloser
}

// RoundTrip implements the net/http.RoundTripper interface.
func (imrt *inMemRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := r.Write(imrt.target); err != nil {
		return nil, fmt.Errorf("failed to write request: %v", err)
	}
	// Would be nicer if we copied this without buffering.
	return http.ReadResponse(bufio.NewReader(imrt.target), nil)
}

func consoleRoundTripper(
	ctx context.Context,
	log *zap.SugaredLogger,
	corev1Client corev1interface.CoreV1Interface,
	cfg *rest.Config,
	cluster *kubermaticv1.Cluster) (*inMemRoundTripper, error) {

	consolePod, err := getReadyOpenshiftConsolePod(corev1Client.Pods(cluster.Status.NamespaceName))
	if err != nil {
		return nil, err
	}

	connection, err := getConnectionForPod(consolePod, corev1Client.RESTClient(), cfg)
	if err != nil {
		return nil, err
	}

	// Source:
	// https://github.com/kubernetes/kubernetes/blob/b88662505d288297750becf968bf307dacf872fa/staging/src/k8s.io/client-go/tools/portforward/portforward.go#L359
	headers := http.Header{}
	headers.Set(corev1.PortHeader, strconv.Itoa(openshiftresources.ConsoleListenPort))
	headers.Set(corev1.PortForwardRequestIDHeader, "0")

	// If the error channel doesn't get opened, the request doesn't get forwarded. Other
	// than that it does not seem to have an obvious purpose.
	headers.Set(corev1.StreamType, corev1.StreamTypeError)
	errorStream, err := connection.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to create error stream: %v", err)
	}
	if err := errorStream.Close(); err != nil {
		return nil, fmt.Errorf("failed to close error stream: %v", err)
	}

	headers.Set(corev1.StreamType, corev1.StreamTypeData)
	dataStream, err := connection.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to open backend connection: %v", err)
	}

	return &inMemRoundTripper{target: dataStream}, nil
}

func getConnectionForPod(
	pod *corev1.Pod,
	restClient rest.Interface,
	cfg *rest.Config) (httpstream.Connection, error) {

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

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	// We can not use client-gos portforward package directly, as it does a portforward, i.E. binds to
	// a local port. What we want here is the raw io.ReadWriteCloser to be able to proxy.
	connection, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade connection: %v", err)
	}
	return connection, nil
}

func getReadyOpenshiftConsolePod(client corev1interface.PodInterface) (*corev1.Pod, error) {
	// TODO: Export the labelselector from the openshift resources
	consolePods, err := client.List(metav1.ListOptions{LabelSelector: "app=openshift-console"})
	if err != nil {
		return nil, fmt.Errorf("failed to get openshift console pod: %v", err)
	}

	readyConsolePods := getReadyPods(consolePods)
	if len(readyConsolePods.Items) < 1 {
		return nil, kubermaticerrors.New(http.StatusBadRequest, "openshift console is not ready")
	}

	return &readyConsolePods.Items[0], nil
}

func getReadyPods(pods *corev1.PodList) *corev1.PodList {
	res := &corev1.PodList{}
	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				res.Items = append(res.Items, pod)
				break
			}
		}
	}
	return res
}
