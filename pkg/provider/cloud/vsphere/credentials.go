/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"context"
	"crypto/x509"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

// Precedence if not infraManagementUser:
// * User from cluster
// * User from Secret
// Precedence if infraManagementUser:
// * User from clusters infraManagementUser
// * User from cluster
// * User form clusters secret infraManagementUser
// * User from clusters secret.
func getUsernameAndPassword(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, infraManagementUser bool) (username, password string, err error) {
	if infraManagementUser {
		username = cloud.VSphere.InfraManagementUser.Username
		password = cloud.VSphere.InfraManagementUser.Password
	}
	if username == "" {
		username = cloud.VSphere.Username
	}
	if password == "" {
		password = cloud.VSphere.Password
	}

	if username != "" && password != "" {
		return username, password, nil
	}

	if cloud.VSphere.CredentialsReference == nil {
		return "", "", errors.New("cluster contains no password and an empty credentialsReference")
	}

	if username == "" && infraManagementUser {
		username, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereInfraManagementUserUsername)
		if err != nil {
			return "", "", err
		}
	}
	if username == "" {
		username, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereUsername)
		if err != nil {
			return "", "", err
		}
	}

	if password == "" && infraManagementUser {
		password, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereInfraManagementUserPassword)
		if err != nil {
			return "", "", err
		}
	}

	if password == "" {
		password, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VspherePassword)
		if err != nil {
			return "", "", err
		}
	}

	if username == "" {
		return "", "", errors.New("unable to get username")
	}

	if password == "" {
		return "", "", errors.New("unable to get password")
	}

	return username, password, nil
}

func getCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, dc *kubermaticv1.DatacenterSpecVSphere) (string, string, error) {
	var username, password string
	var err error

	// InfraManagementUser from Datacenter
	if dc != nil && dc.InfraManagementUser != nil {
		if dc.InfraManagementUser.Username != "" && dc.InfraManagementUser.Password != "" {
			return dc.InfraManagementUser.Username, dc.InfraManagementUser.Password, nil
		}
	}

	// InfraManagementUser from Cluster
	username, password, err = getUsernameAndPassword(cloud, secretKeySelector, true)
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

// ValidateCredentials allows to verify username and password for a specific vSphere datacenter. It does so by attempting
// to create a new session and finding the default folder.
func ValidateCredentials(ctx context.Context, dc *kubermaticv1.DatacenterSpecVSphere, username, password string, caBundle *x509.CertPool) error {
	session, err := newSession(ctx, dc, username, password, caBundle)
	if err != nil {
		return err
	}
	defer session.Logout(ctx)

	_, err = session.Finder.DefaultFolder(ctx)

	return err
}
