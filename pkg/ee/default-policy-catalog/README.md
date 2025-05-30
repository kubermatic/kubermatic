# Default Policy Catalog

This directory contains the default policy catalog for Kubermatic. The policies are based on Kyverno policies and converted to Kubermatic's `PolicyTemplate` format.

## Directory Structure

- `policies/`: Contains all the Kyverno ClusterPolicy files.

## Updating the Policy Catalog

To update the policy catalog with the latest policies from the Kyverno GitHub repository, run the provided script:

```bash
# Fetch the latest policies using a specific version
./hack/update-kyverno-policies.sh release-1.14
```

The policies are then automatically converted to PolicyTemplates at runtime when loaded by the application.

### Adding Kyverno Policies

To add a custom policy:
1. Create a new YAML file in the `policies/` directory
2. Follow the PolicyTemplate format as shown in existing templates
3. Ensure the policy has a unique name

## PolicyTemplate Format

When Kyverno ClusterPolicies are loaded, they are converted to PolicyTemplates with this format:

```yaml
apiVersion: kubermatic.k8c.io/v1
kind: PolicyTemplate
metadata:
  name: [policy-name]
spec:
  title: [From policies.kyverno.io/title annotation]
  description: [From policies.kyverno.io/description annotation]
  category: [From policies.kyverno.io/category annotation]
  severity: [From policies.kyverno.io/severity annotation]
  visibility: Global
  policySpec:
    #  Kyverno policy spec
    validationFailureAction: [Audit/Enforce]
    background: [true/false]
    rules:
    - name: [rule-name]
      # Rule definition
``` 