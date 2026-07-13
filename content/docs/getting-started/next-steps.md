---
title: "Next Steps"
linkTitle: "Next Steps"
weight: 4
description: >
  Where to go after the quickstart
---

You've deployed podinfo with component-operator — here are the recommended next steps.

## Learn more about usage

The [Usage](../../usage) section covers all the concepts you'll need for real-world deployments:

- [Sources](../../usage/sources) — GitRepository, OCIRepository, Bucket, HelmChart, and Blueprint source types
- [Manifests](../../usage/manifests) — Helm and Kustomize rendering, values, template functions
- [Drift Detection](../../usage/drift-detection) — how component-operator detects and corrects configuration drift
- [Dependent Objects](../../usage/dependents) — apply/delete ordering, update and delete policies

## Explore the API reference

- [Component](../../reference/component) — full field reference for the `Component` custom resource
- [Blueprint](../../reference/blueprint) — in-cluster manifest template source

## Configure the operator

- [Command-line Flags](../../configuration/flags) — runtime flags including concurrency and service account defaults
- [Performance](../../configuration/performance) — sizing guidance for larger clusters
