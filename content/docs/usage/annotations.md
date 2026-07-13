---
title: "Dependents Annotations"
linkTitle: "Dependents Annotations"
weight: 8
description: >
  Overview of annotations that can be set with dependent resources
---

The following annotations can be added to any rendered manifest to control how component-operator handles that specific dependent object. Where a component-level default exists, it is noted in the **Component-level default** column.

| Annotation | Values | Component-level default | Description |
|------------|--------|------------------------|-------------|
| `component-operator.cs.sap.com/adoption-policy` | `if-unowned` (default), `never`, `always` | `spec.adoptionPolicy` (`IfUnowned`) | How to handle an object that already exists in the cluster. See [Ownership and Adoption](../ownership). |
| `component-operator.cs.sap.com/reconcile-policy` | `on-object-change` (default), `on-object-or-component-change`, `once` | — | When the object is reconciled. See [Reconcile Modes](../reconcile-modes). |
| `component-operator.cs.sap.com/update-policy` | `ssa-override` (default), `ssa-merge`, `replace`, `recreate` | `spec.updatePolicy` (`SsaOverride`) | How the object is updated in the Kubernetes API. See [Dependents Lifecycle](../dependents#update-policy). |
| `component-operator.cs.sap.com/delete-policy` | `delete` (default), `orphan`, `orphan-on-apply`, `orphan-on-delete` | `spec.deletePolicy` (`Delete`) | What happens to the object when it becomes redundant or the component is deleted. See [Dependents Lifecycle](../dependents#delete-policy). |
| `component-operator.cs.sap.com/apply-order` | integer (default: `0`) | — | The wave in which the object is applied. Lower numbers are applied first. See [Dependents Lifecycle](../dependents#apply-waves). |
| `component-operator.cs.sap.com/delete-order` | integer (default: `0`) | — | The wave in which the object is deleted. Lower numbers are deleted first. Independent of `apply-order`. See [Dependents Lifecycle](../dependents#delete-waves). |
| `component-operator.cs.sap.com/purge-order` | integer | — | The apply wave at the end of which the object is deleted (purged) from the cluster, while its inventory record is set to `Completed`. See [Dependents Lifecycle](../dependents#purge-orders). |
| `component-operator.cs.sap.com/reapply-interval` | duration (e.g. `30m`) | `spec.reapplyInterval` (`60m`) | How often the object is force-reapplied even when in sync. See [Drift Detection](../drift-detection). |
| `component-operator.cs.sap.com/status-hint` | comma-separated hints (see below) | — | Tuning hints for the kstatus-based readiness check. See [Status Detection](../status). |

## Status Hint Values

The `component-operator.cs.sap.com/status-hint` annotation accepts a comma-separated list of the following values:

| Hint | Description |
|------|-------------|
| `has-observed-generation` | Treat the object as having a `status.observedGeneration` field even if not yet set (handles lazy controllers). |
| `has-ready-condition` | Require a `Ready` condition; if absent, treat as `Unknown` (not ready). |
| `conditions=<type1>;<type2>` | Semicolon-separated list of additional condition types that must all be `True` for the object to be considered ready. |

## Annotation Value Normalisation

Annotation values are normalised before being evaluated, so `PascalCase`, `camelCase`, and `kebab-case` representations are all accepted. For example, `IfUnowned`, `ifUnowned`, and `if-unowned` are equivalent for `adoption-policy`.
