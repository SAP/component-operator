---
title: "Decryption using SOPS"
linkTitle: "Decryption using SOPS"
weight: 11
description: >
  Using SOPS to decrypt encrypted manifests with age
---

This tutorial demonstrates how to work with source manifests that are encrypted using [SOPS](https://github.com/getsops/sops). Component-operator currently supports two encryption engines: GPG and [age](https://github.com/filosottile/age). In this example, we'll use age. The component-operator implementation works analogous to how [Flux handles SOPS decryption](https://fluxcd.io/flux/guides/mozilla-sops/).

For more details on decryption configuration, see [Sources](../../usage/sources#decryption).

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide.

## 1. Install age

If you don't have age installed yet, follow the [installation instructions](https://github.com/filosottile/age#installation). For example, on macOS:

```bash
brew install age
```

## 2. Create a private/public keypair

Generate an age keypair:

```bash
age-keygen -o sops.agekey
```

The output will display your public key. **Note down the public key** (starting with `age1...`) — you'll need it in the next steps.

Example output:

```
Public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

## 3. Create a Kubernetes Secret

Create a secret containing the private key:

```bash
kubectl create secret generic sops --from-file=sops.agekey=sops.agekey
```

This secret will be used by component-operator to decrypt the manifests at runtime.

## 4. Prepare unencrypted manifests

Create two simple secret manifests that we'll encrypt:

**secret-1.yaml:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sops-demo-1
stringData:
  foo: bar-1
```

**secret-2.yaml:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sops-demo-2
stringData:
  foo: bar-2
```

## 5. Encrypt the manifests with SOPS

Now encrypt both files using SOPS with your age public key.

**Partial encryption** (encrypt only `data` and `stringData` fields):

```bash
sops --age=<your-age-public-key> --encrypt --encrypted-regex '^(data|stringData)$' secret-1.yaml > secret-1.enc.yaml
```

**Full encryption** (encrypt the entire YAML structure):

```bash
sops --age=<your-age-public-key> --encrypt --input-type binary secret-2.yaml > secret-2.enc.yaml
```

Replace `<your-age-public-key>` with the public key from step 2 (e.g., `age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p`).

The first approach uses SOPS' ability to partially encrypt certain fields in structured data, while the second encrypts everything. Choose the approach that best fits your security requirements.

> **Note:** It is crucial to have no more than one Kubernetes object per file, as SOPS does not handle YAML streams (multiple documents separated by `---`).

## 6. Create a Blueprint with encrypted content

Create a Blueprint that embeds the encrypted manifests:

```yaml
# sops-demo-blueprint.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: sops-demo
  namespace: default
spec:
  files:
    secret-1.yaml: |
      # Paste the entire content of secret-1.enc.yaml here
    
    secret-2.yaml: |
      # Paste the entire content of secret-2.enc.yaml here
```

Copy the full encrypted content from `secret-1.enc.yaml` and `secret-2.enc.yaml` into the respective file entries in the Blueprint spec.

Apply the Blueprint:

```bash
kubectl apply -f sops-demo-blueprint.yaml
```

## 7. Create a Component with decryption enabled

Create a Component that references the Blueprint and enables SOPS decryption:

```yaml
# sops-demo-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: sops-demo
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: sops-demo
  decryption:
    provider: sops
    secretRef:
      name: sops
```

Apply the Component:

```bash
kubectl apply -f sops-demo-component.yaml
```

## 8. Verify the decrypted secrets

Check the component status:

```bash
kubectl get component sops-demo
```

The component should be in `Ready` state, confirming that the encrypted manifests were successfully decrypted and applied.

Component-operator has decrypted the manifests using the age key from the `sops` secret and has created the Kubernetes secrets:

```bash
kubectl get secrets sops-demo-1 sops-demo-2
```

You can verify that the secrets were properly decrypted:

```bash
kubectl get secret sops-demo-1 -o jsonpath='{.data.foo}' | base64 -d
# Output: bar-1

kubectl get secret sops-demo-2 -o jsonpath='{.data.foo}' | base64 -d
# Output: bar-2
```

## 9. Cleanup

Delete the component and its dependent secrets:

```bash
kubectl delete component sops-demo
```

The component deletion removes all dependent objects (the two secrets). The Blueprint and the SOPS key secret are not managed by the component and must be removed separately if desired:

```bash
kubectl delete blueprint sops-demo
kubectl delete secret sops
```