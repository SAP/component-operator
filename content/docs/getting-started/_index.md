---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Get component-operator running and deploy your first application in minutes
---

This quickstart walks you through setting up component-operator on a local [kind](https://kind.sigs.k8s.io/) cluster and deploying the [podinfo](https://github.com/stefanprodan/podinfo) sample application — first from a Helm chart, then from a Kustomize overlay.

1. [Cluster Setup](setup) — create a kind cluster, install Flux source-controller, and install component-operator
2. [Scenario 1: Helm](scenario-helm) — deploy podinfo via its OCI Helm chart
3. [Scenario 2: Kustomize](scenario-kustomize) — deploy podinfo via a Kustomize overlay from Git
4. [Next Steps](next-steps) — where to go from here
