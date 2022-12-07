# Kubermatic Network Interface Manager
A linux based tool which can be used to create and manage dummy network interface.

## Overview
The network-interface-manager is used as an init and side car container by envoy-agent. It creates and manages the dummy interface required for envoy-agent for `Tunnelling` expose strategy.

## Usage
Create interface:

`network-interface-manager -mode init -if envoy -addr 10.10.10.3`

Monitor interface:

`network-interface-manager -mode probe -if envoy -addr 10.10.10.3`


## Release
The network-interface-manager gets automatically built in CI and it is built only for linux targets.