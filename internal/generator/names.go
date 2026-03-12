/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"slices"
	"strings"
	"unicode"
)

func singularize(name string) string {
	switch {
	case strings.HasSuffix(name, "Statuses") && len(name) > len("Statuses"):
		return strings.TrimSuffix(name, "Statuses") + "Status"
	case strings.HasSuffix(name, "statuses") && len(name) > len("statuses"):
		return strings.TrimSuffix(name, "statuses") + "status"
	case strings.HasSuffix(name, "Status") && len(name) > len("Status"):
		return singularize(strings.TrimSuffix(name, "Status")) + "Status"
	case strings.HasSuffix(name, "status") && len(name) > len("status"):
		return singularize(strings.TrimSuffix(name, "status")) + "status"
	case name == "Status" || name == "status":
		return name
	case name == "Stats" || name == "stats":
		return name
	case strings.HasSuffix(name, "Stats") && len(name) > len("Stats"):
		return singularize(strings.TrimSuffix(name, "Stats")) + "Stats"
	case strings.HasSuffix(name, "stats") && len(name) > len("stats"):
		return singularize(strings.TrimSuffix(name, "stats")) + "stats"
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "sses"),
		strings.HasSuffix(name, "shes"),
		strings.HasSuffix(name, "ches"),
		strings.HasSuffix(name, "xes"),
		strings.HasSuffix(name, "zes"):
		return strings.TrimSuffix(name, "es")
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return strings.TrimSuffix(name, "s")
	default:
		return name
	}
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
		if i > 0 {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower)) {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = current[:0]
			}
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}

	return tokens
}

func normalizedTokens(name string) []string {
	tokens := splitCamel(name)
	for i, token := range tokens {
		tokens[i] = singularize(token)
	}
	return tokens
}

func compatibilityKind(rawName string, compatibility CompatibilityConfig) (string, bool) {
	resourceTokens := normalizedTokens(rawName)
	for _, existingKind := range compatibility.ExistingKinds {
		existingTokens := normalizedTokens(existingKind)
		if slices.Equal(resourceTokens, existingTokens) {
			return existingKind, true
		}
		if len(existingTokens) >= len(resourceTokens) && slices.Equal(existingTokens[len(existingTokens)-len(resourceTokens):], resourceTokens) {
			return existingKind, true
		}
		if len(resourceTokens) >= len(existingTokens) && slices.Equal(resourceTokens[len(resourceTokens)-len(existingTokens):], existingTokens) {
			return existingKind, true
		}
	}
	return "", false
}
