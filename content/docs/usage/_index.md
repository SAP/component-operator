---
title: "Usage"
linkTitle: "Usage"
weight: 4
description: >
  About writing manifests and controlling reconciliation of dependent resources
---

Explore the templating language, and how to control the lifecycle of dependent objects.
Learn about tuning the reconciliation on operator level, component level, and on the level of individual resources.

## Glossary

### Component

A component is a coherent set of Kubernetes resources. The `Component` resource (reconciled by Component Operator) ties together a **source** containing manifests with instructions on how to apply the described Kubernetes objects to the cluster. It is the central abstraction in component-operator: you declare what to deploy (via the source), how to parameterize it (via values), and how to manage the lifecycle of the resulting objects.

### Dependent Object

A **dependent object** is a specific Kubernetes resource (e.g., a `Deployment`, `Service`, `ConfigMap`) that results from rendering the component's manifests. Dependent objects are owned and managed by the component throughout their lifecycle: created when the component is applied, updated or deleted when the manifests change, and deleted when the component is removed.

### Source

A **source** defines how component-operator retrieves the manifests. A source can be:

- A **Flux GitRepository**, **OCIRepository**, **Bucket**, or **HelmChart** — externally managed artifact sources provided by the [Flux source controller](https://fluxcd.io/flux/components/source/).
- A **Blueprint** — an in-cluster source type where manifests are stored directly inside a Kubernetes custom resource (`Blueprint`), without requiring an external repository.

See [Sources](./sources) for details.

### Manifest

A **manifest** is a (potentially templated) YAML document specifying a Kubernetes resource. Manifests are the raw input from which dependent objects are derived. They may contain Go template expressions, Helm-style function calls, or kustomize overlays that are evaluated during the rendering step.

### Generation / Rendering

**Generation** (or **rendering**) is the process of producing the final set of Kubernetes manifests from the source files. During this step, template expressions are evaluated, values are substituted, kustomize overlays are applied, and the result is a concrete list of Kubernetes objects ready to be applied to the cluster.