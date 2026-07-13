---
title: "Helm Chart"
linkTitle: "Helm Chart"
weight: 3
description: >
  Installing component-operator using the official Helm chart
---

The recommended way to install component-operator is via the official Helm chart. The chart source is hosted at [`github.com/SAP/component-operator/tree/main/chart`](https://github.com/SAP/component-operator/tree/main/chart) and published to the GitHub OCI registry.

## Installation

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system
```

To pin to a specific version:

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --version <version> \
  --namespace flux-system
```

Refer to the [chart documentation](https://github.com/SAP/component-operator/tree/main/chart) for the full list of available Helm values.

Note: if you want to deploy component-operator into a different namespace (instead of flux-system), then you have to allow the component-operator pod to access the source-controller running flux-system.

The easy way is to install flux with the `--network-policy=false` option. However this is not recommended because it allows all workloads in the cluster to access flux-system. Better is to create an explicit network policy, such as

```yaml
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-component-operator-system
  namespace: flux-system
spec:
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector: {}
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: component-operator-system
```

## Further Configuration

The controller manager behaviour can be tuned after installation — for example, setting a default service account, enabling leader election, or adjusting reconciliation concurrency. See the [Configuration](../../configuration) section for details.
