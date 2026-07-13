---
title: "Ownership and Adoption"
linkTitle: "Ownership and Adoption"
weight: 11
description: >
  How component-operator tracks ownership and handles pre-existing objects
---

Component-operator tracks ownership of dependent objects using a dedicated label: `component-operator.cs.sap.com/owner-id`. This label is set on every object managed by a component and contains a globally unique identifier for the owning component (incorporating a unique ID of the component's Kubernetes cluster). It serves as the basis for all ownership and adoption decisions.

## Adoption Policy

It can happen that a dependent object already exists in the cluster when a component is applied — either because it was created manually, by another tool, or by a different component. The `spec.adoptionPolicy` field on the component
(and the per-object annotation `component-operator.cs.sap.com/adoption-policy`) controls how component-operator handles this situation.

The following values are supported on component level:

| Value | Description |
|-------|-------------|
| `IfUnowned` (default) | Adopt the object if it has no `owner-id` label. Fail if the label exists but points to a different owner. |
| `Never` | Always fail if the object already exists (regardless of ownership). |
| `Always` | Adopt the object unconditionally, even if it is owned by a different component. |

The component-level `spec.adoptionPolicy` sets the default for all dependent objects:

```yaml
spec:
  adoptionPolicy: IfUnowned  # IfUnowned (default) | Never | Always
```

Individual dependent objects can override this default by adding the annotation directly to the manifest:

```yaml
metadata:
  annotations:
    component-operator.cs.sap.com/adoption-policy: always # or never or if-unowned
```

Note that, before being evaluated, the values of this annotation are normalized to Kebab case.
Such that e.g. `if-unowned` and `IfUnowned` or `ifUnowned` are equivalent.

## Migration of Objects between Components

Sometimes it is necessary to move an object from one component (A) to another one (B). Then it matters which component is processed first.

**Case: A is reconciled before B** 

This will work, but by default, the object is deleted by A and recreated by B. In order to avoid the recreation, the object must be shipped with a deletion policy `orphan` or `orphan-on-apply` in a separate preparation rollout.
Afterwards, setting an adoption policy of `always` allows B to claim it. Finally, when the owner change has rolled out everywhere, the temporarily added policies can be removed.

**Case: B is reconciled before A**

This only works if the moved object gets an adoption policy `always`. This allows B to claim it. No explicit orphaning is necessary in this case, because A will recognize the owner change, and just disown the object. Similar to the first case, the temporarily added policy can be removed when the move is fully propagated. 
