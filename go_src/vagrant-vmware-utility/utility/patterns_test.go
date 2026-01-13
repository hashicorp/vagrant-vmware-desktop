// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"fmt"
	"testing"
)

func TestSimpleMatchPattern(t *testing.T) {
	pattern := "^test.$"
	string := "test!"
	_, err := MatchPattern(pattern, string)
	if err != nil {
		t.Errorf("Failed to match pattern: %s", err)
	}
}

func TestSimpleNamedMatchPattern(t *testing.T) {
	pattern := "^(?P<value>test.)$"
	string := "test!"
	matches, err := MatchPattern(pattern, string)
	if err != nil {
		t.Errorf("Failed to match pattern: %s", err)
	}
	if matches["value"] != string {
		t.Errorf("Matched pattern is not expected string %s != %s",
			matches["value"], string)
	}
}

func TestMultipleNamedMatchPattern(t *testing.T) {
	pattern := "^(?P<key>[^=]+)=(?P<value>.+)$"
	string := "config=true"
	matches, err := MatchPattern(pattern, string)
	if err != nil {
		t.Errorf("Failed to match pattern: %s", err)
	}
	if _, ok := matches["key"]; !ok {
		t.Errorf("Failed to locate key in string")
	}
	if _, ok := matches["value"]; !ok {
		t.Errorf("Failed to locate value in string")
	}
	composed := fmt.Sprintf("%s=%s", matches["key"], matches["value"])
	if string != composed {
		t.Errorf("Matches do not equal origin string %s != %s",
			composed, string)
	}
}
