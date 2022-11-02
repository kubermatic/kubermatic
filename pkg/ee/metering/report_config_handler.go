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

package metering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation"

	"k8s.io/apimachinery/pkg/util/sets"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ReportTypes = sets.NewString("cluster", "namespace")

// swagger:parameters getMeteringReportConfiguration
type getMeteringReportConfig struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// swagger:parameters deleteMeteringReportConfiguration
type deleteMeteringReportConfig struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// swagger:parameters createMeteringReportConfiguration
type createReportConfigurationReq struct {
	// in: path
	// required: true
	Name string `json:"name"`

	// in: body
	Body struct {
		Schedule  string    `json:"schedule"`
		Interval  int32     `json:"interval"`
		Retention *int32    `json:"retention,omitempty"`
		Types     *[]string `json:"types,omitempty"`
	}
}

func (m createReportConfigurationReq) Validate() error {
	if errs := k8svalidation.IsDNS1035Label(m.Name); len(errs) != 0 {
		return utilerrors.NewBadRequest("metering report configuration name must be valid rfc1035 label: %s", strings.Join(errs, ","))
	}

	cronExpressionParser := validation.GetCronExpressionParser()
	if _, err := cronExpressionParser.Parse(m.Body.Schedule); err != nil {
		return utilerrors.NewBadRequest("invalid cron expression format: %s", m.Body.Schedule)
	}

	if m.Body.Interval < 1 {
		return utilerrors.NewBadRequest("interval value cannot be smaller than 1.")
	}

	if m.Body.Retention != nil {
		if *m.Body.Retention < 1 {
			return utilerrors.NewBadRequest("retention value cannot be smaller than 1.")
		}
	}

	if m.Body.Types != nil {
		if len(*m.Body.Types) == 0 {
			return utilerrors.NewBadRequest("at least one report type is required")
		}

		for _, reportType := range *m.Body.Types {
			if !ReportTypes.Has(reportType) {
				return utilerrors.NewBadRequest("invalid metering type: %s", reportType)
			}
		}
	}

	return nil
}

// swagger:parameters updateMeteringReportConfiguration
type updateReportConfigurationReq struct {
	// in: path
	// required: true
	Name string `json:"name"`

	// in: body
	Body struct {
		Schedule  string    `json:"schedule,omitempty"`
		Interval  *int32    `json:"interval,omitempty"`
		Retention *int32    `json:"retention,omitempty"`
		Types     *[]string `json:"types,omitempty"`
	}
}

func (m updateReportConfigurationReq) Validate() error {
	if errs := k8svalidation.IsDNS1035Label(m.Name); len(errs) != 0 {
		return utilerrors.NewBadRequest("metering report configuration name must be valid rfc1035 label: %s", strings.Join(errs, ","))
	}

	if m.Body.Schedule != "" {
		cronExpressionParser := validation.GetCronExpressionParser()
		if _, err := cronExpressionParser.Parse(m.Body.Schedule); err != nil {
			return utilerrors.NewBadRequest("invalid cron expression format: %s", m.Body.Schedule)
		}
	}

	if m.Body.Interval != nil {
		if *m.Body.Interval < 1 {
			return utilerrors.NewBadRequest("interval value cannot be smaller than 1.")
		}
	}

	if m.Body.Retention != nil {
		if *m.Body.Retention < 1 {
			return utilerrors.NewBadRequest("retention value cannot be smaller than 1.")
		}
	}

	if m.Body.Types != nil {
		if len(*m.Body.Types) == 0 {
			return utilerrors.NewBadRequest("at least one report type is required")
		}

		for _, reportType := range *m.Body.Types {
			if !ReportTypes.Has(reportType) {
				return utilerrors.NewBadRequest("invalid metering type: %s", reportType)
			}
		}
	}

	return nil
}

func DecodeGetMeteringReportConfigurationReq(r *http.Request) (interface{}, error) {
	var req getMeteringReportConfig

	req.Name = mux.Vars(r)["name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`name` cannot be empty")
	}

	return req, nil
}

func DecodeCreateMeteringReportConfigurationReq(r *http.Request) (interface{}, error) {
	var req createReportConfigurationReq

	req.Name = mux.Vars(r)["name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`name` cannot be empty")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	return req, nil
}

