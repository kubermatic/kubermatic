package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UserClusterBindingComponentValue = "userClusterBinding"
	UserClusterBindingLabelSelector  = "component=userClusterBinding"
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
		if subject.Kind == rbacv1.GroupKind || subject.Kind == rbacv1.UserKind {
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

func ListRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		labelSelector, err := labels.Parse(UserClusterBindingLabelSelector)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleBindingList := &rbacv1.RoleBindingList{}
		if err := client.List(
			ctx,
			roleBindingList,
			&ctrlruntimeclient.ListOptions{LabelSelector: labelSelector, Namespace: req.Namespace},
		); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var bindings []rbacv1.RoleBinding
		for _, binding := range roleBindingList.Items {
			if removeUserClusterRBACPrefix(binding.RoleRef.Name) == req.RoleID {
				bindings = append(bindings, binding)
			}
		}

		return convertInternalRoleBindingsToExternal(bindings), nil
	}
}

// listRoleBindingReq defines HTTP request for listRoleBinding endpoint
// swagger:parameters listRoleBinding
type listRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
}

func DecodeListRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listRoleBindingReq
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

	return req, nil
}

func GetRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleID := addUserClusterRBACPrefix(req.RoleID)

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID, Namespace: req.Namespace}, &rbacv1.Role{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		bindingID := addUserClusterRBACPrefix(req.BindingID)
		binding := &rbacv1.RoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID, Namespace: req.Namespace}, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalRoleBindingToExternal(binding), nil
	}
}

func DeleteRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleID := addUserClusterRBACPrefix(req.RoleID)

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID, Namespace: req.Namespace}, &rbacv1.Role{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		bindingID := addUserClusterRBACPrefix(req.BindingID)
		binding := &rbacv1.RoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID, Namespace: req.Namespace}, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Delete(ctx, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// roleBindingReq defines HTTP request for getRoleBinding endpoint
// swagger:parameters getRoleBinding deleteRoleBinding
type roleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: path
	// required: true
	BindingID string `json:"binding_id"`
}

func DecodeRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req roleBindingReq
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

	bindingID := mux.Vars(r)["binding_id"]
	if bindingID == "" {
		return "", fmt.Errorf("'binding_id' parameter is required but was not provided")
	}
	req.BindingID = bindingID
	return req, nil
}

func PatchRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleID := addUserClusterRBACPrefix(req.RoleID)

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID, Namespace: req.Namespace}, &rbacv1.Role{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		bindingID := addUserClusterRBACPrefix(req.BindingID)
		existingBinding := &rbacv1.RoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID, Namespace: req.Namespace}, existingBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingBindingJSON, err := json.Marshal(existingBinding)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing role binding: %v", err)
		}

		patchedBindingJSON, err := jsonpatch.MergePatch(existingBindingJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch role binding: %v", err)
		}

		var patchedBinding *rbacv1.RoleBinding
		err = json.Unmarshal(patchedBindingJSON, &patchedBinding)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched role binding: %v", err)
		}

		if err := client.Update(ctx, patchedBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalRoleBindingToExternal(patchedBinding), nil
	}
}

// patchRoleBindingReq defines HTTP request for patchRoleBinding endpoint
// swagger:parameters patchRoleBinding
type patchRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: path
	// required: true
	BindingID string `json:"binding_id"`
	// in: body
	Patch []byte
}

func DecodePatchRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchRoleBindingReq
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

	bindingID := mux.Vars(r)["binding_id"]
	if bindingID == "" {
		return "", fmt.Errorf("'binding_id' parameter is required but was not provided")
	}
	req.BindingID = bindingID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func CreateClusterRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClusterRoleBindingReq)
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

		roleID := addUserClusterRBACPrefix(req.RoleID)
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		binding := req.Body
		clusterRoleBinding, err := generateRBACClusterRoleBinding(binding.Name, req.RoleID, binding.Subjects)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster role binding: %v", err)
		}

		if err := client.Create(ctx, clusterRoleBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterRoleBindingToExternal(clusterRoleBinding), nil
	}
}

// Validate validates createRoleReq request
func (r createClusterRoleBindingReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}

	if r.Body.Name == "" {
		return fmt.Errorf("the request Body name cannot be empty")
	}

	if r.Body.RoleRefName != r.RoleID {
		return fmt.Errorf("the request RoleRefName must be the same as RoleID")
	}

	for _, subject := range r.Body.Subjects {
		if subject.Kind == rbacv1.GroupKind || subject.Kind == rbacv1.UserKind {
			continue
		}
		return fmt.Errorf("the request Body subjects contain wrong kind name: '%s'. Should be 'Group' or 'User'", subject.Kind)
	}

	return nil
}

// createClusterRoleBindingReq defines HTTP request for createClusterRoleBinding endpoint
// swagger:parameters createClusterRoleBinding
type createClusterRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: body
	Body apiv1.ClusterRoleBinding
}

func DecodeCreateClusterRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createClusterRoleBindingReq
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

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func ListClusterRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClusterRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		labelSelector, err := labels.Parse(UserClusterBindingLabelSelector)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
		if err := client.List(ctx, clusterRoleBindingList, &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var bindings []rbacv1.ClusterRoleBinding
		for _, binding := range clusterRoleBindingList.Items {
			if removeUserClusterRBACPrefix(binding.RoleRef.Name) == req.RoleID {
				bindings = append(bindings, binding)
			}
		}

		return convertInternalClusterRoleBindingsToExternal(bindings), nil
	}
}

// listClusterRoleBindingReq defines HTTP request for listClusterRoleBinding endpoint
// swagger:parameters listClusterRoleBinding
type listClusterRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
}

func DecodeListClusterRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listClusterRoleBindingReq
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

	return req, nil
}

func GetClusterRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleID := addUserClusterRBACPrefix(req.RoleID)

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterRoleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		bindingID := addUserClusterRBACPrefix(req.BindingID)
		binding := &rbacv1.ClusterRoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID}, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterRoleBindingToExternal(binding), nil
	}
}

func DeleteClusterRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleID := addUserClusterRBACPrefix(req.RoleID)

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterRoleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		bindingID := addUserClusterRBACPrefix(req.BindingID)
		binding := &rbacv1.ClusterRoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID}, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Delete(ctx, binding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// clusterRoleBindingReq defines HTTP request for getClusterRoleBinding endpoint
// swagger:parameters getClusterRoleBinding deleteClusterRoleBinding
type clusterRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	BindingID string `json:"binding_id"`
}

func DecodeClusterRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterRoleBindingReq
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

	bindingID := mux.Vars(r)["binding_id"]
	if bindingID == "" {
		return "", fmt.Errorf("'binding_id' parameter is required but was not provided")
	}
	req.BindingID = bindingID
	return req, nil
}

// PatchClusterRoleBindingEndpoint edits ClusterRoleBindings subjects
func PatchClusterRoleBindingEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchClusterRoleBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if ClusterRole exists for the binding
		clusterRoleID := addUserClusterRBACPrefix(req.RoleID)
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterRoleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Get kubernetes ClusterRoleBinding and patch it with kubermatic API ClusterRoleBinding.
		// The kubermatic ClusterRoleBinding contains kubernetes Subjects type and can be apply on
		// the kubernetes subject object.
		bindingID := addUserClusterRBACPrefix(req.BindingID)
		existingBinding := &rbacv1.ClusterRoleBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: bindingID}, existingBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingBindingJSON, err := json.Marshal(existingBinding)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing cluster role binding: %v", err)
		}

		patchedBindingJSON, err := jsonpatch.MergePatch(existingBindingJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch role binding: %v", err)
		}

		var patchedBinding *rbacv1.ClusterRoleBinding
		err = json.Unmarshal(patchedBindingJSON, &patchedBinding)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched cluster role binding: %v", err)
		}

		if err := client.Update(ctx, patchedBinding); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterRoleBindingToExternal(patchedBinding), nil
	}
}

// patchClusterRoleBindingReq defines HTTP request for patchClusterRoleBinding endpoint
// swagger:parameters patchClusterRoleBinding
type patchClusterRoleBindingReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	BindingID string `json:"binding_id"`
	// in: body
	Patch []byte
}

func DecodePatchClusterRoleBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchClusterRoleBindingReq
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

	bindingID := mux.Vars(r)["binding_id"]
	if bindingID == "" {
		return "", fmt.Errorf("'binding_id' parameter is required but was not provided")
	}
	req.BindingID = bindingID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// generateRBACRoleBinding creates role binding
func generateRBACRoleBinding(name, namespace, roleName string, subjects []rbacv1.Subject) (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addUserClusterRBACPrefix(name),
			Labels:    map[string]string{UserClusterComponentKey: UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     addUserClusterRBACPrefix(roleName),
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

// generateRBACClusterRoleBinding creates cluster role binding
func generateRBACClusterRoleBinding(name, roleName string, subjects []rbacv1.Subject) (*rbacv1.ClusterRoleBinding, error) {

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   addUserClusterRBACPrefix(name),
			Labels: map[string]string{UserClusterComponentKey: UserClusterBindingComponentValue},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     addUserClusterRBACPrefix(roleName),
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
		clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, newSubject)
	}
	return clusterRoleBinding, nil
}

func convertInternalRoleBindingToExternal(clusterRole *rbacv1.RoleBinding) *apiv1.RoleBinding {
	roleBinding := &apiv1.RoleBinding{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                removeUserClusterRBACPrefix(clusterRole.Name),
			Name:              removeUserClusterRBACPrefix(clusterRole.Name),
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Namespace:   clusterRole.Namespace,
		RoleRefName: removeUserClusterRBACPrefix(clusterRole.RoleRef.Name),
		Subjects:    clusterRole.Subjects,
	}

	return roleBinding
}

func convertInternalRoleBindingsToExternal(roleBindings []rbacv1.RoleBinding) []*apiv1.RoleBinding {
	var apiRoleBinding []*apiv1.RoleBinding
	for _, binding := range roleBindings {
		apiRoleBinding = append(apiRoleBinding, convertInternalRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiRoleBinding
}

func convertInternalClusterRoleBindingToExternal(clusterRoleBinding *rbacv1.ClusterRoleBinding) *apiv1.ClusterRoleBinding {
	binding := &apiv1.ClusterRoleBinding{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                removeUserClusterRBACPrefix(clusterRoleBinding.Name),
			Name:              removeUserClusterRBACPrefix(clusterRoleBinding.Name),
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRoleBinding.CreationTimestamp.Time),
		},
		RoleRefName: removeUserClusterRBACPrefix(clusterRoleBinding.RoleRef.Name),
		Subjects:    clusterRoleBinding.Subjects,
	}

	return binding
}

func convertInternalClusterRoleBindingsToExternal(clusterRoleBindings []rbacv1.ClusterRoleBinding) []*apiv1.ClusterRoleBinding {
	var apiClusterRoleBinding []*apiv1.ClusterRoleBinding
	for _, binding := range clusterRoleBindings {
		apiClusterRoleBinding = append(apiClusterRoleBinding, convertInternalClusterRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiClusterRoleBinding
}
