---
title: "Harden Deployments Using Unprivileged Service Accounts"
linkTitle: "Harden Deployments Using Unprivileged Service Accounts"
weight: 8
description: >
  Use --default-service-account to constrain what component-operator can deploy
---

## The default security posture

By default, component-operator performs every reconciliation using its own pod service account. That account typically holds broad cluster-wide permissions — often `cluster-admin` — so that it can create any resource in any namespace.

This creates a privilege-escalation risk: anyone with permission to create a `Component` in a namespace can exploit the operator's elevated rights to deploy arbitrary resources anywhere in the cluster, without being a cluster admin themselves.

A component can voluntarily restrict its own permissions by setting `spec.serviceAccountName`. The operator then impersonates that service account when reconciling the component's dependent objects. But this is an **opt-in** on the component author's side, not an enforceable constraint.

## Enforcing namespace-scoped deployment rights

The `--default-service-account` controller flag changes the default behaviour. When set, component-operator impersonates the named service account — resolved in the component's own namespace — for every component that does not explicitly set `spec.serviceAccountName` or `spec.kubeConfig`.

Now assume a user has enhanced privileges (maybe full access) in a certain namespace. But no or only read permissions outside this namespace. Then this user is able to create components in the namespace, but in all cases, a service account in this namespace is used by component-operator to reconcile the component. May it be set explicitly as `spec.serviceAccountName` or defaulted via the `--default-service-account` flag. Because Kubernetes RBAC prevents privilege escalation, the user cannot create a service account having more privileges than currently held.
Consequently it is impossible to break out of the namespace jail.

This tutorial shows the hardened configuration.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide to create a local kind cluster with everything in place.

## 1. Reconfigure component-operator with a default service account

Reinstall the Helm chart with the `options.defaultServiceAccount` value set:

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system \
  --set options.defaultServiceAccount=deployer
```

This passes `--default-service-account=deployer` to the controller manager. From now on, any component that does not set `spec.serviceAccountName` will be reconciled using a service account named `deployer` in the component's own namespace.

## 2. Create the deployer service account

The service account must exist in the namespace where components will run. Create it in `default`:

```bash
kubectl create serviceaccount deployer -n default
```

The account has no permissions yet.

## 3. Deploy podinfo — and observe the error

Repeat [Scenario 2: Kustomize](../../getting-started/scenario-kustomize) from the getting-started guide. Create the Flux `GitRepository` and the `Component`:

```yaml
# podinfo-git-source.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 5m
  url: https://github.com/stefanprodan/podinfo
  ref:
    branch: master
```

```bash
kubectl apply -f podinfo-git-source.yaml
```

```yaml
# podinfo-kustomize.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: podinfo
  namespace: default
spec:
  sourceRef:
    fluxGitRepository:
      name: podinfo
  path: kustomize
```

```bash
kubectl apply -f podinfo-kustomize.yaml
```

Watch the component status:

```bash
kubectl get component podinfo -w
```

The component will transition to `Error` state. Because the `deployer` service account has no RBAC permissions, component-operator is refused when it attempts to create any of the dependent objects (Deployments, Services, etc.) on its behalf.

Inspect the error message:

```bash
kubectl describe component podinfo
```

You should see something along the lines of:

```
Message: ... is forbidden: User "system:serviceaccount:default:deployer" cannot ...
```

This confirms that the operator is correctly impersonating `deployer` rather than its own privileged account.

## 4. Grant the service account namespace-scoped permissions

Bind the `cluster-admin` ClusterRole to `deployer` with a **RoleBinding** (not a ClusterRoleBinding). This grants full control inside the `default` namespace but nothing outside it:

```bash
kubectl create rolebinding deployer \
  --serviceaccount default:deployer \
  --clusterrole cluster-admin \
  -n default
```

> **Note:** Using `cluster-admin` as the ClusterRole in a RoleBinding is a convenient shorthand for "full access within this namespace". In production, you would typically bind a more narrowly scoped role that covers only the resource types the component actually needs to create.

## 5. Observe the component becoming ready

component-operator retries reconciliation automatically. After the binding is in place, the next retry will succeed. Watch the component converge:

```bash
kubectl get component podinfo -w
```

The component transitions through `Processing` and reaches `Ready`. All podinfo pods are now running:

```bash
kubectl get pods -l app=podinfo
```

The entire deployment was performed under the `deployer` identity — a service account that has admin rights only inside `default` — even though the operator pod itself holds cluster-wide privileges.

## 6. Cleanup

Delete the podinfo resources:

```bash
kubectl delete component podinfo
kubectl delete gitrepository podinfo
```

Remove the service account and its binding:

```bash
kubectl delete rolebinding deployer -n default
kubectl delete serviceaccount deployer -n default
```

Restore the default component-operator configuration (no default service account):

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system --set options.defaultServiceAccount=
```
