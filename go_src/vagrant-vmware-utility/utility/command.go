// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// Runs the given command and returns the exit code.
func Execute(cmd *exec.Cmd) int {
	exitCode := 1
	if err := cmd.Start(); err != nil {
		return exitCode
	}
	exitCode = 0
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}
	if exitCode == 0 && !cmd.ProcessState.Success() {
		exitCode = 1
	}
	return exitCode
}

// Runs the given command and returns combined stdout/stderr output
func ExecuteWithOutput(cmd *exec.Cmd) (exitCode int, output string) {
	buf, err := cmd.CombinedOutput()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			output = fmt.Sprintf("Error running %q: %s", cmd.Path, err)
			return 1, output
		}
	}
	output = string(buf)
	return exitCode, output
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		log.Printf("FileExists os.Stat failed for %q: %s", path, err)
	}
	return err == nil
}
