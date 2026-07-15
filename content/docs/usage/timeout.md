---
title: "Timeout and Stickiness"
linkTitle: "Timeout and Stickiness"
weight: 14
description: >
  Configuring reconciliation timeouts and source revision stickiness
---

## Timeout

The field `spec.timeout` defines how long dependent objects are expected to take to reach a ready state after being applied. If not all dependents are ready within this period, the component's state transitions from `Processing` to `Error`.

```yaml
spec:
  timeout: 10m
```

If `spec.timeout` is not set, it defaults to the effective requeue interval.

**Important:** reaching the timeout does not stop reconciliation. Component-operator continues to attempt reconciliation at the normal requeue interval. The timeout only affects the reported state. Whenever the component itself or any of its referenced objects (e.g., referenced secrets) change, the timeout countdown resets.

## Requeue and Retry Intervals

- `spec.requeueInterval`: period between re-reconciliations after success (default: 10 minutes).
- `spec.retryInterval`: period between re-reconciliations after a retriable error (default: equals the effective requeue interval). For details on what constitutes a retriable error, see the [component-operator-runtime documentation](https://sap.github.io/component-operator-runtime/docs/concepts/reconciler/#tuning-the-retry-behavior). In case of non-retriable errors, the default controller-runtime error backoff applies.

## Stickiness

By default, component-operator always reconciles towards the **latest** available state of the referenced source. If the source changes (e.g., a new commit is pushed to a GitRepository), the operator immediately starts reconciling against that new revision.

Setting `spec.sticky: true` changes this behavior. When sticky mode is enabled and the source revision changes, the operator locks onto the current revision and keeps reconciling it until the component reaches a `Ready` state or the `spec.timeout` is exceeded. Only then will it move on to the latest available revision. Any intermediate revisions published while the operator is stuck on the current one are skipped.

```yaml
spec:
  sticky: true
  timeout: 15m
```

Stickiness is useful in environments where you want to ensure that every source revision is fully and successfully deployed before moving to the next one, rather than racing ahead to the latest revision. This can prevent situations where a partially-applied intermediate state is skipped over and never properly cleaned up.
