---
title: "Container Images"
linkTitle: "Container Images"
weight: 2
description: >
  Where to find the component-operator container images
---

OCI images for component-operator are published to the GitHub Container Registry alongside each release. You can browse all available images and versions on the [GitHub Packages page](https://github.com/orgs/SAP/packages?repo_name=component-operator).

The image reference is:

```
ghcr.io/sap/component-operator:<version>
```

Custom Resource Definitions can be downloaded separately as:

```
ghcr.io/sap/component-operator/crds:<version>
```

Replace `<version>` with the desired release tag (e.g. `v0.12.0`). Using `latest` is not recommended for production deployments.
