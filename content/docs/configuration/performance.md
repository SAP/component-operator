---
title: "Performance and Sizing"
linkTitle: "Performance and Sizing"
weight: 2
description: >
  Guidance for tuning concurrency and resource allocation
---

## Concurrency

The `-max-concurrent-reconciles` flag controls how many `Component` objects can be reconciled simultaneously. Each reconciliation worker handles one component at a time: fetching the source artifact, rendering the manifests, applying or deleting dependent objects, and updating the component's status.

The default of `5` concurrent workers is suitable for small to medium clusters with up to a few hundred components. For larger deployments, consider increasing this value.

Factors that influence the right setting:

- **Number of components**: More components benefit from higher concurrency, up to the point where API server throughput becomes the bottleneck.
- **Component size**: Components with many dependent objects (hundreds of resources) are slower to reconcile. A lower concurrency limit may be preferable to avoid overloading the API server.
- **Requeue interval**: A shorter `requeueInterval` means more frequent reconcile loops, increasing the steady-state load. With a short interval and many components, higher concurrency can help keep up.
- **API server capacity**: Each reconciliation issues multiple API server requests (reads and writes). Monitor API server request rates and latency when tuning concurrency.

As a rough guideline:

| Cluster size | Recommended `-max-concurrent-reconciles` |
|---|---|
| < 100 components | 5 (default) |
| 100–500 components | 10–20 |
| 500–2000 components | 20–50 |
| > 2000 components | 50+ (tune based on API server load) |

## Kubernetes API Server Load

Our experience shows that you can roughly calculate with 0.1 Kubernetes API server requests per component and second.

Typically peaks are occurring regularly (every couple of minutes) where this rate increases by a factor of 5-10 for a short period.

But of course, the number of API server calls depends heavily on
- The complexity of your components; for example, declaring dependencies leads to an increased activity, because reconciliations of components trigger the reconciliation of other components. Also components doing many lookups into the cluster require more API calls.
- How the intervals (`spec.requeueInterval` and `spec.retryInterval`) are set.
- How many real changes are applied.
- How many components are in error state; erroneous components produce more API server load.

## Memory and CPU

Component-operator is a typical Kubernetes controller and has modest resource requirements. Memory usage is primarily driven by:

- The number of components and their inventory (tracked in memory via the controller-runtime informer cache).
- The size of rendered manifests held temporarily during reconciliation.

A starting point for resource requests/limits in a production deployment:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

Increase memory limits if you observe OOMKills, particularly in clusters with large numbers of components or very large manifest sets. CPU limits are generally less critical — the controller is bursty during reconciliation but largely idle at steady state.
