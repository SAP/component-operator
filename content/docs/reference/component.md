---
title: "Component"
linkTitle: "Component"
weight: 1
description: >
  API reference for the Component custom resource
---

## Overview

The `Component` custom resource is the primary API for managing Kubernetes components with component-operator. It defines a set of Kubernetes resources (deployments, services, RBAC, etc.) that are managed as a cohesive unit.

## API Version

`core.cs.sap.com/v1alpha1`

## Kind

`Component`

## Spec

The `ComponentSpec` defines the desired state of a Component.

### Source Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sourceRef` | [SourceReference](#sourcereference) | Yes | Reference to the source containing the manifest templates |
| `revision` | string | No | Pin component to a specific source revision |
| `digest` | string | No | Pin component to a specific source digest |
| `path` | string | No | Subfolder within the source (default: root) |

### Placement and Targeting

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Target namespace for deployment (default: component's namespace) |
| `name` | string | No | Target name for deployment (default: component's name) |
| `serviceAccountName` | string | No | Service account to impersonate during reconciliation |
| `kubeConfig` | [KubeConfigSpec](#kubeconfigspec) | No | KubeConfig for deploying to remote cluster |

### Templating and Values

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `values` | JSON | No | Inline values for templating |
| `valuesFrom` | [][SecretKeyReference](#secretkeyreference) | No | Secret references containing values |
| `decryption` | [Decryption](#decryption) | No | Decryption settings for encrypted manifests |
| `postBuild` | [PostBuild](#postbuild) | No | Post-build variable substitution and patches |

### Reconciliation Control

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `suspend` | boolean | No | Suspend reconciliation (default: false) |
| `requeueInterval` | duration | No | Period for re-reconciliation after success (default: 10m) |
| `retryInterval` | duration | No | Period for re-reconciliation after retriable error |
| `timeout` | duration | No | How long dependent objects have to become ready |
| `sticky` | boolean | No | Stick to source revision until ready or timeout |
| `reapplyInterval` | duration | No | Force reapply interval (default: 60m) |

### Policies

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `adoptionPolicy` | string | No | How to handle existing objects (`IfUnowned`, `Never`, `Always`; default: `IfUnowned`) |
| `updatePolicy` | string | No | How to update objects (`Replace`, `Recreate`, `SsaMerge`, `SsaOverride`; default: `SsaOverride`) |
| `deletePolicy` | string | No | What happens on deletion (`Delete`, `Orphan`; default: `Delete`) |
| `missingNamespacesPolicy` | string | No | Auto-create missing namespaces (`Create`, `DoNotCreate`; default: `Create`) |

### Advanced Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `dependencies` | [][Dependency](#dependency) | No | Dependencies to other components |
| `additionalManagedTypes` | [][TypeInfo](#typeinfo) | No | Additional CRD types managed by this component |

## SourceReference

Defines the source of manifest templates. Exactly one source type must be provided.

| Field | Type | Description |
|-------|------|-------------|
| `blueprint` | [BlueprintReference](#blueprintreference) | Reference to in-cluster Blueprint |
| `httpRepository` | [HttpRepository](#httprepository) | HTTP-accessible repository |
| `fluxGitRepository` | [FluxGitRepositoryReference](#fluxgitrepositoryreference) | Flux GitRepository reference |
| `fluxOciRepository` | [FluxOciRepositoryReference](#fluxocirepositoryreference) | Flux OCIRepository reference |
| `fluxBucket` | [FluxBucketReference](#fluxbucketreference) | Flux Bucket reference |
| `fluxHelmChart` | [FluxHelmChartReference](#fluxhelmchartreference) | Flux HelmChart reference |

### BlueprintReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace of the Blueprint (default: component's namespace) |
| `name` | string | Yes | Name of the Blueprint |

### HttpRepository

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | URL of the source artifact |
| `digestHeader` | string | No | Header containing digest (default: ETag) |
| `revisionHeader` | string | No | Header containing revision (default: digestHeader) |

### FluxGitRepositoryReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace (default: component's namespace) |
| `name` | string | Yes | Name |

### FluxOciRepositoryReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace (default: component's namespace) |
| `name` | string | Yes | Name |

### FluxBucketReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace (default: component's namespace) |
| `name` | string | Yes | Name |

### FluxHelmChartReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace (default: component's namespace) |
| `name` | string | Yes | Name |

## KubeConfigSpec

Reference to a kubeconfig stored in a Secret.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secretRef` | [SecretKeyReference](#secretkeyreference) | Yes | Secret containing kubeconfig |

## Decryption

Settings for decrypting encrypted manifests using SOPS.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | No | Decryption provider (default: `sops`) |
| `secretRef` | [SecretReference](#secretreference) | Yes | Secret containing provider configuration |

## PostBuild

Post-build variable substitution and transformations.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `substitute` | map[string]string | No | Inline variable substitutions |
| `substituteFrom` | [][SubstituteFrom](#substitutefrom) | No | Secrets containing substitution variables |
| `patches` | [][KustomizePatch](#kustomizepatch) | No | Kustomize patches to apply |
| `images` | [][KustomizeImage](#kustomizeimage) | No | Kustomize image replacements |

### SubstituteFrom

Reference to a Secret providing variables for substitution.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Secret name |
| `items` | [][SecretItem](#secretitem) | No | Restrict and rename the secret keys to be used (if omitted, all keys are used as-is) |

### SecretItem

Selects a single key of a Secret and exposes it under a different variable name.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | Key of the secret item |
| `as` | string | Yes | Variable name the key is renamed to |

### KustomizePatch

Kustomize patch specification; a subset of [kustomize's Patch type](https://pkg.go.dev/sigs.k8s.io/kustomize/api/types#Patch).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `patch` | string | No | Strategic-merge or JSON patch, as inline YAML or JSON |
| `target` | [KustomizeSelector](#kustomizeselector) | No | Selects the objects the patch is applied to |

### KustomizeSelector

Kustomize object selector; corresponds to [kustomize's Selector type](https://pkg.go.dev/sigs.k8s.io/kustomize/api/types#Selector).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `group` | string | No | API group of the targeted objects |
| `version` | string | No | API version of the targeted objects |
| `kind` | string | No | Kind of the targeted objects |
| `name` | string | No | Name of the targeted objects (regular expression) |
| `namespace` | string | No | Namespace of the targeted objects (regular expression) |
| `annotationSelector` | string | No | Annotation selector the targeted objects must match |
| `labelSelector` | string | No | Label selector the targeted objects must match |

### KustomizeImage

Kustomize image modifier specification; a subset of [kustomize's Image type](https://pkg.go.dev/sigs.k8s.io/kustomize/api/types#Image).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Image name to be replaced |
| `newName` | string | No | New image name |
| `newTag` | string | No | New image tag |
| `digest` | string | No | New image digest |

## Dependency

Reference to another Component that must be ready before this component.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Namespace of dependency (default: component's namespace) |
| `name` | string | Yes | Name of dependency component |

## TypeInfo

Represents a Kubernetes type for additional managed resources.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `group` | string | Yes | API group (use `*` for wildcard or `*.<suffix>` for pattern matching) |
| `kind` | string | Yes | API kind (use `*` for wildcard) |

## SecretReference

Reference to a Secret.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Secret name |

## SecretKeyReference

Reference to a Secret and optional key.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Secret name |
| `key` | string | No | Secret key (might be defaulted, depending on context) |

## Status

The `ComponentStatus` defines the observed state of a Component.

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Last observed generation |
| `appliedGeneration` | int64 | Generation that was successfully applied |
| `lastObservedAt` | metav1.Time | Timestamp of last observation |
| `lastAppliedAt` | metav1.Time | Timestamp of last successful application |
| `processingDigest` | string | Digest currently being processed |
| `processingSince` | metav1.Time | Timestamp when processing started |
| `lastProcessingDigest` | string | Digest of last processing attempt |
| `conditions` | [][Condition](#condition) | Standard Kubernetes conditions |
| `state` | [State](#state) | Component state |
| `inventory` | []InventoryItem | List of managed resources |
| `sourceRef` | [SourceReferenceStatus](#sourcereferencestatus) | Current source reference state |
| `lastAttemptedDigest` | string | Digest of last reconciliation attempt |
| `lastAttemptedRevision` | string | Revision of last reconciliation attempt |
| `lastAppliedDigest` | string | Digest of last successful application |
| `lastAppliedRevision` | string | Revision of last successful application |

### Condition

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Condition type (e.g., `Ready`) |
| `status` | string | Condition status (`True`, `False`, `Unknown`) |
| `lastTransitionTime` | metav1.Time | Last time the condition transitioned |
| `reason` | string | Machine-readable reason for the condition |
| `message` | string | Human-readable message |

### State

Component state can be one of:
- `Ready` - Component is ready
- `Pending` - Component is pending (e.g., suspended)
- `Processing` - Component is being processed
- `DeletionPending` - Component deletion is pending
- `Deleting` - Component is being deleted
- `Error` - Component encountered an error

### SourceReferenceStatus

| Field | Type | Description |
|-------|------|-------------|
| `artifact` | [Artifact](#artifact) | Source artifact information |
| `digest` | string | Computed digest of source reference |

### Artifact

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | URL of the artifact |
| `digest` | string | Digest of the artifact |
| `revision` | string | Revision of the artifact |