func DecodeUpdateMeteringReportConfigurationReq(r *http.Request) (interface{}, error) {
	var req updateReportConfigurationReq

	req.Name = mux.Vars(r)["name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`name` cannot be empty")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	return req, nil
}

func DecodeDeleteMeteringReportConfigurationReq(r *http.Request) (interface{}, error) {
	var req deleteMeteringReportConfig

	req.Name = mux.Vars(r)["name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`name` cannot be empty")
	}

	return req, nil
}

// GetMeteringReportConfiguration returns metering report configuration.
// Assumes all Seeds uses the same configuration.
func GetMeteringReportConfiguration(seedsGetter provider.SeedsGetter, request interface{}) (*apiv1.MeteringReportConfiguration, error) {
	if seedsGetter == nil {
		return nil, errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	req, ok := request.(getMeteringReportConfig)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}

	for _, seed := range seeds {
		if seed.Spec.Metering == nil || seed.Spec.Metering.ReportConfigurations == nil {
			continue
		}
		if report, ok := seed.Spec.Metering.ReportConfigurations[req.Name]; ok {
			// Metering configuration is replicated across all seeds.
			// We can return after finding configuration in the first seed.
			return &apiv1.MeteringReportConfiguration{
				Name:      req.Name,
				Schedule:  report.Schedule,
				Interval:  report.Interval,
				Retention: report.Retention,
				Types:     report.Types,
			}, nil
		}
	}

	return nil, utilerrors.NewNotFound("MeteringReportConfiguration", req.Name)
}

// ListMeteringReportConfigurations lists metering report configurations.
// Assumes all Seeds uses the same configuration.
func ListMeteringReportConfigurations(seedsGetter provider.SeedsGetter) ([]apiv1.MeteringReportConfiguration, error) {
	if seedsGetter == nil {
		return nil, errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}

	var resp []apiv1.MeteringReportConfiguration
	for _, seed := range seeds {
		if seed.Spec.Metering == nil {
			continue
		}
		for reportConfigName, reportConfig := range seed.Spec.Metering.ReportConfigurations {
			resp = append(resp, apiv1.MeteringReportConfiguration{
				Name:      reportConfigName,
				Schedule:  reportConfig.Schedule,
				Interval:  reportConfig.Interval,
				Retention: reportConfig.Retention,
				Types:     reportConfig.Types,
			})
		}
		// Metering configuration is replicated across all seeds.
		// We can break after finding configuration in the first seed.
		break
	}

	return resp, nil
}

// CreateMeteringReportConfiguration adds new metering report configuration to the existing map.
func CreateMeteringReportConfiguration(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter,
	masterClient ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	req, ok := request.(createReportConfigurationReq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, fmt.Errorf("failed listing seeds: %w", err)
	}

	var meteringConf *apiv1.MeteringReportConfiguration
	for _, seed := range seeds {
		if meteringConf, err = createMeteringReportConfiguration(ctx, req, seed, masterClient); err != nil {
			return meteringConf, err
		}
	}

	return meteringConf, nil
}

// UpdateMeteringReportConfiguration adds new metering report configuration to the existing map.
func UpdateMeteringReportConfiguration(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter,
	masterClient ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	req, ok := request.(updateReportConfigurationReq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, fmt.Errorf("failed listing seeds: %w", err)
	}

	var meteringConf *apiv1.MeteringReportConfiguration
	for _, seed := range seeds {
		if meteringConf, err = updateMeteringReportConfiguration(ctx, req, seed, masterClient); err != nil {
			return meteringConf, err
		}
	}

	return meteringConf, nil
}

// DeleteMeteringReportConfiguration removes metering report configuration from the existing map.
func DeleteMeteringReportConfiguration(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter,
	masterClient ctrlruntimeclient.Client) error {
	req, ok := request.(deleteMeteringReportConfig)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	seeds, err := seedsGetter()
	if err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}

	for _, seed := range seeds {
		if err := deleteMeteringReportConfiguration(ctx, req.Name, seed, masterClient); err != nil {
			return err
		}
	}

	return nil
}

