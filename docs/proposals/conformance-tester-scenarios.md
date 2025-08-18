<!--
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Conformance Tester Scenarios

**Author**: Soeren Henning (@soer3n)

**Status**: Draft proposal; Prototype in progress.

## Motivation and Background

The previous command-line-only approach for the conformance tester was very limiting. It was only possible to test a single scenario per provider, making it difficult to cover a wide variety of configurations and permutations of settings. A file-based approach allows for defining numerous scenarios, leading to better versioning, sharing, and management of test configurations.

This new functionality also enables two key use cases:
1.  **End-User Verification:** The conformance tester can be shipped to end-users, who can then use it to verify that their environment is correctly configured to work with Kubermatic.
2.  **Comprehensive QA:** The `generate` command allows the QA team to easily create a test suite that covers all possible provider settings, ensuring a high level of test coverage.

## Implementation proposal

The conformance tester now supports a `--scenarios-file` flag, which accepts a YAML file defining the scenarios to run.

### Scenario File Format

```yaml
versions:
  - "1.31"
  - "1.32"
scenarios:
- provider: "kubevirt"
  operatingSystem: "ubuntu"
  flavors:
  - name: "small"
    value:
      virtualMachine:
        template:
          cpus: "2"
          memory: "2Gi"
          primaryDisk:
            size: "20Gi"
            storageClassName: "local"
            osImage: "http://image-repo.kube-system.svc/images/ubuntu-22.04.qcow2"
- provider: "aws"
  operatingSystem: "ubuntu"
  # providers without flavors simply omit the list
```

Execution model:
- One cluster is created per scenario per version (provider + OS + version).
- All flavors in a scenario are realized as separate MachineDeployments in the same cluster.
- Providers that do not support flavors produce their default MachineDeployment(s).
- Minor versions are resolved to the latest supported patch before execution.

### Generate Command

A new `generate` subcommand has been added to the conformance tester. This command takes a template file and generates a `scenarios.yaml` file containing all possible permutations of the defined providers, operating systems, and provider-specific flavors. The `versions` are copied verbatim from the template into the top-level `versions:` list in the output.

#### Usage

```bash
conformance-tester generate --from <template.yaml> --to scenarios.yaml
```

#### Template File Format

The template file is a YAML file that defines the available options for each provider.

```yaml
providers:
- kubevirt
- hetzner

distributions:
- ubuntu
- rockylinux

versions:
- "1.31"
- "1.32"

kubevirt:
  virtualMachine:
    template:
      cpus: ["1", "2"]
      memory: ["2Gi", "4Gi"]
      primaryDisk:
        size: ["20Gi", "40Gi"]
        storageClassName: ["local"]
        osImage: ["http://image-repo.kube-system.svc/images/ubuntu-22.04.qcow2"]
```

## Implementation Details

- The generator recursively traverses the template and builds the Cartesian product for provider-specific flavor lists, creating a `flavors:` list per scenario.
- YAML keys are sanitized to strings to avoid invalid keys in the output.
- For KubeVirt, the image is expected at `virtualMachine.template.primaryDisk.osImage` and is also read from that path at runtime when applying a flavor.
- At runtime, versions from the scenario file take precedence. Each minor is resolved to the latest supported patch using the existing helper.
- The runner wraps each provider/OS/version scenario in a multi-flavor scenario, creating one MachineDeployment per flavor in a single cluster.

## Future Work

- Advanced filtering of scenarios and flavors via tags or expressions.
- CI/CD integration for dynamic generation.
- Optional per-flavor node counts or labels to drive differentiated workloads.

