// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

func IsBigSurMin() bool {
	m, err := GetDarwinMajor()
	if err != nil {
		return false
	}
	return m >= 20
}
