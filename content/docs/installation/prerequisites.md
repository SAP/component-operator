---
title: "Prerequisites"
linkTitle: "Prerequisites"
weight: 1
description: >
  What needs to be in place before installing Component Operator
---

Component-operator requires the [Flux source controller](https://fluxcd.io/flux/components/source/) to be installed in the cluster. It relies on Flux's `GitRepository`, `OCIRepository`, `Bucket`, and `HelmChart` source types to fetch manifest artifacts. You can install just the source controller without the full Flux stack:

```bash
flux install --components source-controller
```

Alternatively, install the complete Flux toolkit if you plan to use other Flux features alongside component-operator:

```bash
flux install
```
