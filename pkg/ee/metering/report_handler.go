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
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	minioCredentials "github.com/minio/minio-go/v7/pkg/credentials"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"

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
		return nil, k8cerrors.NewBadRequest("invalid request")
	}

	seedsMap, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	listOptions := minio.ListObjectsOptions{
		MaxKeys:    request.MaxKeys,
		StartAfter: request.StartAfter,
		Prefix:     ReportPrefix,
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
		return "", k8cerrors.NewBadRequest("invalid request")
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
		_, err = mc.StatObject(ctx, bucket, request.ReportName, minio.StatObjectOptions{})
		if err != nil {
			return "", err
		}

		presignedURL, err := mc.PresignedGetObject(ctx, bucket, request.ReportName, urlValidTime, nil)
		if err != nil {
			return "", err
		}

		urlString := presignedURL.String()
		return urlString, nil
	}

	return "", k8cerrors.New(404, "report not found")
}

func getReportsForSeed(ctx context.Context, options minio.ListObjectsOptions, seedClient ctrlruntimeclient.Client) ([]apiv1.MeteringReport, error) {
	mc, s3bucket, err := getS3DataFromSeed(ctx, seedClient)
	if err != nil {
		return nil, err
	}

	var reports []apiv1.MeteringReport

	mcCtx, cancel := context.WithCancel(context.Background())

	for report := range mc.ListObjects(mcCtx, s3bucket, options) {
		if report.Err != nil {
			cancel()
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
	cancel()

	return reports, nil
}

func getS3DataFromSeed(ctx context.Context, seedClient ctrlruntimeclient.Client) (*minio.Client, string, error) {
	var s3 corev1.Secret
	err := seedClient.Get(ctx, secretNamespacedName, &s3)
	if err != nil {
		return nil, "", err
	}

	s3endpoint := string(s3.Data[Endpoint])
	s3accessKeyID := string(s3.Data[AccessKey])
	s3secretAccessKey := string(s3.Data[SecretKey])

	secure := true

	if strings.Contains(s3endpoint, "https://") {
		s3endpoint = strings.Replace(s3endpoint, "https://", "", 1)
	} else if strings.Contains(s3endpoint, "http://") {
		s3endpoint = strings.Replace(s3endpoint, "http://", "", 1)
		secure = false
	}

	mc, err := minio.New(s3endpoint, &minio.Options{
		Creds:  minioCredentials.NewStaticV4(s3accessKeyID, s3secretAccessKey, ""),
		Secure: secure,
	})

	if err != nil {
		return nil, "", err
	}

	s3bucket := string(s3.Data[Bucket])

	return mc, s3bucket, nil
}

// swagger:parameters listMeteringReports
type listReportReq struct {
	// in: query
	StartAfter string `json:"start_after"`
	// in: query
	MaxKeys int `json:"max_keys"`
}

// swagger:parameters getMeteringReport
type getReportReq struct {
	// in: path
	// required: true
	ReportName string `json:"report_name"`
}

func DecodeListMeteringReportReq(r *http.Request) (interface{}, error) {
	var req listReportReq

	maxKeys := r.URL.Query().Get("max_keys")

	if maxKeys == "" {
		req.MaxKeys = 1000
	} else {
		mK, err := strconv.Atoi(maxKeys)
		if err != nil {
			return nil, k8cerrors.NewBadRequest("invalid value for `may_keys`")
		}
		req.MaxKeys = mK
	}

	req.StartAfter = r.URL.Query().Get("start_after")

	return req, nil
}

func DecodeGetMeteringReportReq(r *http.Request) (interface{}, error) {
	var req getReportReq
	req.ReportName = mux.Vars(r)["report_name"]

	if req.ReportName == "" {
		return nil, k8cerrors.NewBadRequest("`report_name` cannot be empty")
	}

	return req, nil
}
