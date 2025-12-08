// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"errors"
	"regexp"
)

func MatchPattern(regexpPattern string, content string) (map[string]string, error) {
	matches := map[string]string{}
	pattern, err := regexp.Compile(regexpPattern)
	if err != nil {
		return matches, err
	}
	patternNames := pattern.SubexpNames()
	match := pattern.FindStringSubmatch(content)
	if match == nil {
		return matches, errors.New("Failed to match pattern within content")
	}
	for i, name := range patternNames {
		if i == 0 {
			continue
		}
		matches[name] = match[i]
	}
	return matches, nil
}

func NamePatternResults(matches, names []string) (result map[string]string) {
	result = map[string]string{}
	for i, name := range names {
		if i == 0 || i >= len(matches) {
			continue
		}
		result[name] = matches[i]
	}
	return result
}
