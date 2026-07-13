---
title: "Blueprint"
linkTitle: "Blueprint"
weight: 2
description: >
  API reference for the Blueprint custom resource
---

## Overview

The `Blueprint` custom resource provides an in-cluster mechanism for storing manifest templates. Blueprints serve as reusable component templates that can be referenced by Component resources, eliminating the need for external source repositories.

## API Version

`core.cs.sap.com/v1alpha1`

## Kind

`Blueprint`

## Spec

The `BlueprintSpec` defines the desired state of a Blueprint.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `files` | map[string]string | No | Map of file paths to file contents |

### Files Field

The `files` field contains a map where:
- **Key**: The relative file path within the blueprint (e.g., `resources/deployment.yaml`, `kustomization.yaml`)
- **Value**: The complete file contents as a string

Files can contain:
- Plain YAML manifests
- Go-templated YAML (will be rendered when used by a Component)
- Kustomization files
- Helm charts (if a `Chart.yaml` is present)

## Related Resources

- [Component API Reference](./component)
- [Kustomize Generator Documentation](https://sap.github.io/component-operator-runtime/docs/generators/kustomize/)
- [Helm Generator Documentation](https://sap.github.io/component-operator-runtime/docs/generators/helm/)
