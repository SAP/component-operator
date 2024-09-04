[![REUSE status](https://api.reuse.software/badge/github.com/SAP/component-operator)](https://api.reuse.software/info/github.com/SAP/component-operator)

# component-operator

## About this project

The Kubernetes operator provided by this repository allows to manage the lifecycle of components in Kubernetes clusters.
Here, components are understood as sets of Kubernetes resources (such as deployments, services, RBAC entities, ...), and components are described as instances of the custom resource type `components.cs.core.sap.com` (kind `Component`), which will be reconciled by component-operator. The `Component` type allows to specify
- the source of the manifests describing the dependent resources
- parameterization of these manifests, in case the manifests use go templates, such as Helm charts
- dependencies to other component instances.

Thus, components are similar to flux kustomizations, but have some important advantages:
- They use the [component-operator-runtime](https://github.com/sap/component-operator-runtime) framework to render and deploy dependent objects; therefore the manifest source can be any input that is understood by component-operator-runtime's [KustomizeGenerator](https://sap.github.io/component-operator-runtime/docs/generators/kustomize/) or [HelmGenerator](https://sap.github.io/component-operator-runtime/docs/generators/helm/). In particular, go-templatized kustomizations are allowed; check the [component-operator-runtime documentation](https://sap.github.io/component-operator-runtime/docs) for details.
- It is possible to pin components to a specific source revision through the `spec.revision` field.
- Dependencies (as specified in `spec.dependencies`) are not only honored at creation/update, but also on deletion.

A sample component could look like this:

```yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  namespace: comp-ns
  name: comp-name
spec:
  namespace: target-ns
  name: target-name
  kubeConfig:
    secretRef:
      name: kubeconfig
  requeueInterval: 10m
  retryInterval: 3m
  timeout: 5m
  sourceRef:
    fluxGitRepository:
      namespace: source-ns
      name: source-name
  revision: main/4b14dbc37ca976be75a7508bb41fb99d4a36ab9
  path: ./deploy
  values:
    someKey: someValue
  valuesFrom:
  - name: values-secret
  decryption:
    provider: sops
    secretRef:
      name: sops-secret
  postBuild:
    substitute:
      VAR: VALUE
    substituteFrom:
    - name: subst-secret
  dependencies:
  - namespace: other-namespace
    name: other-component
  - name: another-component
```

### Target namespace and name

The optional fields `spec.namespace` and `spec.name` may be provided to customize the deployment namespace/name of the component (as defined by component-operator-runtime). If not set, the target namespace and name will equal `metadata.namespace` and `metadata.name` of the component.

### KubeConfig reference

By default, the dependent objects will be deployed in the same cluster where the component resides. A kubeconfig may be provided to deploy into a remote cluster,
by specifying a secret (in the component's namespace) as `spec.kubeConfig.secretRef.name`. By default, the secret will be looked up at one of the following keys: `value`, `value.yaml`, `value.yml` (in this order). A custom key can be specified as `spec.kubeConfig.secretRef.key`. Note that the used kubeconfig must not use any kubectl plugins, such as kubelogin.

### Reqeue interval, retry interval and timeout

The field `spec.requeueInterval` defines the period, after which a component is re-reconciled after a previously successful reconcilation.
It is optional, the default is 10 minutes.

The field `spec.retryInterval` defines the period, after which a component is re-reconciled if a retriable error occurs.
Check the [component-operator-runtime documentation](https://sap.github.io/component-operator-runtime/docs/concepts/reconciler/#tuning-the-retry-behavior) for more details about retriable errors. This field is optional; if unset the retry interval equals the effective requeue interval.

Finally, the field `spec.timeout` defines how long dependent objects are expected to take to become ready. If not all depenents are ready, then the component state is `Processing` until the timeout has elapsed; afterwards, the component state flips to `Error`. Note that the operator still tries to reconcile the dependent objects in that case, just as before. The timeout restarts counting down whenever the component itself, or the rendered dependent manifests change.
The timeout field is optional; if unset, it is defaulted with the effective requeue interval.

### Source reference

The mandatory field `spec.sourceRef` defines the source of the manifests used for generation of the dependent objects.
Currently, the following types of sources are supported (exactly one must be present):

```yaml
# Flux GitRepository
sourceRef:
  fluxGitRepository:
    # namespace: source-ns
    name: gitrepo-name

# Flux OciRepository
sourceRef:
  fluxOciRepository:
    # namespace: source-ns
    name: ocirepo-name

# Flux Bucket
sourceRef:
  fluxBucket:
    # namespace: source-ns
    name: bucket-name

# Flux HelmChart
sourceRef:
  fluxHelmChart:
    # namespace: source-ns
    name: helmchart-name
```

Cross-namespace references are allowed; if namespace is not provided, the source will be assumed to exist in the component's namespace.

### Source revision

It is possible to pin a `Component` resource to a specific revision of the source artifact by setting `spec.revision`.
Pinning means that the component will remain in a `Pending` state, until the used source object's revision matches the the value
specified in `spec.revision`. More precisely, in case of flux source resources, the field refers to `status.artifact.revision` of the flux object.
It should be noted that a revision mismatch in the above sense never blocks the deletion of the component.

### Source path

By default, the manifests (as kustomization or helm chart) are assumed to be located at the root folder of the specified source artifact.
A subfolder can be specified by setting `spec.path`.

### Values and ValuesFrom

Parameters for the referenced helm chart or kustomization can be provided either inline, by `spec.Values` or as a secret key references, listed
in `spec.ValuesFrom`, such as

```yaml
valuesFrom:
- name: values-secret
  # key: valuesKey
```

The `key` attribute is optional; if missing the following default keys will be tried: `values`, `values.yaml`, `values.yml` (in this order),
until one is found in the referenced secret. If multiple secret keys are referenced in `spec.valuesFrom`, their contents will be deeply merged onto each other in
the order of appearance. If in addition `spec.values` is defined, its content will be merged onto the merged values from `spec.valuesFrom`.

The aggregated values object obtained this way will then be passed as parameters to the according `KustomizeGenerator` or `HelmGenerator`, depending
on whether the source artifacts contains a kustomization or a helm chart.

### Decryption

Parts of the given manifests may be encrypted. Currently, only [SOPS](https://github.com/getsops/sops) is supported as encryption provider.
So `spec.decryption.provider` must be set to the value `sops`. In that case, a secret reference must be provided, which follows the exact same
[logic as used by flux](https://fluxcd.io/flux/guides/mozilla-sops/). With the restriction that only GPG and Age are supported as encryption engines.

### Post-build variable substitution

The rendered manifest may contain bash-style variable references, as defined by the [envsubst](https://github.com/drone/envsubst) Go package.
The replacements may be defined either inline in `spec.postBuild.substitute` as `KEY: VALUE` pairs, or loaded by secret references, where
the keys of the secrets will be interpreted as variable names (and therefore have to be valid bash variable names). If multiple secrets, and
maybe inline substitutions are provided, they will be merged in the usual order (secrets in order of appearance, and then inline content). 

### Dependencies

As with flux kustomizations, it is possible to declare dependencies between `Component` objects, that means, to list other components,
even cross-namespace, in `spec.dependencies`. Providing no namespace means that the referenced component is assumed to be in the same
namespace as the depending component. During creation or update, a component will remain in status `Pending` until all its dependencies
are in a `Ready` state. Other than in the flux case, where dependencies are only evaluated during creation/update, but not for
deletion, component dependencies are honored (in reverse order) also during deletion. That means, if a component which is listed as dependency of one or more other components, is being deleted, then it will go into a `DeletionPending` state, until all these depending components are gone.

## Requirements and Setup

The operator relies on the flux source controller, which can be installed by running

```bash
flux install --components source-controller
```

The recommended deployment method for the operator itself is to use the [Helm chart](https://github.com/SAP/component-operator/tree/main/chart) included in this repository:

```bash
helm upgrade -i component-operator oci://ghcr.io/sap/component-operator/charts/component-operator
```

## Documentation

The API reference can be found here: [https://pkg.go.dev/github.com/sap/component-operator](https://pkg.go.dev/github.com/sap/component-operator).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/component-operator/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/SAP/component-operator/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and component-operator contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/component-operator).
