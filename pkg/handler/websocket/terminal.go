/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package websocket

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	END_OF_TRANSMISSION               = "\u0004"
	timeout                           = 2 * time.Minute
	appName                           = "webterminal"
	webTerminalStorage                = "web-terminal-storage"
	podLifetime                       = 30 * time.Minute
	remainingExpirationTimeForWarning = 5 * time.Minute // should be lesser than "podLifetime"
	expirationCheckInterval           = 1 * time.Minute // should be lesser than "remainingExpirationTimeForWarning"
	expirationTimestampKey            = "ExpirationTimestamp"
)

type TerminalConnStatus string

const (
	KubeconfigSecretMissing TerminalConnStatus = "KUBECONFIG_SECRET_MISSING"
	WebterminalPodPending   TerminalConnStatus = "WEBTERMINAL_POD_PENDING"
	WebterminalPodFailed    TerminalConnStatus = "WEBTERMINAL_POD_FAILED"
	ConnectionPoolExceeded  TerminalConnStatus = "CONNECTION_POOL_EXCEEDED"
)

// PtyHandler is what remote command expects from a pty.
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// TerminalSession implements PtyHandler (using a websocket connection).
type TerminalSession struct {
	websocketConn *websocket.Conn
	sizeChan      chan remotecommand.TerminalSize
	doneChan      chan struct{}

	userEmailID   string
	clusterClient ctrlruntimeclient.Client
}

// TerminalMessage is the messaging protocol between ShellController and TerminalSession.
//
// OP          DIRECTION  FIELD(S) USED  DESCRIPTION
// ---------------------------------------------------------------------
// stdin       fe->be     Data           Keystrokes/paste buffer.
// resize      fe->be     Rows, Cols     New terminal size.
// refresh     fe->be                    Signal to extend expiration time.
// stdout      be->fe     Data           Output from the process.
// toast       be->fe     Data           OOB message to be shown to the user.
// msg         be->fe     Data           Any necessary message from the backend to the frontend.
// expiration  be->fe     Data           Expiration timestamp in seconds.
type TerminalMessage struct {
	Op, Data   string
	Rows, Cols uint16
}

// TerminalSize handles pty->process resize events.
// Called in a loop from remotecommand as long as the process is running.
func (t TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	case <-t.doneChan:
		return nil
	}
}

// Read handles pty->process messages (stdin, resize, refresh).
// Called in a loop from remotecommand as long as the process is running.
func (t TerminalSession) Read(p []byte) (int, error) {
	_, m, err := t.websocketConn.ReadMessage()
	if err != nil {
		// Send terminated signal to process to avoid resource leak
		return copy(p, END_OF_TRANSMISSION), err
	}

	var msg TerminalMessage
	if err := json.Unmarshal(m, &msg); err != nil {
		return copy(p, END_OF_TRANSMISSION), err
	}

	switch msg.Op {
	case "stdin":
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
		return 0, nil
	case "refresh":
		return 0, t.extendExpirationTime(context.Background())
	default:
		return copy(p, END_OF_TRANSMISSION), fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

// Write handles process->pty stdout.
// Called from remotecommand whenever there is any output.
func (t TerminalSession) Write(p []byte) (int, error) {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		return 0, err
	}

	if err = t.websocketConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Toast can be used to send the user any OOB messages.
// hterm puts these in the center of the terminal.
func (t TerminalSession) Toast(p string) error {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "toast",
		Data: p,
	})
	if err != nil {
		return err
	}

	if err = t.websocketConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return err
	}
	return nil
}

func (t TerminalSession) extendExpirationTime(ctx context.Context) error {
	return t.clusterClient.Update(ctx,
		genWebTerminalConfigMap(
			userAppName(t.userEmailID),
		)) // regenerate the configmap to extend the expiration period
}

func userAppName(userEmailID string) string {
	return fmt.Sprintf("%s-%s", appName, userEmailID)
}

func expirationCheckRoutine(ctx context.Context, clusterClient ctrlruntimeclient.Client, userEmailID string, websocketConn *websocket.Conn) {
	for {
		time.Sleep(expirationCheckInterval)

		webTerminalConfigMap := &corev1.ConfigMap{}
		if err := clusterClient.Get(ctx, ctrlruntimeclient.ObjectKey{
			Namespace: metav1.NamespaceSystem,
			Name:      userAppName(userEmailID),
		}, webTerminalConfigMap); err != nil {
			log.Logger.Debug(err)
			continue
		}

		if webTerminalConfigMap.Data == nil {
			log.Logger.Debug(errors.New("no data set for webterminal configmap"))
			break
		}

		expirationTimestampStr, isExpirationSet := webTerminalConfigMap.Data[expirationTimestampKey]
		if !isExpirationSet {
			log.Logger.Debug(errors.New("no expiration set in the webterminal configmap"))
			break
		}

		expirationTimestamp, err := strconv.ParseInt(expirationTimestampStr, 10, 64)
		if err != nil {
			log.Logger.Debug(errors.New("invalid expiration timestamp in the webterminal configmap"))
			break
		}

		expirationTime := time.Unix(expirationTimestamp, 0)
		remainingExpirationTime := time.Until(expirationTime)
		if remainingExpirationTime < remainingExpirationTimeForWarning {
			_ = websocketConn.WriteJSON(TerminalMessage{
				Op:   "expiration",
				Data: expirationTimestampStr,
			})
		}
	}
}

