---
title: "Intervals and Timing"
linkTitle: "Intervals and Timing"
weight: 10
description: >
  Tuning reconciliation timing
---

Reconciliation timing is about
- how often a component is reconciled in success state
- how often a component is reconciled in various error states
- how often a component or single dependent object is force-reapplied to the Kubernetes cluster
- temporarily disabling (suspending) reconciliation of a component.

## Reconciliation Intervals

### Requeue Interval

`spec.requeueInterval` defines how often a component is re-reconciled after a successful reconciliation.

Requeues are not particularly expensive, but depending on the number of components in a cluster it might be necessary to increase the requeue interval.

Defaults to 10 minutes.

```yaml
spec:
  requeueInterval: 15m
```

### Retry Interval

`spec.retryInterval` defines how often reconciliation is retried after a retriable error.

Retriable errors are thrown in various situations. For example, if a referenced secret or source object does not exist, or a depended component is not ready.

Defaults to the effective requeue interval.

```yaml
spec:
  retryInterval: 2m
```

### Reapply Interval

By default, component-operator force-reapplies all dependent objects to the Kubernetes API every 60 minutes, even if they appear to be in sync. More details about drift detection can be found [here](../drift-detection). Forced reapply can help to correct drifts caused by out-of-band changes (e.g., manual `kubectl edit`).

The reapply interval can be set on component level as `spec.reapplyInterval`

```yaml
spec:
  reapplyInterval: 2h
```

and can be overridden on object level by adding the annotation `component-operator.cs.sap.com/reapply-interval` to the object manifest.

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/reapply-interval: 30m
```

The value should be greater than the effective requeue interval; otherwise the reapply may be limited by the requeue frequency.

## Suspension

Setting `spec.suspend: true` suspends all reconciliation of the component. The component enters a `Pending` state with reason `Suspended`. Deletion is not affected — a suspended component still processes deletion normally.

```yaml
spec:
  suspend: true
```