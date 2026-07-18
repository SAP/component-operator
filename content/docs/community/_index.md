---
title: "Community"
linkTitle: "Community"
weight: 7
description: >
  How to get support and how to contribute to Component Operator
---

The component-operator project is hosted on GitHub at [github.com/SAP/component-operator](https://github.com/SAP/component-operator). Contributions of all kinds are welcome.

## Reporting Issues

If you encounter a bug, have a feature request, or want to suggest an improvement, please [open an issue](https://github.com/SAP/component-operator/issues/new/choose) on GitHub.

When reporting a bug, include:

- A clear description of the problem and the expected behaviour.
- The version of component-operator you are running (`kubectl get deployment -n flux-system component-operator -o jsonpath='{.spec.template.spec.containers[0].image}'`).
- Relevant `Component` manifests and `kubectl describe component <name>` output.
- Any error messages from the controller logs (`kubectl logs -n flux-system deployment/component-operator`).

## Contributing Code

Contributions are made through [pull requests](https://github.com/SAP/component-operator/pulls) on GitHub.

1. **Fork** the repository and create a feature branch from `main`.
2. **Make your changes**, following the existing code style. Run the test suite locally before pushing.
3. **Open a pull request** against `main`. Describe what the change does and link any related issues.
4. A maintainer will review the pull request. Please respond promptly to review comments.

For significant changes (new features, breaking changes, large refactors), it is a good idea to open an issue first to discuss the approach before investing time in a full implementation.

## Code of Conduct

This project follows the [SAP Open Source Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md). Please be respectful and constructive in all interactions.
