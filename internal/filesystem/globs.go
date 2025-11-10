package filesystem

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// GlobToRegex converts a glob pattern to a Seatbelt-compatible regex
func GlobToRegex(pattern string) (string, error) {
	// Normalise the pattern first if it's not a glob
	if !ContainsGlob(pattern) {
		normPath, err := NormalisePath(pattern)
		if err != nil {
			return "", err
		}
		pattern = normPath
	}

	var result strings.Builder
	result.WriteString("^")

	i := 0
	for i < len(pattern) {
		ch := pattern[i]

		switch ch {
		case '*':
			// Check for **
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// ** matches anything including / (zero or more path segments)
				i += 2

				// Skip following / if present
				if i < len(pattern) && pattern[i] == '/' {
					i++
				}

				result.WriteString(".*")
				continue
			}

			// Single * matches anything except /
			result.WriteString("[^/]*")
			i++

		case '?':
			// ? matches any single character except /
			result.WriteString("[^/]")
			i++

		case '[':
			// Character class - find the end
			j := i + 1
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}

			if j >= len(pattern) {
				return "", fmt.Errorf("unclosed character class in pattern")
			}

			// Copy the character class as-is
			result.WriteString(pattern[i : j+1])
			i = j + 1

		case '{':
			// Alternation - find the end
			j := i + 1
			depth := 1
			for j < len(pattern) && depth > 0 {
				if pattern[j] == '{' {
					depth++
				} else if pattern[j] == '}' {
					depth--
				}
				j++
			}

			if depth != 0 {
				return "", fmt.Errorf("unclosed brace in pattern")
			}

			// Convert {a,b,c} to (a|b|c)
			inner := pattern[i+1 : j-1]
			parts := strings.Split(inner, ",")
			result.WriteString("(")
			for idx, part := range parts {
				if idx > 0 {
					result.WriteString("|")
				}
				// Recursively convert each part
				converted, err := GlobToRegex(part)
				if err != nil {
					return "", err
				}
				// Strip ^ and $ from the converted part
				converted = strings.TrimPrefix(converted, "^")
				converted = strings.TrimSuffix(converted, "$")
				result.WriteString(converted)
			}
			result.WriteString(")")
			i = j

		case '.', '+', '^', '$', '(', ')', '|', '\\':
			// Escape regex special characters
			result.WriteString("\\")
			result.WriteByte(ch)
			i++

		default:
			result.WriteByte(ch)
			i++
		}
	}

	result.WriteString("$")
	return result.String(), nil
}

// MatchGlob checks if a path matches a glob pattern
func MatchGlob(pattern, path string) (bool, error) {
	regex, err := GlobToRegex(pattern)
	if err != nil {
		return false, err
	}

	matched, err := regexp.MatchString(regex, path)
	if err != nil {
		return false, fmt.Errorf("regex match error: %w", err)
	}

	return matched, nil
}

// ExpandGlob expands a glob pattern to matching file paths
func ExpandGlob(pattern string) ([]string, error) {
	// Use filepath.Glob for expansion
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob expansion failed: %w", err)
	}

	// Normalise all matches
	normalised := make([]string, len(matches))
	for i, match := range matches {
		norm, err := NormalisePath(match)
		if err != nil {
			return nil, err
		}
		normalised[i] = norm
	}

	return normalised, nil
}
