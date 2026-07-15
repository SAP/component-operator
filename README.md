[![REUSE status](https://api.reuse.software/badge/github.com/SAP/component-operator)](https://api.reuse.software/info/github.com/SAP/component-operator)

# component-operator

## About this project

The Kubernetes operator provided by this repository allows to manage the lifecycle of components in Kubernetes clusters.
Here, components are understood as sets of Kubernetes resources (such as deployments, services, RBAC entities, ...). Components are described as instances of the custom resource type `components.cs.core.sap.com` (kind `Component`), which are reconciled by component-operator.

The `Component` type allows to specify
- the source of the manifests describing the dependent resources
- parameterization of these manifests, in case the manifests use go templates, such as Helm charts
- deployment instructions
- dependencies to other component instances.

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
  serviceAccountName: some-sa
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
  revision: main@sha1:4b14dbc37ca976be75a7508bb41fb99d4a36ab9
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

Thus, components look similar to [Flux Kustomizations](https://fluxcd.io/flux/components/kustomize/kustomizations/) but offer more features and flexibility. Besides other sources, components are fully integrated with [Flux sources](https://fluxcd.io/flux/components/source/). Thus the operator could be seen as a 'third' Flux deployer (besides kustomize-controller and helm-controller), which can seamlessly replace or interoperate with the Flux deployers.

## Documentation

**Documentation:** [https://sap.github.io/component-operator](https://sap.github.io/component-operator).

**API Reference:** [https://pkg.go.dev/github.com/sap/component-operator](https://pkg.go.dev/github.com/sap/component-operator).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/component-operator/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/SAP/component-operator/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2026 SAP SE or an SAP affiliate company and component-operator contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/component-operator).
