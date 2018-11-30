package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type validateUserContext struct {
	kubermaticClient kubermaticclientset.Interface
	config           *rest.Config
}

func main() {
	var kubeconfig, masterURL string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	var err error
	ctx := validateUserContext{}
	ctx.config, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	ctx.config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	ctx.config.APIPath = "/apis"
	ctx.kubermaticClient = kubermaticclientset.NewForConfigOrDie(ctx.config)

	if err := validateUsers(&ctx); err != nil {
		glog.Fatalf("failed to validate users: %v", err)
	}

	glog.Infoln("validation success")

}

func validateUsers(ctx *validateUserContext) error {
	usersEmailmap := make(map[string]bool)
	userList, err := ctx.kubermaticClient.KubermaticV1().Users().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	w := sync.WaitGroup{}
	w.Add(len(userList.Items))

	for i := range userList.Items {
		email := strings.ToLower(userList.Items[i].Spec.Email)
		if _, ok := usersEmailmap[email]; ok {
			return fmt.Errorf("two users with the same email: %s", email)
		}
		usersEmailmap[email] = true
	}

	return nil
}
