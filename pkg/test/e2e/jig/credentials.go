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

func env(key string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		panic(fmt.Sprintf("No %s environment variable set.", key))
	}

	return value
}

type AWSCredentials struct {
	CommonCredentials

	AccessKeyID     string
	SecretAccessKey string
}

func (c *AWSCredentials) AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&c.KKPDatacenter, "aws-kkp-datacenter", c.KKPDatacenter, "KKP datacenter to use for AWS clusters")
}

func (c *AWSCredentials) Parse() error {
	if c.KKPDatacenter == "" {
		return errors.New("no -aws-kkp-datacenter flag given")
	}

	c.AccessKeyID = env("AWS_E2E_TESTS_KEY_ID")
	if c.AccessKeyID == "" {
		return errors.New("no AWS_E2E_TESTS_KEY_ID environment variable defined")
	}

	c.SecretAccessKey = env("AWS_E2E_TESTS_SECRET")
	if c.SecretAccessKey == "" {
		return errors.New("no AWS_E2E_TESTS_SECRET environment variable defined")
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

func (c *HetznerCredentials) Parse() error {
	if c.KKPDatacenter == "" {
		return errors.New("no -hhetzner-kkp-datacenter flag given")
	}

	c.Token = os.Getenv("HCLOUD_TOKEN")
	if len(c.Token) == 0 {
		c.Token = os.Getenv("HZ_TOKEN")
		if len(c.Token) == 0 {
			c.Token = os.Getenv("HZ_E2E_TOKEN")
			if len(c.Token) == 0 {
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

func (c *AzureCredentials) Parse() error {
	if c.KKPDatacenter == "" {
		return errors.New("no -azure-kkp-datacenter flag given")
	}

	c.TenantID = env("AZURE_E2E_TESTS_TENANT_ID")
	if c.TenantID == "" {
		return errors.New("no AZURE_E2E_TESTS_TENANT_ID environment variable defined")
	}

	c.SubscriptionID = env("AZURE_E2E_TESTS_SUBSCRIPTION_ID")
	if c.SubscriptionID == "" {
		return errors.New("no AZURE_E2E_TESTS_SUBSCRIPTION_ID environment variable defined")
	}

	c.ClientID = env("AZURE_E2E_TESTS_CLIENT_ID")
	if c.ClientID == "" {
		return errors.New("no AZURE_E2E_TESTS_CLIENT_ID environment variable defined")
	}

	c.ClientSecret = env("AZURE_E2E_TESTS_CLIENT_SECRET")
	if c.ClientSecret == "" {
		return errors.New("no AZURE_E2E_TESTS_CLIENT_SECRET environment variable defined")
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
