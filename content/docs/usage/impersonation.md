---
title: "Impersonation and Remote Clusters"
linkTitle: "Impersonation and Remote Clusters"
weight: 13
description: >
  Deploying dependent objects as a specific service account or to a remote cluster
---

By default, component-operator uses its own controller kubeconfig to interact with the Kubernetes API when reconciling a `Component`. Both impersonation and remote cluster deployment allow you to override this default.

## Service Account Impersonation

If `spec.serviceAccountName` is set, the controller impersonates that service account (relative to the component's namespace) when applying and deleting dependent objects. This allows fine-grained access control: the service account determines what the component is permitted to create, modify, and delete.

```yaml
spec:
  serviceAccountName: my-deploy-sa
```

If `spec.serviceAccountName` is not set, but the controller was started with the `--default-service-account` command-line flag, the specified service account is used as the default impersonation for all components that do not explicitly configure one. This is useful for enforcing a least-privilege default across all components in a namespace without having to annotate each one.

## Deploying to a Remote Cluster

By default, dependent objects are deployed in the same cluster where the `Component` object resides. To deploy into a remote cluster, provide a kubeconfig via `spec.kubeConfig.secretRef`:

```yaml
spec:
  kubeConfig:
    secretRef:
      name: remote-cluster-kubeconfig
      # key: value  # optional; defaults to trying 'value', 'value.yaml', 'value.yml'
```

The referenced Secret must exist in the component's namespace. The kubeconfig must not use any `kubectl` plugins (such as `kubelogin`), as these are not supported. 

Note that, when using a kubeconfig pointing to a remote cluster, then it is probably necessary to set `spec.namespace`, or to ensure that all source manifests specify the target namespace explicitly. Otherwise, the namespace would be defaulted by the component's own namespace, which is probably not what is intended.

## The `localLookup` Functions in Kustomize Templates

When a `kubeConfig` is configured, the regular `lookup` and `mustLookup` template functions query the **remote** (target) cluster. In some cases you need to look up a resource in the **local** cluster — for example, to read a Secret from the same cluster where the controller runs and pass its content to the remote deployment.

For this purpose, the Kustomize generator provides the `localLookup` and `mustLocalLookup` functions, which always query the local cluster regardless of whether a `kubeConfig` is set:

```yaml
# Example: read a certificate from the local cluster, pass it to the remote deployment
{{- $cert := localLookup "v1" "Secret" "cert-manager" "my-tls-cert" }}
apiVersion: v1
kind: Secret
metadata:
  name: tls-cert
type: kubernetes.io/tls
data:
  tls.crt: {{ index $cert.data "tls.crt" }}
  tls.key: {{ index $cert.data "tls.key" }}
```