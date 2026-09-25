/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"strconv"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	supportedKeyAlgorithms = sets.New(kubermaticv1.KeyAlgorithmRSA, kubermaticv1.KeyAlgorithmECDSA)
	supportedRSAKeySizes   = sets.New[int32](2048, 3072, 4096)
	supportedECDSACurves   = sets.New(kubermaticv1.ECDSACurveP256, kubermaticv1.ECDSACurveP384)
)

func supportedRSAKeySizeNames() []string {
	names := []string{}
	for _, size := range sets.List(supportedRSAKeySizes) {
		names = append(names, strconv.Itoa(int(size)))
	}
	return names
}

// ValidateKeyConfiguration validates a KeyConfiguration, wherever it is configured.
func ValidateKeyConfiguration(config *kubermaticv1.KeyConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if config == nil {
		return allErrs
	}

	allErrs = append(allErrs, validateKeySpec(config.ServiceAccountKey, fldPath.Child("serviceAccountKey"))...)
	allErrs = append(allErrs, validateKeySpec(config.Certificates, fldPath.Child("certificates"))...)

	return allErrs
}

// validateKeySpec rejects combinations that the CRD enums cannot express, in
// particular settings that belong to the algorithm which was not selected. Such
// a spec would otherwise be silently ignored, and an operator who mistyped the
// algorithm would end up with key material they did not ask for.
func validateKeySpec(spec *kubermaticv1.KeySpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec == nil {
		return allErrs
	}

	if spec.Algorithm != "" && !supportedKeyAlgorithms.Has(spec.Algorithm) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("algorithm"), spec.Algorithm, sets.List(supportedKeyAlgorithms)))
	}

	if spec.RSAKeySize != 0 && !supportedRSAKeySizes.Has(spec.RSAKeySize) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("rsaKeySize"), spec.RSAKeySize, supportedRSAKeySizeNames()))
	}

	if spec.ECDSACurve != "" && !supportedECDSACurves.Has(spec.ECDSACurve) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("ecdsaCurve"), spec.ECDSACurve, sets.List(supportedECDSACurves)))
	}

	switch spec.Algorithm {
	case kubermaticv1.KeyAlgorithmECDSA:
		if spec.RSAKeySize != 0 {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("rsaKeySize"), "must not be set when algorithm is ECDSA"))
		}

	case kubermaticv1.KeyAlgorithmRSA, "":
		if spec.ECDSACurve != "" {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("ecdsaCurve"), "must not be set when algorithm is RSA"))
		}
	}

	return allErrs
}

// ValidateKeyConfigurationUpdate enforces that the key configuration a cluster
// was created with never changes.
//
// The key material of a running cluster cannot follow such a change: CAs are
// never replaced once they exist, while leaf certificates are re-issued when
// they approach expiry. A cluster whose configuration changed would therefore
// drift into a chain whose leaves use a different algorithm than the CA they
// were issued for, without anyone acting on it and without an error anywhere.
//
// Changing the key material of an existing cluster therefore requires an
// explicit rotation, which KKP does not implement yet: rotating a CA means
// re-issuing every leaf certificate, every internal kubeconfig, every kubelet
// client certificate and every webhook CA bundle, in an order that keeps the
// control plane reachable throughout. Until that exists, the only way to move a
// cluster to different key material is to recreate it.
//
// This lives in its own function so that a rotation feature can deliberately
// allow the change for the one update that starts the rotation.
func ValidateKeyConfigurationUpdate(oldConfig, newConfig *kubermaticv1.KeyConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if oldConfig != nil {
		return append(allErrs, apimachineryvalidation.ValidateImmutableField(newConfig, oldConfig, fldPath)...)
	}

	// A cluster created without the field already has RSA-2048 key material that
	// cannot be changed retroactively, so accepting a value now would only apply
	// to whatever is generated next -- the mixed chain described above.
	if newConfig != nil {
		allErrs = append(allErrs, field.Forbidden(fldPath, "key configuration cannot be added to an existing cluster; its key material was generated as RSA-2048 and rotating it is not supported yet, so the algorithm can only be chosen when a cluster is created"))
	}

	return allErrs
}
