package amigo

import (
	"fmt"
	"slices"
	"strings"
)

func validatePathParameters(path string, input InputMetadata) error {
	wildcards, err := pathWildcards(path)
	if err != nil {
		return err
	}

	declared := make(map[string]bool)
	for _, parameter := range input.Parameters {
		if parameter.Source != ParameterPath {
			continue
		}
		if declared[parameter.Name] {
			return fmt.Errorf("path parameter %q is declared more than once", parameter.Name)
		}
		declared[parameter.Name] = true
		if !slices.Contains(wildcards, parameter.Name) {
			return fmt.Errorf("path parameter %q is not declared in the route", parameter.Name)
		}
	}

	for _, wildcard := range wildcards {
		if !declared[wildcard] {
			return fmt.Errorf("route path parameter %q has no matching input field", wildcard)
		}
	}
	return nil
}

func pathWildcards(path string) ([]string, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path must start with a slash")
	}

	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	wildcards := make([]string, 0)
	seen := make(map[string]bool)
	for index, segment := range segments {
		if !strings.ContainsAny(segment, "{}") {
			continue
		}
		if segment == "{$}" {
			if index != len(segments)-1 {
				return nil, fmt.Errorf("end-of-path wildcard {$} must be the final segment")
			}
			continue
		}
		if len(segment) < 3 || segment[0] != '{' || segment[len(segment)-1] != '}' {
			return nil, fmt.Errorf("invalid path wildcard segment %q", segment)
		}

		name := segment[1 : len(segment)-1]
		if strings.HasSuffix(name, "...") {
			name = strings.TrimSuffix(name, "...")
			if index != len(segments)-1 {
				return nil, fmt.Errorf("multi-segment wildcard %q must be the final segment", name)
			}
		}
		if name == "" || strings.ContainsAny(name, "{}") {
			return nil, fmt.Errorf("invalid path wildcard segment %q", segment)
		}
		if seen[name] {
			return nil, fmt.Errorf("route path parameter %q is declared more than once", name)
		}
		seen[name] = true
		wildcards = append(wildcards, name)
	}
	return wildcards, nil
}