// startProcess is called by terminal session creation.
// Executed cmd in the container specified in request and connects it up with the ptyHandler (a session).
func startProcess(ctx context.Context, client ctrlruntimeclient.Client, k8sClient kubernetes.Interface, cfg *rest.Config, userEmailID string, cmd []string, ptyHandler PtyHandler, websocketConn *websocket.Conn) error {
	userAppName := userAppName(userEmailID)

	// check if WEB terminal Pod exists, if not create
	pod := &corev1.Pod{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      userAppName,
	}, pod); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// create Configmap and Pod if not found
		if err := client.Create(ctx, genWebTerminalConfigMap(userAppName)); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			err := client.Update(ctx, genWebTerminalConfigMap(userAppName))
			if err != nil {
				return err
			}
		}
		if err := client.Create(ctx, genWebTerminalPod(userAppName, userEmailID)); err != nil {
			return err
		}
	}

	if !WaitFor(5*time.Second, timeout, func() bool {
		pod := &corev1.Pod{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{
			Namespace: metav1.NamespaceSystem,
			Name:      userAppName,
		}, pod); err != nil {
			return false
		}

		var status string

		switch pod.Status.Phase {
		case corev1.PodRunning:
			return true
		case corev1.PodPending:
			status = string(WebterminalPodPending)
		case corev1.PodFailed:
			status = string(WebterminalPodFailed)
		default:
			status = fmt.Sprintf("pod in %s phase", pod.Status.Phase)
		}
		SendMessage(websocketConn, status)
		return false
	}) {
		return fmt.Errorf("the WEB terminal Pod is not ready")
	}

	go expirationCheckRoutine(ctx, client, userEmailID, websocketConn)

	req := k8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(userAppName).
		Namespace(metav1.NamespaceSystem).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Command: cmd,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, http.MethodPost, req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             ptyHandler,
		Stdout:            ptyHandler,
		Stderr:            ptyHandler,
		TerminalSizeQueue: ptyHandler,
		Tty:               true,
	})
	if err != nil {
		return err
	}

	return nil
}

func genWebTerminalConfigMap(userAppName string) *corev1.ConfigMap {
	expirationTime := time.Now().Add(podLifetime)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userAppName,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Data: map[string]string{
			expirationTimestampKey: strconv.FormatInt(expirationTime.Unix(), 10),
		},
	}
}

func genWebTerminalPod(userAppName, userEmailID string) *corev1.Pod {
	pod := &corev1.Pod{}
	pod.Name = userAppName
	pod.Namespace = metav1.NamespaceSystem
	pod.Labels = map[string]string{
		"app": appName,
	}
	pod.Spec = corev1.PodSpec{}
	pod.Spec.Volumes = getVolumes(userEmailID)
	pod.Spec.InitContainers = []corev1.Container{}
	pod.Spec.Containers = []corev1.Container{
		{
			Name:    userAppName,
			Image:   resources.RegistryQuay + "/kubermatic/web-terminal:0.2.0",
			Command: []string{"/bin/bash", "-c", "--"},
			Args:    []string{"while true; do sleep 30; done;"},
			Env: []corev1.EnvVar{
				{
					Name:  "KUBECONFIG",
					Value: "/etc/kubernetes/kubeconfig/kubeconfig",
				},
				{
					Name:  "PS1",
					Value: "\\$ ",
				},
			},
			VolumeMounts: getVolumeMounts(),
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: resources.Bool(false),
			},
		},
	}

	pod.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser:  resources.Int64(1000),
		RunAsGroup: resources.Int64(3000),
		FSGroup:    resources.Int64(2000),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	return pod
}

func getVolumes(userEmailID string) []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.WEBTerminalKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: handlercommon.KubeconfigSecretName(userEmailID),
				},
			},
		},
		{
			Name: webTerminalStorage,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}
	return vs
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.WEBTerminalKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      webTerminalStorage,
			ReadOnly:  false,
			MountPath: "/data/terminal",
		},
	}
}

// Terminal is called for any new websocket connection.
func Terminal(ctx context.Context, ws *websocket.Conn, client ctrlruntimeclient.Client, k8sClient kubernetes.Interface, cfg *rest.Config, userEmailID string) {
	if err := startProcess(
		ctx,
		client,
		k8sClient,
		cfg,
		userEmailID,
		[]string{"bash", "-c", "cd /data/terminal && /bin/bash"},
		TerminalSession{
			websocketConn: ws,
			userEmailID:   userEmailID,
			clusterClient: client,
		},
		ws); err != nil {
		log.Logger.Debug(err)
		return
	}
}

func EncodeUserEmailtoID(email string) string {
	hasher := md5.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))
}

// WaitFor is a function to wait until callback function return true.
func WaitFor(interval time.Duration, timeout time.Duration, callback func() bool) bool {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		return callback(), nil
	})
	return err == nil
}

// SendMessage sends TerminalMessage to the client. It usually contains a context related
// to the status of background tasks responsible for setting up the terminal.
func SendMessage(wsConn *websocket.Conn, message string) {
	_ = wsConn.WriteJSON(TerminalMessage{
		Op:   "msg",
		Data: message,
	})
}
