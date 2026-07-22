---
title: "Component Deletion"
linkTitle: "Component Deletion"
weight: 9
description: >
  What happens when a Component is deleted
---

When a `Component` object is deleted, component-operator reconciles the deletion by removing all dependent objects it manages. The process is more nuanced than a simple bulk delete, particularly around extension types.

## Basic Deletion Flow

Upon receiving a delete request, the component enters a `Deleting` state. Dependent objects are then deleted according to their [delete order](../dependents), wave by wave. Each wave only starts once all objects of the previous wave have fully disappeared from the cluster.

The [delete policy](../dependents#delete-policy) controls whether objects are actually deleted or orphaned when the component is removed.

## Extension Types and CRDs

Component-operator has special handling for **extension types** — types that extend the Kubernetes API, such as Custom Resource Definitions (CRDs) or types provided through APIService registrations.

If the component's manifests include extension types (e.g., a CRD) as well as instances of those types (e.g., custom resources), the following ordering logic applies:

- **During apply**: Instances of extension types are applied as late as possible, to ensure the corresponding controllers and webhooks are up and running before the instances are submitted.
- **During delete**: Instances of extension types are deleted as early as possible, before the extension type definition itself and related objects are removed, so that controllers and webhooks can still process the deletion.

### Blocking Deletion by Foreign Instances

A particularly important safety mechanism: **foreign instances** of managed extension types block the deletion of the entire component. That is, if a component manages an extension type and there exist instances of that type in the cluster that are not part of this component, the component will not proceed with deletion until those foreign instances are gone. This ensures that controllers and webhooks responsible for the extension type - of course only if being part of the component as well - stay alive and remain able to reconcile the deletion of the foreign instances. This pattern efficiently addresses the well-known 'stuck finalizer' problem. 

## Additional Managed Types

Some components implicitly introduce extension types into the cluster — not by directly including them in their manifests, but through controllers that install CRDs or register API services as a side effect. A classic example are [Crossplane providers](https://docs.crossplane.io/latest/packages/providers/), which typically register CRDs for their managed resource types on startup.

In such cases, the component does not list those CRDs in its manifests, so component-operator would not normally know about them. The field `spec.additionalManagedTypes` addresses this by explicitly declaring these implicitly managed types:

```yaml
spec:
  additionalManagedTypes:
    - group: example.crossplane.io
      kind: MyManagedResource
    - group: "*.crossplane.io"   # wildcard: match all groups with suffix .crossplane.io
      kind: "*"                  # wildcard: match all kinds
```

With `additionalManagedTypes` declared, component-operator applies the same foreign-instance blocking logic to these types: the component will not be deleted while foreign instances of the declared types exist. Wildcards can be used as follows:
- The kind can be provided as  `*`, which matches any value.
- The group can be `*` (matching any value) or have the form `*.suffix`; in this case, the asterisk matches one or multiple DNS labels.


