---
title: "Apply, Delete and Purge Orders"
linkTitle: "Apply, Delete and Purge Orders"
weight: 2
description: >
  Control dependent-object lifecycle ordering using apply-order, delete-order, and purge-order annotations
---

This tutorial shows how component-operator's ordering annotations let you sequence the creation, deletion, and one-shot execution of dependent objects within a single component. You will use a `Blueprint` as the manifest source so no external Git repository or Helm repository is needed.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, the [Cluster Setup](../../getting-started/setup) guide walks you through creating a local kind cluster with everything in place.

## What you will build

The component manages these resources:

| Resource | `apply-order` | `delete-order` | `purge-order` | Role |
|---|---|---|---|---|
| `ConfigMap` `nginx` | `(0)`| `1` | — | Nginx environment variables |  
| `PersistentVolumeClaim` `webdata` | `(0)` | `(0)` | — | Shared storage between the init job<br> and the web server |
| `Job` `init-content` | `(0)` | `(0)` | `0` | Writes an HTML file into the PVC;<br>removed from the cluster once it succeeds |
| `Deployment` `nginx` | `1` | `(0)` | — | Serves the HTML file;<br>deleted before the ConfigMap on teardown |

The expected progression:

1. **Apply wave 0** — ConfigMap and PVC are created; job runs, writes `Hello, world!` to the PVC, then exits. At the end of wave 0 the Job is **purged**: deleted from the cluster while its inventory record in the component's status is retained with state `Completed`.
2. **Apply wave 1** — nginx Deployment starts. Its pods find the PVC already populated.

On deletion the order reverses: nginx and the volume claim (delete-order `0`) are torn first. The PVC stays around for about 30 seconds because the nginx pods binding it have an artificial `preStop` delay. Only after the PVC is gone, the ConfigMap is deleted. Note: using delete waves here is not functionally necessary, but it demonstrates the idea.

## 1. Create the Blueprint

Create a `Blueprint` that holds all manifest templates under a single file key:

```yaml
# wave-demo-blueprint.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: wave-demo
  namespace: default
spec:
  files:
    resources.yaml: |
      ---
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: nginx
        annotations:
          # component-operator.cs.sap.com/apply-order: "0"
          component-operator.cs.sap.com/delete-order: "1"
      data:
        NGINX_HOST: foobar.io 
      ---
      apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: webdata
        annotations:
          # component-operator.cs.sap.com/apply-order: "0"
          # component-operator.cs.sap.com/delete-order: "0"
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 100Mi
      ---
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: init-content
        annotations:
          # component-operator.cs.sap.com/apply-order: "0"
          # component-operator.cs.sap.com/delete-order: "0"
          component-operator.cs.sap.com/purge-order: "0"
      spec:
        template:
          spec:
            restartPolicy: Never
            containers:
              - name: data
                image: alpine
                command:
                  - /bin/sh
                  - -c
                  - echo "Hello, world!" > /data/index.html
                lifecycle:
                  postStart:
                    sleep:
                      seconds: 30
                volumeMounts:
                  - name: webdata
                    mountPath: /data
            volumes:
              - name: webdata
                persistentVolumeClaim:
                  claimName: webdata
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: nginx
        annotations:
          component-operator.cs.sap.com/apply-order: "1"
          # component-operator.cs.sap.com/delete-order: "0"
      spec:
        replicas: 2
        selector:
          matchLabels:
            app: nginx
        template:
          metadata:
            labels:
              app: nginx
          spec:
            containers:
              - name: nginx
                image: nginx:alpine
                envFrom:
                  - configMapRef:
                      name: nginx
                ports:
                  - containerPort: 80
                lifecycle:
                  preStop:
                    sleep:
                      seconds: 30
                volumeMounts:
                  - name: webdata
                    mountPath: /usr/share/nginx/html
                    readOnly: true
            volumes:
              - name: webdata
                persistentVolumeClaim:
                  claimName: webdata
```

```bash
kubectl apply -f wave-demo-blueprint.yaml
```

A few things to note about the Job definition:

- The `postStart` lifecycle hook of the job, and the `preStop` hook of the nginx deployment are added to make the waves observable.
- The PVC uses `ReadWriteOnce`, which on a single-node kind cluster allows both nginx replicas to mount the volume (all pods land on the same node). On multi-node clusters you would need a `ReadWriteMany` storage class or a single replica.

## 2. Create the Component

```yaml
# wave-demo-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: wave-demo
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: wave-demo
```

```bash
kubectl apply -f wave-demo-component.yaml
```

## 3. Observe apply wave ordering

Open terminals to watch the progression in parallel.

**Terminal 1** — component state:

```bash
kubectl get component wave-demo -w
```

**Terminal 2** — configmap state:

```bash
kubectl get cm -w
```

**Terminal 3** — pvc state:

```bash
kubectl get pvc -w
```

**Terminal 4** — job state:

```bash
kubectl get job -w
```

**Terminal 5** — deployment state:

```bash
kubectl get deployment -w
```

You should observe the following sequence:

1. The `nginx` ConfigMap, and the `webdata` PVC are created and the `init-content` Job appears. Its pod runs for ~30 seconds. During this time the component stays in `Processing` state and the `nginx` Deployment **does not yet exist**.
2. The Job pod exits successfully. The `init-content` Job object **disappears** from the cluster — it has been purged at the end of apply wave 0 which is now done.
3. Now, apply wave 1 starts, the `nginx` Deployment appears and its pods start up.
4. Once the deployment is ready, apply wave 1 is finished, and component transitions to `Ready`.

## 4. Inspect the inventory

The purged Job is gone from the cluster but is still tracked by component-operator. Query the component's inventory:

```bash
kubectl get component wave-demo \
  -o jsonpath='{range .status.inventory[*]}{.kind}{"\t"}{.phase}{"\n"}{end}'
```

The output will look similar to:

```
PersistentVolumeClaim   Ready
Job                     Completed
Deployment              Ready
ConfigMap               Ready
```

The `Completed` state for the Job confirms it fulfilled its role and was intentionally removed, not accidentally deleted or failed.

Verify that nginx is actually serving the content written by the Job:

```bash
kubectl port-forward deployment/nginx 8080:80 &
sleep 1; curl http://localhost:8080
# Hello, world!
sleep 1; kill %1
```

## 5. Delete the Component and observe delete wave ordering

Delete the component:

```bash
kubectl delete component wave-demo
```

The nginx Deployment, and the (completed) Job, and the PVC (delete-order `0`) are deleted first. Only after they all have fully disappeared component-operator proceeds to delete the ConfigMap (delete-order `1`).
Note that the Job is already absent (purged during the apply phase), such that its deletion is a no-op.

## 6. Cleanup

Component deletion removes all dependent objects. The `Blueprint` is not managed by the component and must be removed separately:

```bash
kubectl delete blueprint wave-demo
```
