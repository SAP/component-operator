---
title: "Dependents Lifecycle"
linkTitle: "Dependents Lifecycle"
weight: 7
description: >
  Controlling how dependent objects are applied and deleted
---

## Reconciliation in Waves

Component-operator supports ordered reconciliation of dependent objects using the concept of **waves**. Objects are grouped into waves and processed wave by wave, in ascending order of wave number. Wave numbers can be negative, zero (the default), or positive integers in the range -32768 to 32767.

### Apply Waves

During apply (creation or update), objects are reconciled wave by wave. The next wave only begins once **all objects of the previous wave are fully ready** — not just created or updated, but actually ready according to their [status](../status). Simply initiating creation or an update is not enough.

This ensures that prerequisites (e.g., CRDs, operators, or databases) are fully operational before depending resources are applied.

An object's apply wave is set via the annotation `component-operator.cs.sap.com/apply-order` in the rendered manifest:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/apply-order: "-10"   # early wave
```

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/apply-order: "10"    # late wave
```

Objects with no annotation are treated as wave `0`.

### Delete Waves

During deletion, objects are also deleted wave by wave, in ascending order. The next delete wave only begins once **all objects of the previous delete wave are fully gone** from the cluster. Simply having issued the delete call is not enough — the objects must have disappeared (that is, a 404 NotFound error is returned by the API when getting the object).

The delete wave is independent of the apply wave and is configured separately:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/delete-order: "5"
```

Objects with no annotation are treated as delete wave `0`.

### Purge Orders

A related concept is the **purge order**, set via `component-operator.cs.sap.com/purge-order`. An object having a purge order defined is deleted from the cluster at the end of the specified apply wave, and its inventory record is set to `Completed`. This is useful for spawning ephemeral objects during reconciliation (similar to Helm hooks) that are part of the component, but must not or are not required to physically exist in the cluster all the time.

### Implicit Ordering Within Waves

When applying a wave, a certain default ordering logic is used under the hood, to circumvent obvious issues that would occur otherwise. For example, this means:
- Namespaces are always created before namespaced objects using it.
- RBAC objects are always reconciled early.
- If a wave contains custom resource definitions **and** corresponding instances, then the reconciliation of the instances is postponed until all other objects in the component are ready.
- And some more ...

Note that a similar implicit logic exists for the deletion process.

## Update Policy

Multiple modes are supported for component-operator applying (creating or updating) objects in the Kubernetes API.

| Value | Description |
|-------|-------------|
| `Replace` | Full PUT request (equivalent to `kubectl replace`). |
| `Recreate` | Delete and re-create the object on update. |
| `SsaMerge` | Server-side apply without reclaiming fields from other field managers. |
| `SsaOverride` (default) | Server-side apply, plus reclaim fields owned by `kubectl` or `helm` field managers. |

The default behavior is `SsaOverride` which is equivalent to what the FluxCD kustomize-controller is doing:
- reclaim fields which have a field owner matching the pattern 'kubectl*'
- then use Server-Side-Apply to submit the object to the Kubernetes API. 

The update mode can be defaulted on component level.

```yaml
spec:
  updatePolicy: SsaOverride
```

And it can be overridden per object.

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/update-policy: ssa-merge
```

## Delete Policy

In general, dependent objects may be deleted in two cases:
- if the component is updated with a new manifest set no longer containing a previously included object
- if the component is deleted as a whole.

In both cases, the default behavior is to delete the dependents from the Kubernetes API through a DELETE request.
Sometimes it is desired to orphan an object. Orphaning means to stop the tracking as a dependent object without physically deleting it from the cluster.

Orphaning can be controlled through the delete policy.

| Value | Description |
|-------|-------------|
| `Delete` (default) | Objects are deleted when the component is deleted. |
| `Orphan` | Objects are left in the cluster (orphaned) in both cases, i.e. when the object becomes redundant during a component update, and also when the component is deleted. |
| `OrphanOnApply` | Objects are deleted when the whole component is deleted, but are orphaned when becoming obsolete because of a component manifest change. |
| `OrphanOnDelete` | Objects are deleted when becoming obsolete because of a component manifest change, but are kept when the component as a whole is deleted.  |

Component-level default:

```yaml
spec:
  deletePolicy: Delete
```

Per-object override:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/delete-policy: orphan
```