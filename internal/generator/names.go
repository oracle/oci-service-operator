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
	{suffix: "Indices", replacement: "Index"},
	{suffix: "indices", replacement: "index"},
	{suffix: "Statuses", replacement: "Status"},
	{suffix: "statuses", replacement: "status"},
	{suffix: "Status", replacement: "Status", recursive: true, preserveExact: true},
	{suffix: "status", replacement: "status", recursive: true, preserveExact: true},
	{suffix: "Stats", replacement: "Stats", recursive: true, preserveExact: true},
	{suffix: "stats", replacement: "stats", recursive: true, preserveExact: true},
}

var esPluralSuffixes = []string{"sses", "shes", "ches", "xes", "zes"}

var unpluralizedSuffixes = []string{"Metadata", "metadata", "Information", "information"}

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
	case strings.HasSuffix(name, "Index"):
		return strings.TrimSuffix(name, "Index") + "Indices"
	case strings.HasSuffix(name, "index"):
		return strings.TrimSuffix(name, "index") + "indices"
	case hasUnpluralizedSuffix(name):
		return name
	case strings.HasSuffix(name, "Status"), strings.HasSuffix(name, "status"):
		return name + "es"
	case strings.HasSuffix(name, "Stats"), strings.HasSuffix(name, "stats"):
		return name
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return name
	case strings.HasSuffix(name, "y") && len(name) > 1:
		if hasVowelBeforeSuffix(name, 'y') {
			return name + "s"
		}
		return strings.TrimSuffix(name, "y") + "ies"
	case strings.HasSuffix(name, "ss"):
		return name + "es"
	case strings.HasSuffix(name, "x"),
		strings.HasSuffix(name, "z"),
		strings.HasSuffix(name, "ss"),
		strings.HasSuffix(name, "ch"),
		strings.HasSuffix(name, "sh"),
		strings.HasSuffix(name, "ss"):
		return name + "es"
	default:
		return name + "s"
	}
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
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

func applySpecialSingularRules(name string) (string, bool) {
	for _, rule := range specialSingularRules {
		if singular, ok := applySpecialSingularRule(name, rule); ok {
			return singular, true
		}
	}

	return "", false
}

func applySpecialSingularRule(name string, rule singularRule) (string, bool) {
	if !strings.HasSuffix(name, rule.suffix) {
		return "", false
	}

	stem := strings.TrimSuffix(name, rule.suffix)
	if stem == "" && rule.preserveExact {
		return name, true
	}
	if rule.recursive {
		return singularize(stem) + rule.replacement, true
	}

	return stem + rule.replacement, true
}

func singularizeStandardSuffix(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return strings.TrimSuffix(name, "ies") + "y", true
	case hasPluralESSuffix(name):
		return strings.TrimSuffix(name, "es"), true
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return strings.TrimSuffix(name, "s"), true
	default:
		return "", false
	}
}

func hasPluralESSuffix(name string) bool {
	for _, suffix := range esPluralSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	return false
}

func shouldSplitCamelToken(runes []rune, index int) bool {
	if index == 0 {
		return false
	}

	current := runes[index]
	if !unicode.IsUpper(current) {
		return false
	}

	prev := runes[index-1]
	return unicode.IsLower(prev) || unicode.IsDigit(prev) || endsUpperRunBeforeLower(runes, index, prev)
}

func endsUpperRunBeforeLower(runes []rune, index int, prev rune) bool {
	return unicode.IsUpper(prev) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
}

func appendLowerToken(tokens []string, current []rune) []string {
	if len(current) == 0 {
		return tokens
	}

	return append(tokens, strings.ToLower(string(current)))
}

func hasUnpluralizedSuffix(name string) bool {
	for _, suffix := range unpluralizedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	return false
}

func hasVowelBeforeSuffix(name string, suffix byte) bool {
	if len(name) < 2 || name[len(name)-1] != suffix {
		return false
	}

	switch unicode.ToLower(rune(name[len(name)-2])) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}
