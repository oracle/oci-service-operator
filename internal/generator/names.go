/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"strings"
	"unicode"
)

type singularRule struct {
	suffix        string
	replacement   string
	recursive     bool
	preserveExact bool
}

var specialSingularRules = []singularRule{
	{suffix: "Statuses", replacement: "Status"},
	{suffix: "statuses", replacement: "status"},
	{suffix: "Status", replacement: "Status", recursive: true, preserveExact: true},
	{suffix: "status", replacement: "status", recursive: true, preserveExact: true},
	{suffix: "Stats", replacement: "Stats", recursive: true, preserveExact: true},
	{suffix: "stats", replacement: "stats", recursive: true, preserveExact: true},
}

var esPluralSuffixes = []string{"sses", "shes", "ches", "xes", "zes"}

func singularize(name string) string {
	if singular, ok := applySpecialSingularRules(name); ok {
		return singular
	}
	if singular, ok := singularizeStandardSuffix(name); ok {
		return singular
	}
	return name
}

func pluralize(name string) string {
	switch {
	case strings.HasSuffix(name, "Status"), strings.HasSuffix(name, "status"):
		return name + "es"
	case strings.HasSuffix(name, "Stats"), strings.HasSuffix(name, "stats"):
		return name
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return name
	case strings.HasSuffix(name, "y") && len(name) > 1:
		return strings.TrimSuffix(name, "y") + "ies"
	case strings.HasSuffix(name, "x"),
		strings.HasSuffix(name, "z"),
		strings.HasSuffix(name, "ch"),
		strings.HasSuffix(name, "sh"):
		return name + "es"
	default:
		return name + "s"
	}
}

func fileStem(name string) string {
	return strings.ToLower(name)
}

func lowerCamel(name string) string {
	tokens := splitCamel(name)
	if len(tokens) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(tokens[0])
	for _, token := range tokens[1:] {
		builder.WriteString(strings.ToUpper(token[:1]))
		builder.WriteString(token[1:])
	}

	return builder.String()
}

func splitCamel(name string) []string {
	if strings.TrimSpace(name) == "" {
		return nil
	}

	var tokens []string
	var current []rune
	runes := []rune(name)
	for i, r := range runes {
		if shouldSplitCamelToken(runes, i) {
			tokens = appendLowerToken(tokens, current)
			current = current[:0]
		}
		current = append(current, r)
	}

	return appendLowerToken(tokens, current)
}

func normalizedTokens(name string) []string {
	tokens := splitCamel(name)
	for i, token := range tokens {
		tokens[i] = singularize(token)
	}
	return tokens
}
