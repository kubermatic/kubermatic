package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserClusterBindingComponentValue = "userClusterBinding"
)

func CreateRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		binding := req.Body

		roleBinding, err := generateRBACRoleBinding(binding.Name, binding.Namespace, req.RoleID, binding.Subjects)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster role: %v", err)
		}

		if err := client.Create(ctx, roleBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalRoleBindingToExternal(roleBinding), nil
	}
}

// Validate validates createRoleReq request
func (r createRoleBindingReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}

	if r.Body.Namespace == "" || r.Body.Name == "" {
		return fmt.Errorf("the request Body name and namespace cannot be empty")
	}

	if r.Body.Namespace != r.Namespace {
		return fmt.Errorf("the request namespace must be the same as role binding namespace")
	}

	if r.Body.RoleRefName != r.RoleID {
		return fmt.Errorf("the request RoleRefName must be the same as RoleID")
	}

	for _, subject := range r.Body.Subjects {
		if subject.Kind == "Group" || subject.Kind == "User" {
			continue
		}
		return fmt.Errorf("the request Body subjects contain wrong kind name: '%s'. Should be 'Group' or 'User'", subject.Kind)
	}

	return nil
}

// createRoleBindingReq defines HTTP request for createRoleBinding endpoint
// swagger:parameters createRoleBinding
type createRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: body
	Body apiv1.RoleBinding
}

func DecodeCreateRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createRoleBindingReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
	req.ClusterID = clusterID

	roleID := mux.Vars(r)["role_id"]
	if roleID == "" {
		return "", fmt.Errorf("'role_id' parameter is required but was not provided")
	}
	req.RoleID = roleID
	namespace := mux.Vars(r)["namespace"]
	if namespace == "" {
		return "", fmt.Errorf("'namespace' parameter is required but was not provided")
	}
	req.Namespace = namespace

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// generateRBACRoleBinding creates role binding
func generateRBACRoleBinding(name, namespace, roleName string, subjects []apiv1.Subject) (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addUserClusterRolePrefix(name),
			Labels:    map[string]string{UserClusterComponentKey: UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     addUserClusterRolePrefix(roleName),
		},
		Subjects: []rbacv1.Subject{},
	}

	for _, subject := range subjects {
		newSubject := rbacv1.Subject{
			Kind:     rbacv1.UserKind,
			APIGroup: rbacv1.GroupName,
			Name:     subject.Name,
		}
		if subject.Kind == "Group" {
			newSubject.Kind = rbacv1.GroupKind
		}
		roleBinding.Subjects = append(roleBinding.Subjects, newSubject)
	}
	return roleBinding, nil
}

func convertInternalRoleBindingToExternal(clusterRole *rbacv1.RoleBinding) *apiv1.RoleBinding {
	roleBinding := &apiv1.RoleBinding{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                removeUserClusterRolePrefix(clusterRole.Name),
			Name:              removeUserClusterRolePrefix(clusterRole.Name),
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Namespace:   clusterRole.Namespace,
		RoleRefName: removeUserClusterRolePrefix(clusterRole.RoleRef.Name),
		Subjects:    []apiv1.Subject{},
	}

	for _, subjectInternal := range clusterRole.Subjects {
		subjectExternal := apiv1.Subject{
			Kind: subjectInternal.Kind,
			Name: subjectInternal.Name,
		}
		roleBinding.Subjects = append(roleBinding.Subjects, subjectExternal)
	}

	return roleBinding
}
