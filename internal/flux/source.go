/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package flux

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
)

type Source interface {
	client.Object
	fluxsourcev1.Source
	GetConditions() []metav1.Condition
}

func IsSourceReady(source Source) bool {
	for _, condition := range source.GetConditions() {
		if condition.Type != "Ready" {
			continue
		}
		if condition.ObservedGeneration != source.GetGeneration() {
			return false
		}
		return condition.Status == metav1.ConditionTrue
	}
	return false
}
