//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resourcequota

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// swagger:parameters getResourceQuota deleteResourceQuota
type getResourceQuota struct {
	// in: path
	// required: true
	Name string `json:"quota_name"`
}

// swagger:parameters listResourceQuotas
type listResourceQuotas struct {
	// in: query
	// required: false
	SubjectName string `json:"subject_name,omitempty"`

	// in: query
	// required: false
	SubjectKind string `json:"subject_kind,omitempty"`
}

// swagger:parameters createResourceQuota
type createResourceQuota struct {
	// in: body
	// required: true
	Body struct {
		SubjectName string                       `json:"subjectName"`
		SubjectKind string                       `json:"subjectKind"`
		Quota       kubermaticv1.ResourceDetails `json:"quota"`
	}
}

// swagger:parameters patchResourceQuota
type patchResourceQuota struct {
	// in: path
	// required: true
	Name string `json:"quota_name"`

	// in: body
	// required: true
	Body kubermaticv1.ResourceDetails
}

func (m createResourceQuota) Validate() error {
	if m.Body.SubjectName == "" {
		return utilerrors.NewBadRequest("subject's name cannot be empty")
	}

	if m.Body.SubjectKind == "" {
		return utilerrors.NewBadRequest("subject's kind cannot be empty")
	}

	return nil
}

func DecodeResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req getResourceQuota

	req.Name = mux.Vars(r)["quota_name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`quota_name` cannot be empty")
	}

	return req, nil
}

func DecodeListResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req listResourceQuotas

	req.SubjectName = r.URL.Query().Get("subjectName")
	req.SubjectKind = r.URL.Query().Get("subjectKind")

	return req, nil
}

func DecodeCreateResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req createResourceQuota

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	return req, nil
}

func DecodePatchResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req patchResourceQuota

	req.Name = mux.Vars(r)["quota_name"]
	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`quota_name` cannot be empty")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func GetResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) (*apiv2.ResourceQuota, error) {
	req, ok := request.(getResourceQuota)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	resourceQuota, err := provider.GetUnsecured(ctx, req.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return nil, err
	}

	return convertToAPIStruct(resourceQuota), nil
}

func GetResourceQuotaForProject(ctx context.Context, request interface{}, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter,
	quotaProvider provider.ResourceQuotaProvider) (*apiv2.ResourceQuota, error) {
	req, ok := request.(common.GetProjectRq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}
	if len(req.ProjectID) == 0 {
		return nil, utilerrors.NewBadRequest("the id of the project cannot be empty")
	}

	kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return nil, err
	}

	projectResourceQuota, err := quotaProvider.Get(ctx, userInfo, kubermaticProject.Name, strings.ToLower(kubermaticProject.Kind))
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertToAPIStruct(projectResourceQuota), nil
}

func ListResourceQuotas(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) ([]*apiv2.ResourceQuota, error) {
	req, ok := request.(listResourceQuotas)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	labelSet := make(map[string]string)
	if req.SubjectKind != "" {
		labelSet[kubermaticv1.ResourceQuotaSubjectKindLabelKey] = req.SubjectKind
	}
	if req.SubjectName != "" {
		labelSet[kubermaticv1.ResourceQuotaSubjectNameLabelKey] = req.SubjectName
	}

	resourceQuotaList, err := provider.ListUnsecured(ctx, labelSet)
	if err != nil {
		return nil, err
	}

	resp := make([]*apiv2.ResourceQuota, len(resourceQuotaList.Items))
	for idx, rq := range resourceQuotaList.Items {
		resp[idx] = convertToAPIStruct(&rq)
	}

	return resp, nil
}

func CreateResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(createResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return utilerrors.NewBadRequest(err.Error())
	}

	if err := provider.CreateUnsecured(ctx, kubermaticv1.Subject{Name: req.Body.SubjectName, Kind: req.Body.SubjectKind}, req.Body.Quota); err != nil {
		if apierrors.IsAlreadyExists(err) {
			name := buildNameFromSubject(kubermaticv1.Subject{Name: req.Body.SubjectName, Kind: req.Body.SubjectKind})
			return utilerrors.NewAlreadyExists("ResourceQuota", name)
		}
		return err
	}
	return nil
}

func PatchResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(patchResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	originalResourceQuota, err := provider.GetUnsecured(ctx, req.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return err
	}
	newResourceQuota := originalResourceQuota.DeepCopy()
	newResourceQuota.Spec.Quota = req.Body

	if err := provider.PatchUnsecured(ctx, originalResourceQuota, newResourceQuota); err != nil {
		if apierrors.IsNotFound(err) {
			return utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return err
	}
	return nil
}

func convertToAPIStruct(resourceQuota *kubermaticv1.ResourceQuota) *apiv2.ResourceQuota {
	return &apiv2.ResourceQuota{
		Name:                     resourceQuota.Name,
		SubjectName:              resourceQuota.Spec.Subject.Name,
		SubjectKind:              resourceQuota.Spec.Subject.Kind,
		Quota:                    resourceQuota.Spec.Quota,
		Status:                   resourceQuota.Status,
		SubjectHumanReadableName: resourceQuota.Labels[kubermaticv1.ResourceQuotaSubjectHumanReadableNameLabelKey],
	}
}

func DeleteResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(getResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	if err := provider.DeleteUnsecured(ctx, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return err
	}
	return nil
}
