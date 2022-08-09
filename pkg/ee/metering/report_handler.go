//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/util/s3"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const ReportPrefix = "report-"

var urlValidTime = time.Hour * 1

// ListReports returns a list of all reports generated by metering
// Assumes all Seeds uses the same secrets.
func ListReports(ctx context.Context, req interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) ([]apiv1.MeteringReport, error) {
	if seedsGetter == nil || seedClientGetter == nil {
		return nil, errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	request, ok := req.(listReportReq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	seedsMap, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	prefix := ReportPrefix
	if request.ConfigurationName != "" {
		prefix = request.ConfigurationName + "/" + prefix
	}

	listOptions := minio.ListObjectsOptions{
		MaxKeys:    request.MaxKeys,
		StartAfter: request.StartAfter,
		Prefix:     prefix,
	}

	for _, seed := range seedsMap {
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return nil, err
		}

		reports, err := getReportsForSeed(ctx, listOptions, seedClient)
		if err != nil {
			return nil, err
		}

		return reports, nil
	}

	return nil, nil
}

func GetReport(ctx context.Context, req interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) (string, error) {
	if seedsGetter == nil || seedClientGetter == nil {
		return "", errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	request, ok := req.(getReportReq)
	if !ok {
		return "", utilerrors.NewBadRequest("invalid request")
	}

	reportPath := request.ReportName
	if request.ConfigurationName != "" {
		reportPath = request.ConfigurationName + "/" + reportPath
	}

	seedsMap, err := seedsGetter()
	if err != nil {
		return "", err
	}

	for _, seed := range seedsMap {
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return "", err
		}

		mc, bucket, err := getS3DataFromSeed(ctx, seedClient)
		if err != nil {
			return "", err
		}

		_, err = mc.StatObject(ctx, bucket, reportPath, minio.StatObjectOptions{})
		if err != nil {
			return "", err
		}

		presignedURL, err := mc.PresignedGetObject(ctx, bucket, reportPath, urlValidTime, nil)
		if err != nil {
			return "", err
		}

		urlString := presignedURL.String()
		return urlString, nil
	}

	return "", utilerrors.New(http.StatusNotFound, "report not found")
}

func DeleteReport(ctx context.Context, req interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	if seedsGetter == nil || seedClientGetter == nil {
		return errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	request, ok := req.(deleteReportReq)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	reportPath := request.ReportName
	if request.ConfigurationName != "" {
		reportPath = request.ConfigurationName + "/" + reportPath
	}

	seedsMap, err := seedsGetter()
	if err != nil {
		return err
	}

	for _, seed := range seedsMap {
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return err
		}

		mc, bucket, err := getS3DataFromSeed(ctx, seedClient)
		if err != nil {
			return err
		}

		err = mc.RemoveObject(ctx, bucket, reportPath, minio.RemoveObjectOptions{})
		if err != nil {
			return err
		}

		return nil
	}

	return utilerrors.New(http.StatusNotFound, "report not found")
}

func getReportsForSeed(ctx context.Context, options minio.ListObjectsOptions, seedClient ctrlruntimeclient.Client) ([]apiv1.MeteringReport, error) {
	mc, s3bucket, err := getS3DataFromSeed(ctx, seedClient)
	if err != nil {
		return nil, err
	}

	var reports []apiv1.MeteringReport

	mcCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for report := range mc.ListObjects(mcCtx, s3bucket, options) {
		if report.Err != nil {
			return nil, errors.New(report.Err.Error())
		}

		reports = append(reports, apiv1.MeteringReport{
			Name:         report.Key,
			LastModified: report.LastModified,
			Size:         report.Size,
		})
		if len(reports) == options.MaxKeys {
			break
		}
	}

	return reports, nil
}

func getS3DataFromSeed(ctx context.Context, seedClient ctrlruntimeclient.Client) (*minio.Client, string, error) {
	var s3secret corev1.Secret
	err := seedClient.Get(ctx, secretNamespacedName, &s3secret)
	if err != nil {
		return nil, "", err
	}

	s3endpoint := string(s3secret.Data[Endpoint])
	s3accessKeyID := string(s3secret.Data[AccessKey])
	s3secretAccessKey := string(s3secret.Data[SecretKey])

	mc, err := s3.NewClient(s3endpoint, s3accessKeyID, s3secretAccessKey, nil)
	if err != nil {
		return nil, "", err
	}

	s3bucket := string(s3secret.Data[Bucket])

	return mc, s3bucket, nil
}

// swagger:parameters listMeteringReports
type listReportReq struct {
	// in: query
	StartAfter string `json:"start_after"`
	// in: query
	MaxKeys int `json:"max_keys"`
	// in: query
	ConfigurationName string `json:"configuration_name"`
}

// swagger:parameters getMeteringReport
type getReportReq struct {
	// in: path
	// required: true
	ReportName string `json:"report_name"`
	// in: query
	ConfigurationName string `json:"configuration_name"`
}

// swagger:parameters deleteMeteringReport
type deleteReportReq struct {
	// in: path
	// required: true
	ReportName string `json:"report_name"`
	// in: query
	ConfigurationName string `json:"configuration_name"`
}

func DecodeListMeteringReportReq(r *http.Request) (interface{}, error) {
	var req listReportReq

	maxKeys := r.URL.Query().Get("max_keys")

	if maxKeys == "" {
		req.MaxKeys = 1000
	} else {
		mK, err := strconv.Atoi(maxKeys)
		if err != nil {
			return nil, utilerrors.NewBadRequest("invalid value for `may_keys`")
		}
		req.MaxKeys = mK
	}

	req.StartAfter = r.URL.Query().Get("start_after")

	req.ConfigurationName = r.URL.Query().Get("configuration_name")

	return req, nil
}

func DecodeGetMeteringReportReq(r *http.Request) (interface{}, error) {
	var req getReportReq
	req.ReportName = mux.Vars(r)["report_name"]

	if req.ReportName == "" {
		return nil, utilerrors.NewBadRequest("`report_name` cannot be empty")
	}

	req.ConfigurationName = r.URL.Query().Get("configuration_name")

	return req, nil
}

func DecodeDeleteMeteringReportReq(r *http.Request) (interface{}, error) {
	var req deleteReportReq
	req.ReportName = mux.Vars(r)["report_name"]

	if req.ReportName == "" {
		return nil, utilerrors.NewBadRequest("`report_name` cannot be empty")
	}

	req.ConfigurationName = r.URL.Query().Get("configuration_name")

	return req, nil
}
