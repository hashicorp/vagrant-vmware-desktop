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
