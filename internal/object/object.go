/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package object

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	client.Object
	GetConditions() []metav1.Condition
}

func IsReady(object Object) bool {
	for _, condition := range object.GetConditions() {
		if condition.Type != "Ready" {
			continue
		}
		if condition.ObservedGeneration != object.GetGeneration() {
			return false
		}
		return condition.Status == metav1.ConditionTrue
	}
	return false
}
