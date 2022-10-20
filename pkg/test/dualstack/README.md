# Dualstack E2E Tests

## AWS Preconditions

To make the E2E tests work on AWS, the following steps need to be taken:

* See https://docs.kubermatic.com/kubermatic/main/tutorials-howtos/networking/dual-stack/#aws
* The VPC and Subnets must have IPv6 CIDRs assigned.
* The default route table needs to have an entry for the VPC's IPv6 CIDR.
* An egress-only internet gateway must be created.
