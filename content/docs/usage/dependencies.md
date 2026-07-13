---
title: "Dependencies"
linkTitle: "Dependencies"
weight: 15
description: >
  Declaring ordering constraints between Component objects
---

Component-operator supports declaring dependencies between `Component` objects via `spec.dependencies`. This allows you to express that a component must not be applied or deleted until other components have reached a certain state.

## Declaring Dependencies

```yaml
spec:
  dependencies:
    - name: database          # same namespace as this component
    - namespace: infra
      name: cert-manager      # cross-namespace reference
```

If `namespace` is omitted, the dependency is assumed to reside in the same namespace as the declaring component.

## Behavior During Apply

When a component has dependencies, it remains in `Pending` state until **all declared dependencies are in a `Ready` state**. Only then does reconciliation proceed and dependent objects are applied.

This guarantees that prerequisites (e.g., a database, a certificate authority, or an operator) are fully operational before workloads that depend on them are started.

## Behavior During Deletion — Reverse Order

Unlike Flux, where dependencies are only evaluated during creation and updates, **component-operator also honors dependencies during deletion, in reverse order**.

If component **A** declares a dependency on component **B**, then when **B** is being deleted, it will enter a `DeletionPending` state and wait until **A** has been fully deleted first. Only after all components that depend on B are gone will B proceed with its own deletion.

This ensures clean teardown: workloads are removed before their prerequisites, preventing errors caused by dependent services disappearing before the workloads that rely on them are shut down.

## Example

Given:

- Component `app` depends on `database`
- Component `database` depends on `storage`

**Apply order**: `storage` → `database` → `app`  
**Delete order**: `app` → `database` → `storage`

If `storage` is deleted while `app` and `database` still exist, `storage` will enter `DeletionPending` until both are gone.
