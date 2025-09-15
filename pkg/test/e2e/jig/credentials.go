/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package jig

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

type CommonCredentials struct {
	KKPDatacenter string
}

func env(key string) (string, error) {
	value := os.Getenv(key)
	if len(value) == 0 {
		return "", fmt.Errorf("no %s environment variable defined", key)
	}

	return value, nil
}

type AlibabaCredentials struct {
	CommonCredentials

	AccessKeyID     string
	AccessKeySecret string
}

func (c *AlibabaCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "alibaba-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for Alibaba clusters")
}

func (c *AlibabaCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -alibaba-kkp-datacenter flag given")
	}

	if c.AccessKeyID, err = env("ALIBABA_ACCESS_KEY_ID"); err != nil {
		return err
	}

	if c.AccessKeySecret, err = env("ALIBABA_ACCESS_KEY_SECRET"); err != nil {
		return err
	}

	return nil
}

type AWSCredentials struct {
	CommonCredentials

	AccessKeyID     string
	SecretAccessKey string
}

func (c *AWSCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "aws-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for AWS clusters")
}

func (c *AWSCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -aws-kkp-datacenter flag given")
	}

	if c.AccessKeyID, err = env("AWS_E2E_TESTS_KEY_ID"); err != nil {
		return err
	}

	if c.SecretAccessKey, err = env("AWS_E2E_TESTS_SECRET"); err != nil {
		return err
	}

	return nil
}

type HetznerCredentials struct {
	CommonCredentials

	Token string
}

func (c *HetznerCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "hetzner-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for Hetzner clusters")
}

func (c *HetznerCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -hhetzner-kkp-datacenter flag given")
	}

	if c.Token, err = env("HCLOUD_TOKEN"); err != nil {
		if c.Token, err = env("HZ_TOKEN"); err != nil {
			if c.Token, err = env("HZ_E2E_TOKEN"); err != nil {
				return errors.New("no HCLOUD_TOKEN, HZ_TOKEN or HZ_E2E_TOKEN environment variable defined")
			}
		}
	}

	return nil
}

type AzureCredentials struct {
	CommonCredentials

	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
}

func (c *AzureCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "azure-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for Azure clusters")
}

func (c *AzureCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -azure-kkp-datacenter flag given")
	}

	if c.TenantID, err = env("AZURE_E2E_TESTS_TENANT_ID"); err != nil {
		return err
	}

	if c.SubscriptionID, err = env("AZURE_E2E_TESTS_SUBSCRIPTION_ID"); err != nil {
		return err
	}

	if c.ClientID, err = env("AZURE_E2E_TESTS_CLIENT_ID"); err != nil {
		return err
	}

	if c.ClientSecret, err = env("AZURE_E2E_TESTS_CLIENT_SECRET"); err != nil {
		return err
	}

	return nil
}

type OpenstackCredentials struct {
	CommonCredentials

	Username       string
	Password       string
	Tenant         string
	Domain         string
	FloatingIPPool string
	Network        string
}

func (c *OpenstackCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "openstack-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for Openstack clusters")
}

func (c *OpenstackCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -openstack-kkp-datacenter flag given")
	}

	if c.Username, err = env("OS_USERNAME"); err != nil {
		return err
	}

	if c.Password, err = env("OS_PASSWORD"); err != nil {
		return err
	}

	if c.Tenant, err = env("OS_TENANT_NAME"); err != nil {
		return err
	}

	if c.Domain, err = env("OS_DOMAIN"); err != nil {
		return err
	}

	if c.FloatingIPPool, err = env("OS_FLOATING_IP_POOL"); err != nil {
		return err
	}

	c.Network, _ = env("OS_NETWORK_NAME")

	return nil
}

type VSphereCredentials struct {
	CommonCredentials

	Username string
	Password string
}

func (c *VSphereCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "vsphere-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for vSphere clusters")
}

func (c *VSphereCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -vsphere-kkp-datacenter flag given")
	}

	if c.Username, err = env("VSPHERE_E2E_USERNAME"); err != nil {
		return err
	}

	if c.Password, err = env("VSPHERE_E2E_PASSWORD"); err != nil {
		return err
	}

	return nil
}

type GCPCredentials struct {
	CommonCredentials

	ServiceAccount string
}

func (c *GCPCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "gcp-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for GCP clusters")
}

func (c *GCPCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -gcp-kkp-datacenter flag given")
	}

	if c.ServiceAccount, err = env("GOOGLE_SERVICE_ACCOUNT"); err != nil {
		return err
	}

	return nil
}

type DigitaloceanCredentials struct {
	CommonCredentials

	Token string
}

func (c *DigitaloceanCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "digitalocean-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for Digitalocean clusters")
}

func (c *DigitaloceanCredentials) Parse() (err error) {
	if c.KKPDatacenter == "" {
		return errors.New("no -digitalocean-kkp-datacenter flag given")
	}

	if c.Token, err = env("DO_TOKEN"); err != nil {
		return err
	}

	return nil
}

type BYOCredentials struct {
	CommonCredentials
}

func (c *BYOCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "byo-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for BYO clusters")
}

func (c *BYOCredentials) Parse() error {
	if c.KKPDatacenter == "" {
		return errors.New("no -byo-kkp-datacenter flag given")
	}

	return nil
}
