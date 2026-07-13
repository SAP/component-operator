---
title: "Reconcile Modes"
linkTitle: "Reconcile Modes"
weight: 12
description: >
  Controlling how individual dependent objects are reconciled
---

By default, component-operator reconciles a dependent object whenever its generated manifest changes.

More details about drift detection can be found [here](../drift-detection).

## Reconcile Policy Values

This behavior can be customized per object using the annotation `component-operator.cs.sap.com/reconcile-policy`.

| Value | Description |
|-------|-------------|
| `on-object-change` (default) | The object is reconciled whenever its generated manifest changes. |
| `on-object-or-component-change` | The object is reconciled whenever its manifest changes, or whenever the component itself changes (as identified by a change in the component's generation). |
| `once` | The object is reconciled exactly once (on creation). It is never updated again by component-operator, regardless of changes to the manifest or the component. |

In the `on-object-or-component-change` case the object digest additionally incorporates a deep digest of the component.
In the `once` case, the object digest is set to `__once__`. Note that, when changing the effective reconcile policy to `once`, then one last reconcile happens, caused by the change of the policy.

## Usage

Add the annotation to the manifest of a specific dependent object:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/reconcile-policy: once
```

## When to Use Each Mode

- **`on-object-change`**: The default and recommended mode for most objects. Re-applies the object when its desired state changes.

- **`on-object-or-component-change`**: Useful when an object's behavior should be re-triggered whenever the parent component changes — for example, a `Job` that should re-run on every component update.

- **`once`**: Useful for objects that should be bootstrapped once and then handed off entirely to their controller or a human operator. Examples include initial `Secret` values or database seed jobs that must not be overwritten after first creation.
