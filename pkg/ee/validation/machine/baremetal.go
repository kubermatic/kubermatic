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

package machine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"k8c.io/machine-controller/sdk/cloudprovider/baremetal"
	"k8c.io/machine-controller/sdk/cloudprovider/baremetal/plugins/tinkerbell"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/machine-controller/sdk/providerconfig/configvar"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// agentAttributesAnnotation is the annotation set on the Tinkerbell Hardware object by the
	// tink-agent after discovery. It holds a JSON document describing CPU/memory/block devices.
	agentAttributesAnnotation = "tinkerbell.org/agent-attributes"

	// tinkKubeconfigEnvVar is the environment variable KKP injects with the Tinkerbell
	// kubeconfig (see pkg/resources/data.go).
	tinkKubeconfigEnvVar = "TINK_KUBECONFIG"

	hardwareGVKGroup   = "tinkerbell.org"
	hardwareGVKVersion = "v1alpha1"
	hardwareGVKKind    = "Hardware"
)

// getBaremetalResourceRequirements computes the CPU/memory/storage usage of a baremetal
// (Tinkerbell) machine by reading the backing Hardware object. It is best-effort: any failure
// degrades to zero rather than returning an error, so machine creation is never blocked by a
// resource-quota side effect.
func getBaremetalResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := baremetal.GetConfig(*config)
	if err != nil {
		// Malformed baremetal spec: cannot even identify the hardware. Degrade to zero.
		return getZeroResourceDetails(), nil
	}

	var spec tinkerbell.TinkerbellPluginSpec
	if err := json.Unmarshal(rawConfig.DriverSpec.Raw, &spec); err != nil {
		return getZeroResourceDetails(), nil
	}

	kubeconfig, err := resolveTinkerbellKubeconfig(ctx, userClient, spec.Auth.Kubeconfig)
	if err != nil {
		// No inline value and no resolvable TINK_KUBECONFIG source.
		return getZeroResourceDetails(), nil
	}

	client, err := buildTinkerbellClient(kubeconfig)
	if err != nil {
		return getZeroResourceDetails(), nil
	}

	details, err := getBaremetalResourceDetailsFromCluster(ctx, client, spec.HardwareRef)
	if err != nil {
		// Hardware not found / unreachable cluster / unparseable attributes.
		return getZeroResourceDetails(), nil
	}

	return details, nil
}

// resolveTinkerbellKubeconfig replicates machine-controller's GetConfig kubeconfig resolution:
// an inline value must be base64-decoded; otherwise it resolves via the ConfigVar (secret /
// configmap) or the TINK_KUBECONFIG env var, again base64-decoding while tolerating unencoded
// YAML/JSON.
func resolveTinkerbellKubeconfig(ctx context.Context, userClient ctrlruntimeclient.Client, kubeconfigVar providerconfig.ConfigVarString) (string, error) {
	if kubeconfigVar.Value != "" {
		val, err := base64.StdEncoding.DecodeString(kubeconfigVar.Value)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 encoded kubeconfig: %w", err)
		}
		return string(val), nil
	}

	resolver := configvar.NewResolver(ctx, userClient)
	kubeconfig, err := resolver.GetStringValueOrEnv(kubeconfigVar, tinkKubeconfigEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get value of \"kubeconfig\" field: %w", err)
	}

	// We intentionally ignore the decode error with the assumption that an unencoded YAML or
	// JSON kubeconfig may have been passed in.
	if val, err := base64.StdEncoding.DecodeString(kubeconfig); err == nil {
		kubeconfig = string(val)
	}

	return kubeconfig, nil
}

// buildTinkerbellClient constructs a controller-runtime client for the Tinkerbell cluster.
func buildTinkerbellClient(kubeconfig string) (ctrlruntimeclient.Client, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	client, err := ctrlruntimeclient.New(restConfig, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return client, nil
}

// getBaremetalResourceDetailsFromCluster fetches the Hardware object referenced by hardwareRef
// and parses its resource usage. The client is injectable so tests can pass a fake client.
func getBaremetalResourceDetailsFromCluster(ctx context.Context, client ctrlruntimeclient.Client, hardwareRef types.NamespacedName) (*ResourceDetails, error) {
	hardware := &unstructured.Unstructured{}
	hardware.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   hardwareGVKGroup,
		Version: hardwareGVKVersion,
		Kind:    hardwareGVKKind,
	})

	if err := client.Get(ctx, hardwareRef, hardware); err != nil {
		return nil, fmt.Errorf("failed to get Tinkerbell Hardware %s: %w", hardwareRef, err)
	}

	return resourceDetailsFromHardware(hardware)
}

// resourceDetailsFromHardware extracts CPU/memory/storage from a Tinkerbell Hardware object.
// It prefers the tinkerbell.org/agent-attributes annotation; otherwise it falls back to
// spec.resources. Storage prefers spec.resources["disk"], else the sum of blockDevices[].size.
func resourceDetailsFromHardware(hardware *unstructured.Unstructured) (*ResourceDetails, error) {
	annotations := hardware.GetAnnotations()
	if raw, ok := annotations[agentAttributesAnnotation]; ok && raw != "" {
		return resourceDetailsFromAgentAttributes([]byte(raw))
	}

	// Fallback: spec.resources (map of quantity strings).
	resources, found, err := unstructured.NestedStringMap(hardware.Object, "spec", "resources")
	if err != nil || !found {
		return nil, fmt.Errorf("hardware has no resource information")
	}

	return resourceDetailsFromResourceList(resources), nil
}

// resourceDetailsFromAgentAttributes parses the agent-attributes JSON into quantities.
func resourceDetailsFromAgentAttributes(raw []byte) (*ResourceDetails, error) {
	var attrs agentAttributes
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, fmt.Errorf("failed to parse agent-attributes: %w", err)
	}

	cpu := *resource.NewQuantity(int64(attrs.CPU.TotalCores), resource.DecimalSI)

	var mem resource.Quantity
	if attrs.Memory.Total != "" {
		var err error
		mem, err = resource.ParseQuantity(attrs.Memory.Total)
		if err != nil {
			return nil, fmt.Errorf("failed to parse memory quantity %q: %w", attrs.Memory.Total, err)
		}
	}

	storage := resource.Quantity{}
	for _, blockDevice := range attrs.BlockDevices {
		if blockDevice.Size == "" {
			continue
		}
		size, err := resource.ParseQuantity(blockDevice.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to parse block device size %q: %w", blockDevice.Size, err)
		}
		storage.Add(size)
	}

	return NewResourceDetails(cpu, mem, storage), nil
}

// resourceDetailsFromResourceList builds ResourceDetails from a map of quantity strings with
// keys cpu/memory/disk.
func resourceDetailsFromResourceList(resources map[string]string) *ResourceDetails {
	cpu, _ := resource.ParseQuantity(resources["cpu"])
	mem, _ := resource.ParseQuantity(resources["memory"])
	storage, _ := resource.ParseQuantity(resources["disk"])
	return NewResourceDetails(cpu, mem, storage)
}

// agentAttributes mirrors the JSON document in the tinkerbell.org/agent-attributes annotation.
type agentAttributes struct {
	CPU          agentCPU           `json:"cpu"`
	Memory       agentMemory        `json:"memory"`
	BlockDevices []agentBlockDevice `json:"blockDevices"`
}

type agentCPU struct {
	TotalCores int `json:"totalCores"`
}

type agentMemory struct {
	Total string `json:"total"`
}

type agentBlockDevice struct {
	Size string `json:"size"`
}