func createMeteringReportConfiguration(ctx context.Context, reportCfgReq createReportConfigurationReq,
	seed *kubermaticv1.Seed, masterClient ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	if seed.Spec.Metering == nil {
		return nil, fmt.Errorf("metering configuration for %q does not exist", seed.Name)
	}

	if seed.Spec.Metering.ReportConfigurations == nil {
		seed.Spec.Metering.ReportConfigurations = make(map[string]*kubermaticv1.MeteringReportConfiguration)
	}

	if _, exists := seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name]; exists {
		return nil, utilerrors.New(
			http.StatusConflict,
			fmt.Sprintf("report configuration %q already exists", reportCfgReq.Name))
	}

	if reportCfgReq.Body.Types == nil || len(*reportCfgReq.Body.Types) == 0 {
		defaultTypes := ReportTypes.List()
		reportCfgReq.Body.Types = &defaultTypes
	}

	if reportCfgReq.Body.Retention != nil {
		retention := uint32(*reportCfgReq.Body.Retention)
		seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name] = &kubermaticv1.MeteringReportConfiguration{
			Interval:  uint32(reportCfgReq.Body.Interval),
			Schedule:  reportCfgReq.Body.Schedule,
			Retention: &retention,
			Types:     *reportCfgReq.Body.Types,
		}
	} else {
		seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name] = &kubermaticv1.MeteringReportConfiguration{
			Interval:  uint32(reportCfgReq.Body.Interval),
			Schedule:  reportCfgReq.Body.Schedule,
			Retention: nil,
			Types:     *reportCfgReq.Body.Types,
		}
	}

	if err := masterClient.Update(ctx, seed); err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}

	createdConfig := seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name]
	return &apiv1.MeteringReportConfiguration{
		Name:      reportCfgReq.Name,
		Schedule:  createdConfig.Schedule,
		Interval:  createdConfig.Interval,
		Retention: createdConfig.Retention,
		Types:     createdConfig.Types,
	}, nil
}

func updateMeteringReportConfiguration(ctx context.Context, reportCfgReq updateReportConfigurationReq,
	seed *kubermaticv1.Seed, masterClient ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	if seed.Spec.Metering == nil || seed.Spec.Metering.ReportConfigurations == nil {
		return nil, fmt.Errorf("metering report configuration map for %q does not exist", seed.Name)
	}

	if _, exists := seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name]; !exists {
		return nil, utilerrors.New(
			http.StatusNotFound,
			fmt.Sprintf("report configuration %q does not exists", reportCfgReq.Name))
	}

	reportConfiguration := seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name]

	if reportCfgReq.Body.Schedule != "" {
		reportConfiguration.Schedule = reportCfgReq.Body.Schedule
	}

	if reportCfgReq.Body.Interval != nil && *reportCfgReq.Body.Interval >= 1 {
		reportConfiguration.Interval = uint32(*reportCfgReq.Body.Interval)
	}

	if reportCfgReq.Body.Retention == nil {
		reportConfiguration.Retention = nil
	} else if *reportCfgReq.Body.Retention >= 1 {
		retention := uint32(*reportCfgReq.Body.Retention)
		reportConfiguration.Retention = &retention
	}

	if reportCfgReq.Body.Types != nil && len(*reportCfgReq.Body.Types) > 0 {
		reportConfiguration.Types = *reportCfgReq.Body.Types
	}

	if err := masterClient.Update(ctx, seed); err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}

	updatedConfig := seed.Spec.Metering.ReportConfigurations[reportCfgReq.Name]
	return &apiv1.MeteringReportConfiguration{
		Name:      reportCfgReq.Name,
		Schedule:  updatedConfig.Schedule,
		Interval:  updatedConfig.Interval,
		Retention: updatedConfig.Retention,
		Types:     updatedConfig.Types,
	}, nil
}

func deleteMeteringReportConfiguration(ctx context.Context, reportConfigName string, seed *kubermaticv1.Seed, masterClinet ctrlruntimeclient.Client) error {
	if seed.Spec.Metering == nil || seed.Spec.Metering.ReportConfigurations == nil {
		return fmt.Errorf("metering report configuration map for %q does not exist", seed.Name)
	}

	if _, exists := seed.Spec.Metering.ReportConfigurations[reportConfigName]; !exists {
		// Metering report config does not exist. Doing nothing.
		return nil
	}

	delete(seed.Spec.Metering.ReportConfigurations, reportConfigName)
	if err := masterClinet.Update(ctx, seed); err != nil {
		return utilerrors.NewFromKubernetesError(err)
	}

	return nil
}
