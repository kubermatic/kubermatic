# KKP Operator Scheduling Options

The KKP operator can be configured with various scheduling options to control where it runs in your Kubernetes cluster. This is particularly useful for production deployments where you need to ensure the operator runs on specific nodes or with specific scheduling constraints.

## Configuration

The scheduling options can be configured in the `values.yaml` file under the `kubermaticOperator` section:

```yaml
kubermaticOperator:
  # Node scheduling configuration
  tolerations: []
  # Example:
  # - key: "node-role.kubernetes.io/control-plane"
  #   operator: "Exists"
  #   effect: "NoSchedule"

  affinity: {}
  # Example:
  # nodeAffinity:
  #   preferredDuringSchedulingIgnoredDuringExecution:
  #   - weight: 100
  #     preference:
  #       matchExpressions:
  #       - key: node-role.kubernetes.io/control-plane
  #         operator: Exists

  nodeSelector: {}
  # Example:
  # kubernetes.io/os: linux
  # node-role.kubernetes.io/control-plane: ""
```

## Common Use Cases

### Running on Control Plane Nodes

To schedule the operator on control plane nodes:

```yaml
kubermaticOperator:
  tolerations:
    - key: "node-role.kubernetes.io/control-plane"
      operator: "Exists"
      effect: "NoSchedule"
  
  nodeSelector:
    node-role.kubernetes.io/control-plane: ""
```

### High Availability Setup

For a high availability setup with pod anti-affinity:

```yaml
kubermaticOperator:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
              - kubermatic-operator
          topologyKey: kubernetes.io/hostname
```

### Linux-Only Nodes

To ensure the operator only runs on Linux nodes:

```yaml
kubermaticOperator:
  nodeSelector:
    kubernetes.io/os: linux
```

## Best Practices

1. **Control Plane Placement**: Consider running the operator on control plane nodes for better resource isolation and security.

2. **High Availability**: Use pod anti-affinity to ensure operator pods are distributed across different nodes.

3. **Resource Requirements**: Ensure the nodes selected for the operator have sufficient resources to handle the operator's workload.

4. **Taints and Tolerations**: Use tolerations to allow the operator to run on nodes with specific taints, such as control plane nodes.

5. **Node Selectors**: Use node selectors to ensure the operator runs on nodes with specific characteristics, such as OS type or hardware capabilities. 