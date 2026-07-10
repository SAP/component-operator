/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"context"
	"encoding/json"

	"github.com/sap/go-generics/maps"

	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

type Generator struct {
	factory *Factory
}

var _ manifests.Generator = &Generator{}

func NewGenerator(clnt client.Client) (*Generator, error) {
	return &Generator{
		factory: newFactory(clnt),
	}, nil
}

func (g *Generator) Generate(ctx context.Context, namespace string, name string, parameters componentoperatorruntimetypes.Unstructurable) ([]client.Object, error) {
	reconcilerName, err := component.ReconcilerNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	spec := parameters.(*operatorv1alpha1.ComponentSpec)

	url := spec.SourceRef.Artifact().Url
	digest := spec.SourceRef.Artifact().Digest
	path := spec.Path

	var decryptionProvider string
	var decryptionKeys map[string][]byte
	if spec.Decryption != nil {
		decryptionProvider = spec.Decryption.Provider
		decryptionKeys = spec.Decryption.SecretRef.Data()
	}

	generator, err := g.factory.GetGenerator(url, path, digest, decryptionProvider, decryptionKeys)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	for _, ref := range spec.ValuesFrom {
		var v map[string]any
		if err := kyaml.Unmarshal(ref.Value(), &v); err != nil {
			return nil, err
		}
		deepMerge(values, v)
	}
	if spec.Values != nil {
		var v map[string]any
		if err := json.Unmarshal(spec.Values.Raw, &v); err != nil {
			return nil, err
		}
		deepMerge(values, v)
	}

	if spec.PostBuild != nil {
		transformableGenerator := manifests.NewGenerator(generator)
		if len(spec.PostBuild.Substitute) > 0 || len(spec.PostBuild.SubstituteFrom) > 0 {
			substitutions := make(map[string]string)
			for _, ref := range spec.PostBuild.SubstituteFrom {
				shallowMerge(substitutions, maps.Collect(ref.Data(), func(x []byte) string { return string(x) }))
			}
			shallowMerge(substitutions, spec.PostBuild.Substitute)
			transformer, err := manifests.NewSubstitutionObjectTransformer(substitutions, componentoperatorruntimetypes.SelectorFunc[client.Object](func(object client.Object) bool {
				return object.GetAnnotations()[reconcilerName+"/disableSubstitution"] != "true"
			}))
			if err != nil {
				return nil, err
			}
			transformableGenerator.WithObjectTransformer(transformer)
		}
		if len(spec.PostBuild.Patches) > 0 || len(spec.PostBuild.Images) > 0 {
			transformer, err := manifests.NewKustomizeObjectTransformer(spec.PostBuild.Patches, spec.PostBuild.Images)
			if err != nil {
				return nil, err
			}
			transformableGenerator.WithObjectTransformer(transformer)
		}
		generator = transformableGenerator
	}

	objects, err := generator.Generate(ctx, namespace, name, componentoperatorruntimetypes.UnstructurableMap(values))
	if err != nil {
		return nil, err
	}

	return objects, nil
}
