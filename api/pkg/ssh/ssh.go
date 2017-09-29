package ssh

import (
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// UserListOptions returns a ListOptions object for retrieving objects with the label kubermatic-user-hash=username
func UserListOptions(username string) (metav1.ListOptions, error) {
	label, err := labels.NewRequirement(util.DefaultUserLabel, selection.Equals, []string{util.UserToLabel(username)})
	if err != nil {
		return metav1.ListOptions{}, err
	}
	return metav1.ListOptions{
		LabelSelector: labels.NewSelector().Add(*label).String(),
	}, nil
}
