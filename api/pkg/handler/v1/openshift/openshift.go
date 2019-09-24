package openshift

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

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
}

// ConsoleLoginEndpoint is an endpoint that gets an oauth token for the user from the openshift
// oauth service, then redirects back to the openshift console
func ConsoleLoginEndpoint(
	log *zap.SugaredLogger,
	extractor transporthttp.RequestFunc,
	projectProvider provider.ProjectProvider,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {

		log := log.With("endpoint", "openshift-console-login", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)
		request, err := common.DecodeGetClusterReq(ctx, r)
		if err != nil {
			writeHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, err.Error()))
			return
		}

		// The endpoint the middleware is called with is the innermost one, hence we must
		// define it as closure and pass it to the middleware() call below.
		endpoint := func(ctx context.Context, request interface{}) (interface{}, error) {
			cluster, clusterProvider, err := getClusterProviderFromRequest(ctx, request, projectProvider)
			if err != nil {
				writeHTTPError(log, w, err)
				return nil, nil
			}
			log = log.With("cluster", cluster.Name)

			userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
			if !ok {
				writeHTTPError(log, w, kubermaticerrors.New(http.StatusInternalServerError, "couldn't get userInfo"))
				return nil, nil
			}
			if strings.HasPrefix(userInfo.Group, "editors") {
				consoleLogin(ctx, log, w, cluster, clusterProvider.GetSeedClusterAdminRuntimeClient(), r)
			} else {
				writeHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, fmt.Sprintf("user %q does not belong to the editors group", userInfo.Email)))
			}

			return nil, nil
		}
		if _, err := middlewares(endpoint)(ctx, request); err != nil {
			writeHTTPError(log, w, err)
			return
		}
	})
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
			cluster, clusterProvider, err := getClusterProviderFromRequest(ctx, request, projectProvider)
			if err != nil {
				writeHTTPError(log, w, err)
				return nil, nil
			}
			log = log.With("cluster", cluster.Name)

			// Ideally we would cache these to not open a port for every single request
			portforwarder, outBuffer, err := consolePortForwarder(
				ctx,
				log,
				clusterProvider.GetSeedClusterAdminClient().CoreV1(),
				clusterProvider.SeedAdminConfig(),
				cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get portforwarder for console: %v", err)
			}
			defer portforwarder.Close()

			// This is blocking so we have to do it in a distinct goroutine
			errorChan := make(chan error)
			go func() {
				log.Debug("Starting to forward port")
				if err := portforwarder.ForwardPorts(); err != nil {
					errorChan <- err
				}
			}()
			if err := waitForPortForwarder(portforwarder, errorChan); err != nil {
				writeHTTPError(log, w, err)
				return nil, nil
			}
			// PortForwarder does have a `GetPorts` but its plain broken in case the portforwarder picks
			// a random port and always returns 0.
			// TODO @alvaroaleman: Fix upstream
			port, err := getLocalPortFromPortForwardOutput(outBuffer.String())
			if err != nil {
				writeHTTPError(log, w, fmt.Errorf("failed to get backend port: %v", err))
				return nil, nil
			}
			url, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
			if err != nil {
				writeHTTPError(log, w, fmt.Errorf("failed to parse backend url: %v", err))
				return nil, nil
			}

			// The Openshift console needs script-src: unsafe-inline and sryle-src: unsafe-inline.
			// The header here overwrites the setting on the main router, which is more strict.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; object-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; media-src 'self'; frame-ancestors 'self'; frame-src 'self'; connect-src 'self'")
			// Proxy the request
			proxy := httputil.NewSingleHostReverseProxy(url)
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
// This is not done by the user themselves, because:
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
	defer resp.Body.Close()

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
	redirectTargetURLRaw := strings.Replace(initialRequest.URL.Path, "login", "proxy/auth/callback", 1)
	redirectTargetURL, err := url.Parse(redirectTargetURLRaw)
	if err != nil {
		writeHTTPError(log, w, fmt.Errorf("failed to parse target redirect URL: %v", err))
		return
	}
	redirectTargetURL.RawQuery = redirectQueryArgs.Encode()

	http.Redirect(w, initialRequest, redirectTargetURL.String(), http.StatusFound)
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

// writeHTTPError writes an http error out. If debug is enabled, it also gets logged.
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

// While it is tempting to write our own roundTripper to do all the reading/writing
// in memory intead of opening a TCP port it has some drawbacks:
// * net/http.ReadResponse does not work with websockets, because its body is hardcoded to be an
//   io.ReadCloster and not an io.ReadWriteCloser:
//   * https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/transfer.go#L550
//   * https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/httputil/reverseproxy.go#L518
// * RoundTripping is a bit more complicated than just read and write, mimicking that badly is likely
//   to be more expensive than doing the extra round via the TCP socket:
//   https://github.com/golang/go/blob/361ab73305788c4bf35359a02d8873c36d654f1b/src/net/http/transport.go#L454
func consolePortForwarder(
	ctx context.Context,
	log *zap.SugaredLogger,
	corev1Client corev1interface.CoreV1Interface,
	cfg *rest.Config,
	cluster *kubermaticv1.Cluster) (*portforward.PortForwarder, *bytes.Buffer, error) {

	consolePod, err := getReadyOpenshiftConsolePod(corev1Client.Pods(cluster.Status.NamespaceName))
	if err != nil {
		return nil, nil, err
	}

	dealer, err := getDialerForPod(consolePod, corev1Client.RESTClient(), cfg)
	if err != nil {
		return nil, nil, err
	}

	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	errorBuffer := bytes.NewBuffer(make([]byte, 1024))
	outBuffer := bytes.NewBuffer(make([]byte, 1024))
	portforwarder, err := portforward.NewOnAddresses(dealer, []string{"127.0.0.1"}, []string{"0:" + strconv.Itoa(openshiftresources.ConsoleListenPort)}, stopChan, readyChan, outBuffer, errorBuffer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create portforwarder: %v", err)
	}
	// Portforwarding is blocking, so we can't do it here
	return portforwarder, outBuffer, nil
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

func waitForPortForwarder(p *portforward.PortForwarder, errChan <-chan error) error {
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

func getLocalPortFromPortForwardOutput(out string) (int, error) {
	colonSplit := strings.Split(out, ":")
	if n := len(colonSplit); n < 2 {
		return 0, fmt.Errorf("expected at least two results when splitting by colon, got %d", n)
	}
	spaceSplit := strings.Split(colonSplit[1], " ")
	if n := len(spaceSplit); n < 1 {
		return 0, fmt.Errorf("expected at least one result when splitting by space, got %d", n)
	}
	result, err := strconv.Atoi(spaceSplit[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse int: %v", err)
	}
	return result, nil
}

func getClusterProviderFromRequest(
	ctx context.Context,
	request interface{},
	projectProvider provider.ProjectProvider) (*kubermaticv1.Cluster, *kubernetesprovider.ClusterProvider, error) {

	req, ok := request.(common.GetClusterReq)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusBadRequest, "invalid request")
	}
	cluster, err := cluster.GetCluster(ctx, req, projectProvider)
	if err != nil {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, err.Error())
	}

	rawClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "no clusterProvider in request")
	}
	clusterProvider, ok := rawClusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}
	return cluster, clusterProvider, nil
}
