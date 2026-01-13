// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"sync"
)

var L = &sync.Mutex{}
var Shutdown = sync.NewCond(L)
var ShutdownTasks = []func(){}

func RegisterShutdownTask(f func()) {
	L.Lock()
	defer L.Unlock()
	ShutdownTasks = append(ShutdownTasks, f)
}

func RunShutdownTasks() {
	for _, f := range ShutdownTasks {
		f()
	}
}
