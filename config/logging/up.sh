#!/bin/bash

HELMOPTS=--tiller-namespace=kubermatic-installer

helm $HELMOPTS upgrade -i elasticsearch --namespace logging -f ./elasticsearch/values.yaml ./elasticsearch
helm $HELMOPTS upgrade -i fluentd --namespace logging -f ./fluentd/values.yaml ./fluentd
helm $HELMOPTS upgrade -i kibana --namespace logging -f ./kibana/values.yaml ./kibana
