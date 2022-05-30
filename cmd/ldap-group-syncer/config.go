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

package main

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Address string `yaml:"address"`
	Mapping struct {
		Tagged  *TaggedConfig  `yaml:"tagged"`
		Grouped *GroupedConfig `yaml:"grouped"`
	} `yaml:"mapping"`
}

type TaggedConfig struct {
	BaseDN              string `yaml:"baseDN"`
	EmailAttribute      string `yaml:"emailAttribute"`
	GroupNameAttribute  string `yaml:"groupNameAttribute"`
	PersonNameAttribute string `yaml:"personNameAttribute"`
	GroupAttribute      string `yaml:"groupAttribute"`
	Query               string `yaml:"query"`
}

type GroupedConfig struct {
	BaseDN              string `yaml:"baseDN"`
	EmailAttribute      string `yaml:"emailAttribute"`
	GroupNameAttribute  string `yaml:"groupNameAttribute"`
	PersonNameAttribute string `yaml:"personNameAttribute"`
	MemberAttribute     string `yaml:"memberAttribute"`
	Query               string `yaml:"query"`
}

func loadConfig(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(content, config); err != nil {
		return nil, err
	}

	if config.Mapping.Grouped == nil && config.Mapping.Tagged == nil {
		return nil, errors.New("either tagged or grouped mapping must be configured")
	}

	if config.Mapping.Grouped != nil && config.Mapping.Tagged != nil {
		return nil, errors.New("tagged and grouped mapping must not be configured at the same time")
	}

	return config, nil
}
