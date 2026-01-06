/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

// TODO: consolidate all the util files into an internal reuse package

func ref[T any](x T) *T {
	return &x
}
