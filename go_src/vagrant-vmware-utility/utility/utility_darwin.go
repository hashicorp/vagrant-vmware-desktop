// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func GetDarwinVersion() (v string, err error) {
	v, err = unix.Sysctl("kern.osrelease")
	return
}

func GetDarwinMajor() (m int, err error) {
	v, err := GetDarwinVersion()
	p := strings.Split(v, ".")
	if len(p) < 1 {
		return m, fmt.Errorf("Invalid version string encountered - %s", v)
	}
	m, err = strconv.Atoi(p[0])
	return
}
