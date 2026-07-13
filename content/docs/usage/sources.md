---
title: "Sources"
linkTitle: "Sources"
weight: 1
description: >
  Configuring the source of manifests for a Component
---

The mandatory field `spec.sourceRef` defines where component-operator retrieves the manifests used to generate the component's dependent objects. Exactly one source type must be specified.

## Source Types

### Flux GitRepository

References a [Flux `GitRepository`](https://fluxcd.io/flux/components/source/gitrepositories/) object. The source controller fetches and serves the Git repository content as a tarball artifact.

```yaml
spec:
  sourceRef:
    fluxGitRepository:
      name: my-gitrepo
      # namespace: source-ns  # optional; defaults to component's namespace
```

### Flux OCIRepository

References a [Flux `OCIRepository`](https://fluxcd.io/flux/components/source/ocirepositories/) object, backed by an OCI-compatible container registry.

```yaml
spec:
  sourceRef:
    fluxOciRepository:
      name: my-ocirepo
      # namespace: source-ns  # optional; defaults to component's namespace
```

### Flux Bucket

References a [Flux `Bucket`](https://fluxcd.io/flux/components/source/buckets/) object, backed by an S3-compatible object storage bucket.

```yaml
spec:
  sourceRef:
    fluxBucket:
      name: my-bucket
      # namespace: source-ns  # optional; defaults to component's namespace
```

### Flux HelmChart

References a [Flux `HelmChart`](https://fluxcd.io/flux/components/source/helmcharts/) object. This is particularly useful when working with Helm repositories. Note that `spec.path` is still used as the chart path within the artifact.

```yaml
spec:
  sourceRef:
    fluxHelmChart:
      name: my-helmchart
      # namespace: source-ns  # optional; defaults to component's namespace
```

### Blueprint

A `Blueprint` is an in-cluster source type specific to component-operator. Unlike the Flux-based sources, a Blueprint stores the manifest templates directly inside a Kubernetes custom resource (in `spec.files`). This eliminates the need for an external repository and is well-suited for operator-managed or dynamically generated templates.

```yaml
spec:
  sourceRef:
    blueprint:
      name: my-blueprint
      # namespace: source-ns  # defaults to component's namespace
```

See the [Blueprint API reference](../../reference/blueprint) for details on how to define a Blueprint.

For all source types, cross-namespace references are allowed. If the namespace is omitted, the source is assumed to reside in the same namespace as the component.

## Decryption

Parts of the source manifests may be encrypted. Currently, only [SOPS](https://github.com/getsops/sops) is supported as the encryption provider. To enable decryption, set `spec.decryption.provider` to `sops` and provide a secret reference following the [same convention as Flux](https://fluxcd.io/flux/guides/mozilla-sops/). Only GPG and [age](https://github.com/filosottile/age) are supported as encryption backends.

```yaml
spec:
  sourceRef:
    fluxGitRepository:
      name: my-gitrepo
  decryption:
    provider: sops
    secretRef:
      name: sops-key
```

## Pinning to a Specific Revision or Digest

A component can be pinned to a specific revision or digest of the source artifact by setting `spec.revision` or `spec.digest`, or both.

When pinned, the component stays in `Pending` state until the source object's revision or digest matches the specified value. For Flux sources, revision and digest correspond to `status.artifact.revision` and `status.artifact.digest` of the referenced Flux object.

```yaml
spec:
  sourceRef:
    fluxGitRepository:
      name: my-gitrepo
  revision: main@sha1:4b14dbc37ca976be75a7508bb41fb99d4a36ab9
  # or:
  # digest: sha256:abc123...
```

Note that a revision or digest mismatch never blocks the deletion of a component.

### Coordinating Nested Components

Pinning is especially useful in scenarios where multiple nested components reference the same source. Without pinning, each component may reconcile against a different (possibly newer) version of the source as it is updated, leading to unpredictable intermediate states. By pinning nested components to the current revision or digest of the owning component, you ensure they all reconcile against exactly the same source content, providing a consistent and predictable rollout order. This is often achieved like this

```yaml
spec:
  sourceRef:
    fluxGitRepository:
      name: my-gitrepo
  revision: {% component.Status.LastAttemptedRevision %}
  digest: {% component.Status.LastAttemptedDigest %}
```

Here, `component` is a template function (available with the extended template syntax) returning the current object state of the owning component. As a consequence, the sub-component is forced to reconcile the digest or revision of the owning component, avoiding that the owning component goes through, observing an outdated previous ready state of the subcomponent.