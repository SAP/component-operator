/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/sap/component-operator/internal/object"
)

type Source interface {
	object.Object
	fluxsourcev1.Source
}
