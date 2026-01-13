// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import "errors"

func GetDarwinMajor() (m int, err error) {
	err = errors.New("Platform is not darwin")
	return
}
