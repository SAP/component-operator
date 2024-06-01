/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package decrypt

type Decryptor interface {
	Decrypt(input []byte, path string) ([]byte, error)
}
