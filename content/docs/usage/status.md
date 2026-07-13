---
title: "Status Detection"
linkTitle: "Status Detection"
weight: 5
description: >
  How component-operator determines whether a dependent object is ready
---

Component-operator needs to determine whether each dependent object has reached a ready state before proceeding to the next [apply wave](../dependents) or completing the component reconciliation. The readiness check is based on the [kstatus](https://pkg.go.dev/sigs.k8s.io/cli-utils/pkg/kstatus/status) library, with additional tuning capabilities.

## How kstatus Works

For most resources, vanilla kstatus uses the following algorithm to determine if a Kubernetes object is ready (a special logic applies for some builtin types):

**Step 1 — observedGeneration check:**

```
Does the object have status.observedGeneration?
  → Yes: Does status.observedGeneration equal metadata.generation?
           → Yes: proceed to readyCondition check
           → No:  object is NOT ready (generation mismatch)
  → No:  proceed to readyCondition check
```

**Step 2 — readyCondition check:**

```
Does the object have status.conditions[type == "Ready"]?
  → Yes: Is condition.status == "True"?
           → Yes: object is READY
           → No (False or Unknown): object is NOT ready
  → No:  object is READY (absence of a Ready condition means "implicitly ready")
```

This logic works well for controllers that set `status.observedGeneration` and `status.conditions` reliably. In particular it is crucial that 
- objects are born with a `status.observedGeneration: -1` (or some other impossible value)
- on each reconcile iteration, `status.observedGeneration` and `status.conditions` are updated by the responsible controller.

However, some controllers set these fields lazily or not at all, which can cause problems:

- A controller may not immediately set `status.observedGeneration` after an object is created, leaving it absent for a brief period. kstatus would then skip the generation check and might incorrectly conclude the object is ready.
- A controller may never set a `Ready` condition, even though it uses other condition types to indicate readiness.
- A controller may set `status.observedGeneration` but only update it lazily, creating a window where the object appears unready even though the controller has already processed the current generation.

## Tuning Status Detection with Annotations

To handle these cases, the annotation `component-operator.cs.sap.com/status-hint` can be added to any rendered manifest. It accepts a comma-separated list of hints:

### `has-observed-generation`

Tells component-operator to treat the object as having a `status.observedGeneration` field, even if it is not yet set by the controller. This is useful for controllers that set the field lazily: without this hint, the generation check would be skipped during the window where the field is absent, potentially causing a false-positive ready status.

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/status-hint: has-observed-generation
```

### `has-ready-condition`

Tells component-operator to require a `Ready` condition. If the condition is absent, the object is treated as having `status: Unknown` (i.e., not ready). This is useful for objects where the controller will eventually set a `Ready` condition but may not have done so yet at time of first check.

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/status-hint: has-ready-condition
```

### `conditions`

A semicolon-separated list of additional condition types that must all be present and have `status: True` for the object to be considered ready.

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/status-hint: "conditions=Synced;Healthy"
```

### Combining hints

Multiple hints can be combined as a comma-separated list:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/status-hint: "has-observed-generation,has-ready-condition"
```
