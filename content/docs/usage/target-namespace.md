---
title: "Target Namespace and Name"
linkTitle: "Target Namespace and Name"
weight: 6
description: >
  Controlling where dependent objects are deployed
---

The optional fields `spec.namespace` and `spec.name` customize the deployment namespace and name of the component, as defined by component-operator-runtime.

## Defaulting

If `spec.namespace` is not set, the target namespace defaults to `metadata.namespace` — that is, the same namespace in which the `Component` object itself resides.

If `spec.name` is not set, the target name defaults to `metadata.name` — the name of the `Component` object.

### Effect on Dependent Objects

The target namespace is passed to the manifest generator. Any namespaced dependent objects whose manifest does not specify an explicit namespace will be placed in this target namespace. This allows a single set of manifests to be deployed into different namespaces simply by varying `spec.namespace` across Component instances.

The target namespace and name are accessible in
- HelmGenerator as `.Release.Namespace` and `.Release.Name`
- KustomizeGenerator via the `namespace` and `name` template functions.

### Example

```yaml
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  namespace: applications
  name: my-app
spec:
  namespace: my-app   # deploy dependents into the 'my-app' namespace
  name: app           # use 'app' as the deployment name
  sourceRef:
    fluxGitRepository:
      name: app-manifests
```

In this example, `metadata.namespace` is `applications` (where the Component object lives), but all dependent objects without an explicit namespace in their manifests will land in `my-app`.

## Automatic Namespace Creation

By default, component-operator automatically creates namespaces for dependent objects whose namespace does not yet exist.

This behavior can be tweaked by setting `spec.missingNamespacesPolicy` on component level.

| Value | Description |
|-------|-------------|
| `Create` (default) | Missing namespaces are created automatically. |
| `DoNotCreate` | Do not create missing namespaces; fail instead. |


```yaml
spec:
  missingNamespacesPolicy: DoNotCreate
```

Most people probably want to let component-operator automatically create missing namespaces. However in some cases, this might not be desired. For example, if component-operator reconciles a component as a restricted service account lacking the RBAC permissions to create namespaces. 