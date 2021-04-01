/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestProjectReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*projectReconciler, *httptest.Server) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())

	reconciler := projectReconciler{
		Client:        dynamicClient,
		grafanaClient: grafanaClient,
		log:           kubermaticlog.Logger,
		recorder:      record.NewFakeRecorder(10),
	}
	return &reconciler, ts
}

func TestProjectReconcile(t *testing.T) {
	type request struct {
		name     string
		code     int
		method   string
		path     string
		body     string
		response string
	}
	testCases := []struct {
		name         string
		project      *kubermaticv1.Project
		handlerFunc  http.HandlerFunc
		requests     []request
		hasFinalizer bool
		err          bool
	}{
		{
			name: "create org for project",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "create",
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create",
					code:     200,
					method:   "POST",
					path:     "/api/orgs",
					body:     `{"id":0,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
					response: `{"message": "org created", "OrgID": 1}`,
				},
			},
		},
		{
			name: "create org for project - failed",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "create",
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			hasFinalizer: true,
			err:          true,
		},
		{
			name: "create org for project - org already exists",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "create",
					Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			requests: []request{
				{
					name:     "get org by name",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/name/projectName-create",
					response: `{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
				},
				{
					name:     "create",
					code:     409,
					method:   "POST",
					path:     "/api/orgs",
					body:     `{"id":0,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
					response: `{"message": "name already taken"}`,
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "update org for project",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "create",
					Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1",
					response: `{"id":1,"name":"projectName-oldname","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
				},
				{
					name:     "update",
					code:     200,
					method:   "PUT",
					path:     "/api/orgs/1",
					body:     `{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
					response: `{"message": "name already taken"}`,
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "delete org for project",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "delete",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Annotations:       map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			hasFinalizer: false,
			requests: []request{
				{
					name:     "delete org by id",
					code:     200,
					method:   "DELETE",
					path:     "/api/orgs/1",
					response: `{}`,
				},
			},
		},
		{
			name: "delete org for project without annotation",
			project: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "delete",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{mlaFinalizer},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			},
			hasFinalizer: false,
			err:          false,
			requests:     []request{},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handled := 0
			r := mux.NewRouter()
			for _, tcRequest := range tc.requests {
				tcRequest := tcRequest
				r.HandleFunc(tcRequest.path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tcRequest.body != "" {
						defer r.Body.Close()
						decoder := json.NewDecoder(r.Body)
						reqMap := map[string]interface{}{}
						if err := decoder.Decode(&reqMap); err != nil {
							t.Fatalf("%s: unmarshal request failed: %v", tcRequest.name, err)
						}
						tcMap := map[string]interface{}{}
						if err := json.Unmarshal([]byte(tcRequest.body), &tcMap); err != nil {
							t.Fatalf("%s: unmarshal expected map failed: %v", tcRequest.name, err)
						}
						assert.Equal(t, reqMap, tcMap)
					}
					w.WriteHeader(tcRequest.code)
					fmt.Fprint(w, tcRequest.response)
					handled++
				})).Methods(tcRequest.method)
			}
			ctx := context.Background()
			objects := []ctrlruntimeclient.Object{tc.project}
			controller, server := newTestProjectReconciler(t, objects, r)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.project.Name}}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			project := &kubermaticv1.Project{}
			if err := controller.Get(ctx, request.NamespacedName, project); err != nil {
				t.Fatalf("unable to get project: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(project, mlaFinalizer))
			assert.Equal(t, handled, len(tc.requests))
			server.Close()
		})
	}

}
