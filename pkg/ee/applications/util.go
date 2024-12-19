/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

package applications

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"gopkg.in/yaml.v3"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubernetesutil "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNoClusterForTemplating = fmt.Errorf("no cluster found for template data")

// TemplateData is the root context injected into each application values.
type TemplateData struct {
	Cluster ClusterData
}

// ClusterData contains data related to the user cluster
// the application is rendered for.
type ClusterData struct {
	Name string
	// HumanReadableName is the user-specified cluster name.
	HumanReadableName string
	// OwnerEmail is the owner's e-mail address.
	OwnerEmail string
	// ClusterAddress stores access and address information of a cluster.
	Address kubermaticv1.ClusterAddress
	// Version is the exact current cluster version.
	Version string
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor" on the
	// current cluster version.
	MajorMinorVersion string
}

// GetTemplateData fetches the related cluster object by the given cluster namespace, parses pre defined values to a template data struct.
func GetTemplateData(ctx context.Context, seedClient ctrlruntimeclient.Client, clusterNamespace string) (*TemplateData, error) {
	cluster, err := kubernetesutil.ClusterFromNamespace(ctx, seedClient, clusterNamespace)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, ErrNoClusterForTemplating
	}
	var clusterVersion *semverlib.Version
	if s := cluster.Status.Versions.ControlPlane.Semver(); s != nil {
		clusterVersion = s
	} else {
		clusterVersion = cluster.Spec.Version.Semver()
	}
	return &TemplateData{
		Cluster: ClusterData{
			Name:              cluster.Name,
			HumanReadableName: cluster.Spec.HumanReadableName,
			OwnerEmail:        cluster.Status.UserEmail,
			Address:           cluster.Status.Address,
			Version:           fmt.Sprintf("%d.%d.%d", clusterVersion.Major(), clusterVersion.Minor(), clusterVersion.Patch()),
			MajorMinorVersion: fmt.Sprintf("%d.%d", clusterVersion.Major(), clusterVersion.Minor()),
		},
	}, nil
}

// RenderValueTemplate is rendering the given template data into the given map of values an error is returned when undefined values are used in the values map values.
func RenderValueTemplate(applicationValues map[string]interface{}, templateData *TemplateData) (map[string]interface{}, error) {
	yamlData, err := yaml.Marshal(applicationValues)
	if err != nil {
		return nil, err
	}

	parser := template.New("application-pre-defined-values")
	parsed, err := parser.Parse(string(yamlData))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err := parsed.Execute(&buffer, templateData); err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	parsedMap := make(map[string]interface{})
	err = yaml.Unmarshal(buffer.Bytes(), parsedMap)
	if err != nil {
		return nil, err
	}
	return parsedMap, nil
}
