package clusterbackup

import (
	"embed"
	_ "embed"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// embeddedFS is an embedded fs that contains velero CRD manifests
//
//go:embed static/*
var embeddedFS embed.FS

// CRDs returns a list of CRDs.
func CRDs() ([]apiextensionsv1.CustomResourceDefinition, error) {
	files, err := embeddedFS.ReadDir("static")
	if err != nil {
		return nil, err
	}

	result := []apiextensionsv1.CustomResourceDefinition{}

	for _, info := range files {
		crd, err := loadCRD(info.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to open CRD: %w", err)
		}
		result = append(result, *crd)
	}

	return result, nil
}

func loadCRD(filename string) (*apiextensionsv1.CustomResourceDefinition, error) {
	f, err := embeddedFS.Open("static/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	crd := &apiextensionsv1.CustomResourceDefinition{}
	dec := yaml.NewYAMLOrJSONDecoder(f, 1024)
	if err := dec.Decode(crd); err != nil {
		return nil, err
	}

	return crd, nil
}

// CRDReconciler returns a reconciler for a CRD.
func CRDReconciler(crd apiextensionsv1.CustomResourceDefinition) reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return crd.Name, func(target *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			target.Labels = crd.Labels
			target.Annotations = crd.Annotations
			target.Spec = crd.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			target.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return target, nil
		}
	}
}
