// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"strings"
	"testing"
)

func TestBuildServiceBinaryPathName(t *testing.T) {
	tests := []struct {
		name       string
		exePath    string
		configPath string
		logPath    string
	}{
		{
			name:       "path with spaces",
			exePath:    `C:\Program Files\VagrantVMwareUtility\bin\vagrant-vmware-utility.exe`,
			configPath: `C:\ProgramData\HashiCorp\vagrant-vmware-utility\config\service.hcl`,
			logPath:    `C:\ProgramData\HashiCorp\vagrant-vmware-utility\logs\utility.log`,
		},
		{
			name:       "path without spaces",
			exePath:    `C:\HashiCorp\vagrant-vmware-utility.exe`,
			configPath: `C:\HashiCorp\service.hcl`,
			logPath:    `C:\HashiCorp\utility.log`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildServiceBinaryPathName(tt.exePath, tt.configPath, tt.logPath)

			if !strings.HasPrefix(result, `"`) {
				t.Errorf("BinaryPathName must start with a double quote to prevent unquoted path LPE, got: %s", result)
			}

			expectedPrefix := `"` + tt.exePath + `"`
			if !strings.HasPrefix(result, expectedPrefix) {
				t.Errorf("exePath must be enclosed in double quotes, want prefix %q, got: %s", expectedPrefix, result)
			}

			if !strings.Contains(result, `service run`) {
				t.Errorf("BinaryPathName must contain service subcommand, got: %s", result)
			}

			expectedConfigArg := `-config-file="` + tt.configPath + `"`
			if !strings.Contains(result, expectedConfigArg) {
				t.Errorf("BinaryPathName must contain quoted config-file arg %q, got: %s", expectedConfigArg, result)
			}

			expectedLogArg := `-log-file="` + tt.logPath + `"`
			if !strings.Contains(result, expectedLogArg) {
				t.Errorf("BinaryPathName must contain quoted log-file arg %q, got: %s", expectedLogArg, result)
			}
		})
	}
}
