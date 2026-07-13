---
title: "Command-Line Flags"
linkTitle: "Command-Line Flags"
weight: 1
description: >
  Reference for all controller manager command-line flags
---

The component-operator controller manager is configured via command-line flags at startup. All flags are optional unless noted otherwise.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-metrics-bind-address` | string | `:8080` | Address on which the Prometheus metrics endpoint is served. |
| `-health-probe-bind-address` | string | `:8081` | Address on which the liveness and readiness probe HTTP endpoints are served. |
| `-kubeconfig` | string | — | Path to a kubeconfig file. Only required when running out-of-cluster (e.g. for local development). In-cluster deployments use the pod's service account automatically. |
| `-leader-elect` | bool | `false` | Enable leader election. When running multiple replicas of the controller manager, enabling this ensures that only one instance is active at any time. Recommended for production deployments. |
| `-default-service-account` | string | — | Name of the service account to impersonate by default for all components that do not explicitly set `spec.serviceAccountName`. The account is resolved relative to each component's namespace. See [Impersonation and Remote Clusters](../../usage/impersonation). |
| `-events-address` | string | — | HTTP address of a Flux notification-controller events receiver. When set, component-operator streams reconciliation events (successful applies, errors, state transitions) to this endpoint. See [Notifications](../../usage/notifications). |
| `-max-concurrent-reconciles` | int | `5` | Maximum number of component reconciliations that may run in parallel. See [Performance and Sizing](../performance). |

## High Availability and Leader Election

For production deployments, run two or more replicas of the controller manager with `-leader-elect` enabled. Leader election ensures that only one instance actively reconciles at any time; the others remain on standby and take over automatically if the leader fails or is restarted.

This setup eliminates the single point of failure with no extra reconciliation load under normal conditions.

Example Kubernetes deployment snippet:

```yaml
args:
  - -leader-elect
  - -max-concurrent-reconciles=10
```

With leader election enabled, at least two replicas are recommended so that failover is immediate without requiring a pod restart.
